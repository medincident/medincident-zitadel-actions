package handler

import (
	"fmt"

	"github.com/go-redsync/redsync/v4"
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

		mutex := h.rs.NewMutex(
			fmt.Sprintf("%s%s:%s", h.cfg.Redis.LockPrefix, envelope.AggregateType, envelope.AggregateID),
			redsync.WithExpiry(h.cfg.Redis.LockExpiry.Duration()),
		)

		if err := mutex.LockContext(c.Context()); err != nil {
			return oops.In("handler").Code("lock_failed").With("event_type", "user.human.profile.changed").With("aggregate_id", envelope.AggregateID).Wrap(err)
		}
		defer func() {
			if ok, err := mutex.Unlock(); err != nil {
				h.logger.Error().Err(err).Str("aggregate_id", envelope.AggregateID).Msg("failed to unlock mutex")
			} else if !ok {
				h.logger.Warn().Str("aggregate_id", envelope.AggregateID).Msg("mutex was not unlocked")
			}
		}()

		if err := h.pub.Publish(c.Context(), events); err != nil {
			return oops.In("handler").Code("publish_failed").With("event_type", "user.human.profile.changed").With("user_id", envelope.UserID).Wrap(err)
		}

		h.logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Int("published_events", len(events)).
			Msg("processed UserHumanProfileChanged")

		return c.SendStatus(fiber.StatusOK)
	}
}
