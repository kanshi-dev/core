package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/api/v1/response"
	"github.com/kanshi-dev/core/internal/db"
)

type AgentResponse struct {
	AgentID  string    `json:"agentId"`
	LastSeen time.Time `json:"lastSeen"`
	Status   string    `json:"status"`
}

func GetAgentHeartBeat(queries *db.Queries) fiber.Handler {
	return func(c fiber.Ctx) error {

		agents, err := queries.ListAgents(c.Context())
		if err != nil {
			return response.CustomResponse(
				c,
				fiber.StatusInternalServerError,
				"failed to get agents",
				err.Error(),
			)
		}

		now := time.Now().UTC()
		offlineThreshold := 1 * time.Minute

		var result []AgentResponse

		for _, a := range agents {
			status := "offline"
			if now.Sub(a.LastSeen.Time) <= offlineThreshold {
				status = "online"
			}

			result = append(result, AgentResponse{
				AgentID:  a.AgentID,
				LastSeen: a.LastSeen.Time,
				Status:   status,
			})
		}

		return response.CustomResponse(c, fiber.StatusOK, "success", result)
	}
}
