package api

import (
	"context"
	"strings"

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

func NewServer(agentService *service.AgentsService, metricsService *service.MetricsService, ping func(context.Context) error, dashboardKey, allowedOrigins string) *Server {
	app := fiber.New()
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173,http://127.0.0.1:5173"
	}
	origins := strings.Split(allowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	app.Use(cors.New(cors.Config{AllowOrigins: origins, AllowHeaders: []string{"Authorization", "Content-Type"}}))
	server := &Server{
		App:            app,
		MetricsService: metricsService,
		AgentService:   agentService,
		ping:           ping,
	}

	InitRouter(app, server, dashboardKey)

	return server
}
