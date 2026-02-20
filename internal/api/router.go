package api

import (
	"github.com/gofiber/fiber/v3"
	v1 "github.com/kanshi-dev/core/internal/api/v1"
	"github.com/kanshi-dev/core/internal/db"
)

func InitRouter(app *fiber.App, queries *db.Queries) {

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
	v1Group := api.Group("/v1")
	v1.Init(v1Group, queries)

	//404 Endpoint
	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "route not found",
			"path":  c.Path(),
		})
	})
}
