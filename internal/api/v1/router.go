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
) {
	router.Get("/metrics", handlers.GetMetrics(metricService))
	router.Get("/metrics/aggregate", handlers.GetAggregatedMetrics(metricService))
	router.Get("/agents", handlers.GetAgentHeartBeat(agentService))
}
