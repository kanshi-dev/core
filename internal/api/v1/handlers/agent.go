package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/api/v1/response"
	"github.com/kanshi-dev/core/internal/service"
)

type AgentResponse struct {
	AgentID  string    `json:"agentId"`
	LastSeen time.Time `json:"lastSeen"`
	Status   string    `json:"status"`
}

func GetAgentHeartBeat(svc *service.AgentsService) fiber.Handler {
	return func(c fiber.Ctx) error {

		agents, err := svc.ListAgentsWithStatus(c.Context(), 30*time.Second)
		if err != nil {
			return response.CustomResponse(
				c,
				fiber.StatusInternalServerError,
				"failed to get agents",
				err.Error(),
			)
		}

		return response.CustomResponse(c, fiber.StatusOK, "success", agents)
	}
}
