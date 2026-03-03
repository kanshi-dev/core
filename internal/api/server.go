package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/kanshi-dev/core/internal/service"
)

type Server struct {
	App            *fiber.App
	MetricsService *service.MetricsService
	AgentService   *service.AgentsService
}

func NewServer(agentService *service.AgentsService, metricsService *service.MetricsService) *Server {
	app := fiber.New()
	app.Use(cors.New())
	server := &Server{
		App:            app,
		MetricsService: metricsService,
		AgentService:   agentService,
	}

	InitRouter(app, server)

	return &Server{
		App:            app,
		MetricsService: metricsService,
		AgentService:   agentService,
	}
}
