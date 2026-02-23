package api

import (
	"github.com/gofiber/fiber/v3"
	v1 "github.com/kanshi-dev/core/internal/api/v1"
)

func InitRouter(app *fiber.App, apiSever *Server) {

	//Root Endpoint
	app.Get("/", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"service": "kanshi-core",
			"status":  "running",
		})
	})

	//Health Endpoint
	app.Get("/health", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})

	//Versioning Init
	api := app.Group("/api")

	// Verson 1
	v1Group := api.Group("/v1")

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
