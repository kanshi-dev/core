package api

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/kanshi-dev/core/internal/service"
)

type Server struct {
	App            *fiber.App
	MetricsService *service.MetricsService
	AgentService   *service.AgentsService
	ping           func(context.Context) error
}

func NewServer(agentService *service.AgentsService, metricsService *service.MetricsService, ping func(context.Context) error) *Server {
	app := fiber.New()
	app.Use(cors.New())
	server := &Server{
		App:            app,
		MetricsService: metricsService,
		AgentService:   agentService,
		ping:           ping,
	}

	InitRouter(app, server)

	return server
}
