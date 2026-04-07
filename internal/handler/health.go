package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
)

// HealthCheck returns a handler that reports service health.
// Returns 200 if all dependencies are healthy, 503 otherwise.
func HealthCheck(nc *nats.Conn, rc *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		status := fiber.Map{}
		degraded := false

		if nc.Status() != nats.CONNECTED {
			status["nats"] = nc.Status().String()
			degraded = true
		} else {
			status["nats"] = "ok"
		}

		if err := rc.Ping(c.Context()).Err(); err != nil {
			status["redis"] = "unavailable"
			degraded = true
		} else {
			status["redis"] = "ok"
		}

		if degraded {
			status["status"] = "degraded"
			return c.Status(fiber.StatusServiceUnavailable).JSON(status)
		}

		status["status"] = "ok"
		return c.Status(fiber.StatusOK).JSON(status)
	}
}
