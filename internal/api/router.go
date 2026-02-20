package api

import (
	"github.com/gofiber/fiber/v3"
	v1 "github.com/kanshi-dev/core/internal/api/v1"
	"github.com/kanshi-dev/core/internal/db"
)

func InitRouter(app *fiber.App, queries *db.Queries) {
	api := app.Group("/api")
	v1Group := api.Group("/v1")
	v1.Init(v1Group, queries)
}
