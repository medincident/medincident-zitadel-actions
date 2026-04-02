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

		eb := oops.In("events").Code("user_human_profile_changed")

		envelope := new(zitadelevents.Envelope)
		if err := c.Bind().Body(envelope); err != nil {
			return eb.Wrap(err)
		}

		profileData := new(zitadelevents.UserHumanProfileChanged)
		if err := zitadelevents.Unmarshal(envelope, profileData); err != nil {
			return eb.Wrap(err)
		}

		logger.Info().
			Str("user_id", envelope.AggregateID).
			Str("event_type", envelope.EventType).
			Str("first_name", profileData.FirstName).
			Str("last_name", profileData.LastName).
			Str("nick_name", profileData.NickName).
			Str("display_name", profileData.DisplayName).
			Str("preferred_language", profileData.PreferredLanguage).
			Int("gender", profileData.Gender).
			Msg("received UserHumanProfileChanged event")

		return c.SendStatus(fiber.StatusOK)
	}
}
