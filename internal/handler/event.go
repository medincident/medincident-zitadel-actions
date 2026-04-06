package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

// PostAnyEvent returns a handler for the catch-all POST /events endpoint.
// It logs the raw JSON body of any Zitadel event.
func PostAnyEvent(logger *zerolog.Logger) fiber.Handler {
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

		event.Msg("received event")

		return c.SendStatus(fiber.StatusOK)
	}
}
