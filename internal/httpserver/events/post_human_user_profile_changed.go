package events

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/samber/oops"

	zitadelevents "github.com/medincident/medincident-zitadel-actions/internal/zitadel/actions/events"
)

func makePostHumanUserProfileChangedHandler(logger *zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !c.Is("json") {
			return fiber.ErrUnprocessableEntity
		}

		envelope := new(zitadelevents.Envelope[zitadelevents.UserHumanProfileChanged])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("events").Code("user_human_profile_changed").Wrap(err)
		}

		logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Str("first_name", envelope.EventPayload.FirstName).
			Str("last_name", envelope.EventPayload.LastName).
			Str("nick_name", envelope.EventPayload.NickName).
			Str("display_name", envelope.EventPayload.DisplayName).
			Str("preferred_language", envelope.EventPayload.PreferredLanguage).
			Int("gender", envelope.EventPayload.Gender).
			Msg("received UserHumanProfileChanged event")

		return c.SendStatus(fiber.StatusOK)
	}
}
