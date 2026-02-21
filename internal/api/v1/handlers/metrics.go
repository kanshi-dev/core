package handlers

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/kanshi-dev/core/internal/api/v1/response"
	"github.com/kanshi-dev/core/internal/db"
)

// Parse metric params
func parseMetricParams(c fiber.Ctx) (agentID string, name string, err error) {
	agentID = c.Query("agent_id")
	name = c.Query("name")

	if agentID == "" || name == "" {
		err = errors.New("agent_id and name required")
	}
	return
}

// Parse time range params
func parseTimeRange(c fiber.Ctx) (time.Time, time.Time, error) {
	fromParam := c.Query("from")
	toParam := c.Query("to")

	now := time.Now().UTC()

	if fromParam == "" || toParam == "" {
		return now.Add(-1 * time.Hour), now, nil
	}

	fromTime, err := time.Parse(time.RFC3339, fromParam)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid from Time Param (RFC3339)")
	}

	toTime, err := time.Parse(time.RFC3339, toParam)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid to Time Param (RFC3339)")
	}

	// Check if toTime is before fromTime
	if toTime.Before(fromTime) {
		return time.Time{}, time.Time{}, errors.New("to Time must be after from Time")
	}

	//Check if the time range exceeds 1 hour
	if time.Since(fromTime) > time.Hour {
		return time.Time{}, time.Time{}, errors.New("time range exceeds 1 hour")
	}

	return fromTime, toTime, nil
}

func badRequest(c fiber.Ctx, err error) error {
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"code":    fiber.StatusBadRequest,
		"message": "bad request",
		"data":    err.Error(),
	})
}

// GetMetrics
/*
This endpoint returns metrics for a given agent and metric name
*/
func GetMetrics(queries *db.Queries) fiber.Handler {
	return func(c fiber.Ctx) error {

		// Parse params agent and metric name
		agentID, name, err := parseMetricParams(c)
		if err != nil {
			return badRequest(c, err)
		}

		// Parse time range params
		fromTime, toTime, err := parseTimeRange(c)
		if err != nil {
			return badRequest(c, err)
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

		return response.CustomResponse(c, fiber.StatusOK, "success", rows)
	}
}

func GetAggregatedMetrics(queries *db.Queries) fiber.Handler {
	return func(c fiber.Ctx) error {

		agentID, name, err := parseMetricParams(c)
		if err != nil {
			return badRequest(c, err)
		}

		fromTime, toTime, err := parseTimeRange(c)
		if err != nil {
			return badRequest(c, err)
		}

		//Set the default interval to 1 minute
		intervalStr := c.Query("interval", "1m")

		dur, err := time.ParseDuration(intervalStr)
		if err != nil {
			return badRequest(c, errors.New("invalid interval format (use 30s, 1m, 5m, etc)"))
		}

		interval := pgtype.Interval{
			Microseconds: dur.Microseconds(),
			Valid:        true,
		}

		rows, err := queries.GetAggregatedMetrics(
			c.Context(),
			db.GetAggregatedMetricsParams{
				AgentID:  agentID,
				Name:     name,
				Interval: interval,
				FromTs:   pgtype.Timestamptz{Time: fromTime, Valid: true},
				ToTs:     pgtype.Timestamptz{Time: toTime, Valid: true},
			},
		)

		if err != nil {
			return response.CustomResponse(
				c,
				fiber.StatusInternalServerError,
				"failed to get aggregated metrics",
				err.Error(),
			)
		}

		return response.CustomResponse(c, fiber.StatusOK, "success", rows)
	}
}
