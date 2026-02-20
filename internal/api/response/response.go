package response

import "github.com/gofiber/fiber/v3"

func CustomResponse(c fiber.Ctx, code int, message string, data any) error {
	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"message": message,
		"data":    data,
	})
}
