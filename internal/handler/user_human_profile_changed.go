package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// PostHumanUserProfileChanged handles POST /events/user/human/profile/changed.
func (h *EventHandler) PostHumanUserProfileChanged() fiber.Handler {
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
			h.logger.Debug().
				Str("user_id", envelope.UserID).
				Msg("no profile fields changed, skipping publish")
			return c.SendStatus(fiber.StatusOK)
		}

		err = h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			return h.pub.Publish(c.Context(), events)
		})
		if err != nil {
			return err
		}

		h.logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Int("published_events", len(events)).
			Msg("processed UserHumanProfileChanged")

		return c.SendStatus(fiber.StatusOK)
	}
}
