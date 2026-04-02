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
			Str("user_name", userData.UserName).
			Str("first_name", userData.FirstName).
			Str("last_name", userData.LastName).
			Str("nick_name", userData.NickName).
			Str("display_name", userData.DisplayName).
			Str("preferred_language", userData.PreferredLanguage).
			Int("gender", userData.Gender).
			Str("email", userData.Email).
			Str("phone", userData.Phone).
			Msg("received UserHumanAdded event")

		return c.SendStatus(fiber.StatusOK)
	}
}
