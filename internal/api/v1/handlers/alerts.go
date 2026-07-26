package handlers

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/api/v1/response"
	"github.com/kanshi-dev/core/internal/service"
)

func ListAlertRules(svc *service.AlertsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		rules, err := svc.ListRules(c.Context())
		if err != nil {
			return response.CustomResponse(c, fiber.StatusInternalServerError, "failed to list alert rules", err.Error())
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", rules)
	}
}

func CreateAlertRule(svc *service.AlertsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		var in service.AlertRuleInput
		if err := c.Bind().Body(&in); err != nil {
			return badRequest(c, errors.New("invalid request body"))
		}
		rule, err := svc.CreateRule(c.Context(), in)
		if err != nil {
			return alertWriteError(c, err)
		}
		return response.CustomResponse(c, fiber.StatusCreated, "success", rule)
	}
}

func UpdateAlertRule(svc *service.AlertsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := parseRuleID(c)
		if err != nil {
			return badRequest(c, err)
		}
		var in service.AlertRuleInput
		if err := c.Bind().Body(&in); err != nil {
			return badRequest(c, errors.New("invalid request body"))
		}
		rule, err := svc.UpdateRule(c.Context(), id, in)
		if err != nil {
			return alertWriteError(c, err)
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", rule)
	}
}

func DeleteAlertRule(svc *service.AlertsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		id, err := parseRuleID(c)
		if err != nil {
			return badRequest(c, err)
		}
		if err := svc.DeleteRule(c.Context(), id); err != nil {
			return alertWriteError(c, err)
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", nil)
	}
}

func GetActiveAlerts(svc *service.AlertsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		events, err := svc.ListActive(c.Context())
		if err != nil {
			return response.CustomResponse(c, fiber.StatusInternalServerError, "failed to get active alerts", err.Error())
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", events)
	}
}

func GetAlertHistory(svc *service.AlertsService) fiber.Handler {
	return func(c fiber.Ctx) error {
		lim := 0 // service defaults to 100
		if s := c.Query("limit"); s != "" {
			n, err := strconv.Atoi(s)
			if err != nil || n <= 0 {
				return badRequest(c, errors.New("invalid limit"))
			}
			lim = n
		}
		events, err := svc.ListHistory(c.Context(), int32(lim))
		if err != nil {
			return response.CustomResponse(c, fiber.StatusInternalServerError, "failed to get alert history", err.Error())
		}
		return response.CustomResponse(c, fiber.StatusOK, "success", events)
	}
}

func parseRuleID(c fiber.Ctx) (int64, error) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil {
		return 0, errors.New("invalid rule id")
	}
	return id, nil
}

func alertWriteError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidRule):
		return badRequest(c, err)
	case errors.Is(err, service.ErrRuleNotFound):
		return response.CustomResponse(c, fiber.StatusNotFound, "not found", err.Error())
	default:
		return response.CustomResponse(c, fiber.StatusInternalServerError, "alert rule operation failed", err.Error())
	}
}
