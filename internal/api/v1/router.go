package v1

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/api/v1/handlers"
	"github.com/kanshi-dev/core/internal/service"
)

func Init(
	router fiber.Router,
	metricService *service.MetricsService,
	agentService *service.AgentsService,
	alertService *service.AlertsService,
) {
	router.Get("/metrics", handlers.GetMetrics(metricService))
	router.Get("/metrics/aggregate", handlers.GetAggregatedMetrics(metricService))
	router.Get("/agents", handlers.GetAgentHeartBeat(agentService))

	router.Get("/alerts/rules", handlers.ListAlertRules(alertService))
	router.Post("/alerts/rules", handlers.CreateAlertRule(alertService))
	router.Put("/alerts/rules/:id", handlers.UpdateAlertRule(alertService))
	router.Delete("/alerts/rules/:id", handlers.DeleteAlertRule(alertService))
	router.Get("/alerts/active", handlers.GetActiveAlerts(alertService))
	router.Get("/alerts/events", handlers.GetAlertHistory(alertService))
}
