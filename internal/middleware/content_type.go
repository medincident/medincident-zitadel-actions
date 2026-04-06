package middleware

import (
	"github.com/gofiber/fiber/v3"
)

// ContentType returns a Fiber middleware that rejects non-JSON requests
// with 422 Unprocessable Entity.
func ContentType() fiber.Handler {
	return func(c fiber.Ctx) error {
		if !c.Is("json") {
			return fiber.ErrUnprocessableEntity
		}
		return c.Next()
	}
}
