package events

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

func makePostAnyEventHandler(logger *zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !c.Is("json") {
			return fiber.ErrUnprocessableEntity
		}

		event := logger.Info().
			Str("content_type", string(c.Request().Header.ContentType()))

		if body := c.Body(); len(body) > 0 {
			event = event.RawJSON("body", body)
		}

		event.Msg("received event")

		return c.SendStatus(fiber.StatusOK)
	}
}
