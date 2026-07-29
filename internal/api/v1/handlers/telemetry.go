package handlers

import (
	"encoding/hex"
	"errors"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/api/v1/response"
	"github.com/kanshi-dev/core/internal/service"
)

const maxTelemetryTimeRange = 24 * time.Hour

func ListServices(svc *service.TelemetryService) fiber.Handler {
	return func(c fiber.Ctx) error {
		from, to, err := parseTelemetryRange(c)
		if err != nil {
			return badRequest(c, err)
		}
		limit, err := parseLimit(c, 100)
		if err != nil {
			return badRequest(c, err)
		}
		services, err := svc.ListServices(c.Context(), from, to, limit)
		if err != nil {
			return response.CustomResponse(c, fiber.StatusInternalServerError, "failed to list services", err.Error())
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", services)
	}
}

func SearchTraces(svc *service.TelemetryService) fiber.Handler {
	return func(c fiber.Ctx) error {
		from, to, err := parseTelemetryRange(c)
		if err != nil {
			return badRequest(c, err)
		}
		limit, err := parseLimit(c, 100)
		if err != nil {
			return badRequest(c, err)
		}
		statusCode := int16(-1)
		switch c.Query("status") {
		case "":
		case "ok":
			statusCode = 1
		case "error":
			statusCode = 2
		default:
			return badRequest(c, errors.New("status must be ok or error"))
		}
		minDuration := 0.0
		if raw := c.Query("minDurationMs"); raw != "" {
			minDuration, err = strconv.ParseFloat(raw, 64)
			if err != nil || minDuration < 0 {
				return badRequest(c, errors.New("minDurationMs must be a non-negative number"))
			}
		}
		traceID := c.Query("traceId")
		if traceID != "" && !validHexID(traceID, 16) {
			return badRequest(c, errors.New("traceId must be 32 hexadecimal characters"))
		}
		traces, err := svc.SearchTraces(c.Context(), service.TraceSearch{
			From: from, To: to, ServiceName: c.Query("service"), StatusCode: statusCode,
			MinDuration: minDuration, TraceID: traceID, Limit: limit,
		})
		if err != nil {
			return response.CustomResponse(c, fiber.StatusInternalServerError, "failed to search traces", err.Error())
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", traces)
	}
}

func GetTrace(svc *service.TelemetryService) fiber.Handler {
	return func(c fiber.Ctx) error {
		traceID := c.Params("traceId")
		if !validHexID(traceID, 16) {
			return badRequest(c, errors.New("traceId must be 32 hexadecimal characters"))
		}
		spans, err := svc.GetTrace(c.Context(), traceID)
		if err != nil {
			return response.CustomResponse(c, fiber.StatusInternalServerError, "failed to get trace", err.Error())
		}
		if len(spans) == 0 {
			return response.CustomResponse(c, fiber.StatusNotFound, "trace not found", nil)
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", fiber.Map{"traceId": traceID, "spans": spans})
	}
}

func SearchLogs(svc *service.TelemetryService) fiber.Handler {
	return func(c fiber.Ctx) error {
		from, to, err := parseTelemetryRange(c)
		if err != nil {
			return badRequest(c, err)
		}
		limit, err := parseLimit(c, 100)
		if err != nil {
			return badRequest(c, err)
		}
		traceID, spanID := c.Query("traceId"), c.Query("spanId")
		if traceID != "" && !validHexID(traceID, 16) {
			return badRequest(c, errors.New("traceId must be 32 hexadecimal characters"))
		}
		if spanID != "" && !validHexID(spanID, 8) {
			return badRequest(c, errors.New("spanId must be 16 hexadecimal characters"))
		}
		logs, err := svc.SearchLogs(c.Context(), service.LogSearch{
			From: from, To: to, ServiceName: c.Query("service"),
			TraceID: traceID, SpanID: spanID, Limit: limit,
		})
		if err != nil {
			return response.CustomResponse(c, fiber.StatusInternalServerError, "failed to search logs", err.Error())
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", logs)
	}
}

func parseTelemetryRange(c fiber.Ctx) (time.Time, time.Time, error) {
	now := time.Now().UTC()
	fromRaw, toRaw := c.Query("from"), c.Query("to")
	if fromRaw == "" && toRaw == "" {
		return now.Add(-time.Hour), now, nil
	}
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, errors.New("from and to must be provided together")
	}
	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("from must be RFC3339")
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil || to.Before(from) || to.After(now) || to.Sub(from) > maxTelemetryTimeRange {
		return time.Time{}, time.Time{}, errors.New("invalid time range; maximum is 24 hours")
	}
	return from, to, nil
}

func parseLimit(c fiber.Ctx, fallback int32) (int32, error) {
	raw := c.Query("limit")
	if raw == "" {
		return fallback, nil
	}
	limit, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || limit < 1 || limit > 500 {
		return 0, errors.New("limit must be between 1 and 500")
	}
	return int32(limit), nil
}

func validHexID(value string, bytes int) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == bytes
}
