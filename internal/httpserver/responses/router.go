package responses

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
)

func SetupRoutes(router fiber.Router, logger *zerolog.Logger) {
	if logger == nil {
		nop := zerolog.Nop()
		logger = &nop
	}

	router.Post("/response", makePostAnyResponseHandler(logger))
}
