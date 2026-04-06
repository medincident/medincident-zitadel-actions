package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// PostHumanUserProfileChanged returns a handler for POST /events/user/human/profile/changed.
func PostHumanUserProfileChanged(logger *zerolog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		if !c.Is("json") {
			return fiber.ErrUnprocessableEntity
		}

		envelope := new(zitadel.Envelope[zitadel.UserHumanProfileChanged])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code("bind_failed").With("event_type", "user.human.profile.changed").Wrap(err)
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
