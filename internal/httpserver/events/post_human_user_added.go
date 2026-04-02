package events

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"

	zitadelevents "github.com/medincident/medincident-zitadel-actions/internal/zitadel/actions/events"
)

func makePostHumanUserAddedHandler(logger *zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !c.Is("json") {
			return fiber.ErrUnprocessableEntity
		}

		envelope := new(zitadelevents.Envelope)
		if err := c.Bind().Body(envelope); err != nil {
			return err
		}

		userData := new(zitadelevents.UserHumanAdded)
		if err := zitadelevents.Unmarshal(envelope, userData); err != nil {
			return err
		}

		logger.Info().
			Str("user_id", envelope.AggregateID).
			Str("event_type", envelope.EventType).
			Str("first_name", userData.FirstName).
			Str("last_name", userData.LastName).
			Str("email", userData.Email).
			Msg("received UserHumanAdded event")

		return c.SendStatus(fiber.StatusOK)
	}
}
