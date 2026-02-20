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

		//Query Params
		agentID := c.Query("agent_id")
		name := c.Query("name")

		fromParam := c.Query("from")
		toParam := c.Query("to")

		//Time Param Validation
		var fromTime time.Time
		var toTime time.Time
		var err error

		now := time.Now()

		if agentID == "" || name == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "agent_id and name required",
			})
		}

		// Validate time params
		if fromParam == "" || toParam == "" {
			fromTime = now.Add(-1 * time.Hour)
			toTime = now
		} else {

			fromTime, err = time.Parse(time.RFC3339, fromParam)
			if err != nil {
				return response.CustomResponse(c, fiber.StatusBadRequest, "invalid from Time Param (RFC3339)", nil)
			}

			toTime, err = time.Parse(time.RFC3339, toParam)
			if err != nil {
				return response.CustomResponse(c, fiber.StatusBadRequest, "invalid to Time Param (RFC3339)", nil)
			}

			//Check if fromTime is before toTime
			if toTime.Before(fromTime) {
				return response.CustomResponse(c, fiber.StatusBadRequest, "to Time must be after from Time", nil)
			}

			//Check if the time range exceeds 1 hour
			if time.Since(fromTime) > time.Hour {
				return response.CustomResponse(c, fiber.StatusBadRequest, "time range exceeds 1 hour", nil)
			}

		}

		rows, err := queries.GetMetricsByTimeRange(
			c.Context(),
			db.GetMetricsByTimeRangeParams{
				AgentID: agentID,
				Name:    name,
				FromTs:  pgtype.Timestamptz{Time: fromTime, Valid: true},
				ToTs:    pgtype.Timestamptz{Time: toTime, Valid: true},
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
