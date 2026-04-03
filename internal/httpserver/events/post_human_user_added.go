package events

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/samber/oops"

	zitadelevents "github.com/medincident/medincident-zitadel-actions/internal/zitadel/actions/events"
)

func makePostHumanUserAddedHandler(logger *zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !c.Is("json") {
			return fiber.ErrUnprocessableEntity
		}

		envelope := new(zitadelevents.Envelope[zitadelevents.UserHumanAdded])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("events").Code("user_human_added").Wrap(err)
		}

		logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Str("user_name", envelope.EventPayload.UserName).
			Str("first_name", envelope.EventPayload.FirstName).
			Str("last_name", envelope.EventPayload.LastName).
			Str("nick_name", envelope.EventPayload.NickName).
			Str("display_name", envelope.EventPayload.DisplayName).
			Str("preferred_language", envelope.EventPayload.PreferredLanguage).
			Int("gender", envelope.EventPayload.Gender).
			Str("email", envelope.EventPayload.Email).
			Str("phone", envelope.EventPayload.Phone).
			Msg("received UserHumanAdded event")

		return c.SendStatus(fiber.StatusOK)
	}
}
