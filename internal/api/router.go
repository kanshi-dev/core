package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/api/handlers"
	"github.com/kanshi-dev/core/internal/api/response"
	"github.com/kanshi-dev/core/internal/db"
)

func InitRouter(app *fiber.App, queries *db.Queries) {
	v1 := app.Group("/v1")

	handlers.Init(queries)

	v1.Get("/", func(c fiber.Ctx) error {
		return response.CustomResponse(
			c,
			fiber.StatusOK,
			"Welcome to kanshi-core",
			nil,
		)
	})

	v1.Get("/metrics", handlers.GetMetrics)
}
