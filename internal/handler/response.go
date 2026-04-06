package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

// PostAnyResponse returns a handler for the catch-all POST /responses endpoint.
func PostAnyResponse(logger *zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !c.Is("json") {
			return fiber.ErrUnprocessableEntity
		}

		event := logger.Info().
			Str("content_type", string(c.Request().Header.ContentType()))

		if body := c.Body(); len(body) > 0 {
			if json.Valid(body) {
				event = event.RawJSON("body", body)
			} else {
				event = event.Bytes("body", body)
			}
		}

		event.Msg("received response")

		return c.SendStatus(fiber.StatusOK)
	}
}
