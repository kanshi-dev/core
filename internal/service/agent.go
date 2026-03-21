package service

import (
	"context"
	"errors"
	"time"

	"github.com/kanshi-dev/core/internal/db"
)

var ErrNoDatabase = errors.New("database connection not established")

type AgentStatus struct {
	AgentID     string    `json:"agentId"`
	HostName    string    `json:"hostName"`
	Os          string    `json:"os"`
	Platform    string    `json:"platform"`
	Arch        string    `json:"arch"`
	CpuCores    int32     `json:"cpuCores"`
	TotalMemory int64     `json:"totalMemory"`
	Version     string    `json:"version"`
	LastSeen    time.Time `json:"lastSeen"`
	Status      string    `json:"status"`
	DiskSize    int64     `json:"diskSize"`
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

	if s.queries == nil {
		return nil, ErrNoDatabase
	}

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
			AgentID:     a.AgentId,
			HostName:    a.HostName,
			Os:          a.Os,
			Platform:    a.Platform,
			Arch:        a.Arch,
			CpuCores:    a.CpuCores,
			TotalMemory: a.TotalMemory,
			Version:     a.Version,
			LastSeen:    a.LastSeen.Time,
			Status:      status,
			DiskSize:    a.DiskSize,
		})
	}

	return result, nil
}
