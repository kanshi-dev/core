package api

import (
	"github.com/gofiber/fiber/v3"
	"github.com/kanshi-dev/core/internal/db"
)

type Server struct {
	App *fiber.App
}

func NewServer(queries *db.Queries) *Server {
	app := fiber.New()

	InitRouter(app, queries)

	return &Server{
		App: app,
	}
}
