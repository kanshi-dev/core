package v1

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/api/v1/handlers"
	"github.com/kanshi-dev/core/internal/db"
)

func Init(router fiber.Router, queries *db.Queries) {
	router.Get("/metrics", handlers.GetMetrics(queries))
}
