// Package middleware holds Fiber middleware used on Zitadel webhook
// routes: JSON content-type enforcement and HMAC-SHA256 signature
// verification.
package middleware

import (
	"github.com/gofiber/fiber/v3"
)

// ContentType rejects non-JSON requests with 422. Zitadel Actions v2
// targets always POST JSON, so any other content type is treated as a
// malformed delivery.
func ContentType() fiber.Handler {
	return func(c fiber.Ctx) error {
		if !c.Is("json") {
			return fiber.ErrUnprocessableEntity
		}
		return c.Next()
	}
}
