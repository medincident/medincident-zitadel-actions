package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nats-io/nats.go"
)

// HealthCheck returns a handler that reports service health.
// Returns 200 if NATS is connected, 503 otherwise.
func HealthCheck(nc *nats.Conn) fiber.Handler {
	return func(c fiber.Ctx) error {
		if nc.Status() != nats.CONNECTED {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"status": "degraded",
				"nats":   nc.Status().String(),
			})
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	}
}
