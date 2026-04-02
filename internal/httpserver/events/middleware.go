package events

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

// makeEventLoggingMiddleware returns a Fiber middleware that logs every
// incoming Zitadel event request body at Debug level and the resulting
// response status at Info level. It does NOT consume the request body —
// downstream handlers can still read it via c.Bind().Body().
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

		err := c.Next()

		logger.Info().
			Str("method", c.Method()).
			Str("path", c.Path()).
			Int("status", c.Response().StatusCode()).
			Msg("event request handled")

		return err
	}
}
