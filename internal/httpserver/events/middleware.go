package events

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

// makeEventLoggingMiddleware returns a Fiber middleware that logs every
// incoming Zitadel event request body at Debug level. It does NOT consume
// the request body — downstream handlers can still read it via c.Bind().Body().
//
// WARNING: The raw body may contain PII (names, emails, phone numbers).
// Only enable Debug level in development/E2E environments, never in production.
func makeEventLoggingMiddleware(logger *zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		event := logger.Debug().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Str("content_type", string(c.Request().Header.ContentType()))

		if body := c.Body(); len(body) > 0 {
			event = event.RawJSON("body", body)
		}

		event.Msg("incoming event request")

		return c.Next()
	}
}
