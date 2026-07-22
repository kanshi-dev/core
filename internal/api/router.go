package api

import (
	"context"
	"crypto/subtle"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	v1 "github.com/kanshi-dev/core/internal/api/v1"
)

func InitRouter(app *fiber.App, apiSever *Server, dashboardKey string) {

	//Root Endpoint
	app.Get("/", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"service": "kanshi-core",
			"status":  "running",
		})
	})

	//Health Endpoint
	app.Get("/health", func(c fiber.Ctx) error {
		statusCode := fiber.StatusServiceUnavailable
		body := fiber.Map{"status": "degraded", "db": "error"}
		if apiSever.ping != nil {
			ctx, cancel := context.WithTimeout(c.Context(), time.Second)
			defer cancel()
			if apiSever.ping(ctx) == nil {
				statusCode = fiber.StatusOK
				body = fiber.Map{"status": "ok", "db": "ok"}
			}
		}
		return c.Status(statusCode).JSON(body)
	})

	//Versioning Init
	api := app.Group("/api")

	// Verson 1
	v1Group := api.Group("/v1")
	v1Group.Use(func(c fiber.Ctx) error {
		if c.Method() == fiber.MethodOptions {
			return c.Next()
		}
		if !authorized(c.Get(fiber.HeaderAuthorization), dashboardKey) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"code": fiber.StatusUnauthorized, "message": "unauthorized", "data": nil})
		}
		return c.Next()
	})

	// Calls Init() from v1/router.go
	v1.Init(v1Group, apiSever.MetricsService, apiSever.AgentService)

	//404 Endpoint
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "route not found",
			"path":  c.Path(),
		})
	})
}

func authorized(authorization, dashboardKey string) bool {
	provided, ok := strings.CutPrefix(authorization, "Bearer ")
	return ok && provided != "" && subtle.ConstantTimeCompare([]byte(provided), []byte(dashboardKey)) == 1
}
