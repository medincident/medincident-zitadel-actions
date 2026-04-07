package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// PostHumanUserProfileChanged returns a handler for POST /events/user/human/profile/changed.
func PostHumanUserProfileChanged(logger *zerolog.Logger, js jetstream.JetStream) fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.UserHumanProfileChanged])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code("bind_failed").With("event_type", "user.human.profile.changed").Wrap(err)
		}

		events, err := mapper.MapUserHumanProfileChanged(envelope)
		if err != nil {
			return oops.In("handler").Code("map_failed").With("event_type", "user.human.profile.changed").With("user_id", envelope.UserID).Wrap(err)
		}

		if len(events) == 0 {
			logger.Debug().
				Str("user_id", envelope.UserID).
				Msg("no profile fields changed, skipping publish")
			return c.SendStatus(fiber.StatusOK)
		}

		if err := publish.PublishEvents(c.Context(), js, events); err != nil {
			return oops.In("handler").Code("publish_failed").With("event_type", "user.human.profile.changed").With("user_id", envelope.UserID).Wrap(err)
		}

		logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Int("published_events", len(events)).
			Msg("processed UserHumanProfileChanged")

		return c.SendStatus(fiber.StatusOK)
	}
}
