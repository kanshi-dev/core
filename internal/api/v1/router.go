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
	telemetryService *service.TelemetryService,
	profilesService *service.ProfilesService,
) {
	router.Get("/metrics", handlers.GetMetrics(metricService))
	router.Get("/metrics/aggregate", handlers.GetAggregatedMetrics(metricService))
	router.Get("/agents", handlers.GetAgentHeartBeat(agentService))
	router.Post("/agents/:agentId/profiles", handlers.CreateProfile(profilesService))
	router.Get("/agents/:agentId/profiles", handlers.ListProfiles(profilesService))
	router.Get("/profiles/:id", handlers.GetProfile(profilesService))
	router.Get("/profiles/:id/flamegraph", handlers.GetProfileFlamegraph(profilesService))
	router.Get("/profiles/:id/download", handlers.DownloadProfile(profilesService))

	router.Get("/alerts/rules", handlers.ListAlertRules(alertService))
	router.Post("/alerts/rules", handlers.CreateAlertRule(alertService))
	router.Put("/alerts/rules/:id", handlers.UpdateAlertRule(alertService))
	router.Delete("/alerts/rules/:id", handlers.DeleteAlertRule(alertService))
	router.Get("/alerts/active", handlers.GetActiveAlerts(alertService))
	router.Get("/alerts/events", handlers.GetAlertHistory(alertService))

	router.Get("/services", handlers.ListServices(telemetryService))
	router.Get("/traces", handlers.SearchTraces(telemetryService))
	router.Get("/traces/:traceId", handlers.GetTrace(telemetryService))
	router.Get("/logs", handlers.SearchLogs(telemetryService))
}
