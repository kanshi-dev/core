package handlers

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kanshi-dev/core/internal/api/v1/response"
	"github.com/kanshi-dev/core/internal/db"
)

func GetMetrics(queries *db.Queries) fiber.Handler {
	return func(c fiber.Ctx) error {
		agentID := c.Query("agent_id")
		name := c.Query("name")

		if agentID == "" || name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "agent_id and name required",
			})
		}

		now := time.Now()

		rows, err := queries.GetMetricsByTimeRange(
			c.Context(),
			db.GetMetricsByTimeRangeParams{
				AgentID: agentID,
				Name:    name,
				FromTs:  pgtype.Timestamptz{Time: now.Add(-1 * time.Hour), Valid: true},
				ToTs:    pgtype.Timestamptz{Time: now, Valid: true},
			},
		)

		if err != nil {
			return response.CustomResponse(c, fiber.StatusInternalServerError, "failed to get metrics", err.Error())
		}

		if rows == nil {
			rows = []db.GetMetricsByTimeRangeRow{}
		}

		return response.CustomResponse(c, fiber.StatusOK, "ok", rows)
	}
}
