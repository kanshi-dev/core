package service

import (
	"context"
	"time"

	"github.com/kanshi-dev/core/internal/db"
)

type AgentStatus struct {
	AgentID  string    `json:"agentId"`
	LastSeen time.Time `json:"lastSeen"`
	Status   string    `json:"status"`
}

type AgentsService struct {
	queries *db.Queries
}

func NewAgentsService(q *db.Queries) *AgentsService {
	return &AgentsService{queries: q}
}

func (s *AgentsService) ListAgentsWithStatus(
	ctx context.Context,
	offlineThreshold time.Duration,
) ([]AgentStatus, error) {

	rows, err := s.queries.ListAgents(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	var result []AgentStatus

	for _, a := range rows {

		status := "offline"
		if now.Sub(a.LastSeen.Time) <= offlineThreshold {
			status = "online"
		}

		result = append(result, AgentStatus{
			AgentID:  a.AgentId,
			LastSeen: a.LastSeen.Time,
			Status:   status,
		})
	}

	return result, nil
}
