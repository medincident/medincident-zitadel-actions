package handler

import (
	"encoding/json"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

// PostDebugWebhook logs the raw JSON body of any Zitadel webhook it
// receives and returns 200. For local inspection only — no binding,
// no publishing, no HMAC check happens here (the middleware still
// runs upstream).
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
