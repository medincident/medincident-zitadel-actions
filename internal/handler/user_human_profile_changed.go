package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// derefStr safely dereferences a pointer to string, returning empty string if nil.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// derefInt safely dereferences a pointer to int, returning 0 if nil.
func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

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
			Str("first_name", derefStr(envelope.EventPayload.FirstName)).
			Str("last_name", derefStr(envelope.EventPayload.LastName)).
			Str("nick_name", derefStr(envelope.EventPayload.NickName)).
			Str("display_name", derefStr(envelope.EventPayload.DisplayName)).
			Str("preferred_language", derefStr(envelope.EventPayload.PreferredLanguage)).
			Int("gender", derefInt(envelope.EventPayload.Gender)).
			Msg("received UserHumanProfileChanged event")

		return c.SendStatus(fiber.StatusOK)
	}
}
