package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

// PostDebugWebhook returns a handler that logs the raw JSON body of any
// Zitadel webhook. For debugging only — no binding, no publishing.
func PostDebugWebhook(logger *zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		event := logger.Info().
			Str("content_type", string(c.Request().Header.ContentType()))

		if body := c.Body(); len(body) > 0 {
			if json.Valid(body) {
				event = event.RawJSON("body", body)
			} else {
				event = event.Bytes("body", body)
			}
		}

		event.Msg("received debug webhook")

		return c.SendStatus(fiber.StatusOK)
	}
}
