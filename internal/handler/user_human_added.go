package handler

import (
	"fmt"

	"github.com/go-redsync/redsync/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// PostHumanUserAdded handles POST /events/user/human/added.
func (h *EventHandler) PostHumanUserAdded() fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.UserHumanAdded])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code("bind_failed").With("event_type", "user.human.added").Wrap(err)
		}

		mutex := h.rs.NewMutex(
			fmt.Sprintf("%s%s:%s", h.cfg.Redis.LockPrefix, envelope.AggregateType, envelope.AggregateID),
			redsync.WithExpiry(h.cfg.Redis.LockExpiry.Duration()),
		)

		if err := mutex.LockContext(c.Context()); err != nil {
			return oops.In("handler").Code("lock_failed").With("event_type", "user.human.added").With("aggregate_id", envelope.AggregateID).Wrap(err)
		}
		defer func() {
			if ok, err := mutex.Unlock(); err != nil {
				h.logger.Error().Err(err).Str("aggregate_id", envelope.AggregateID).Msg("failed to unlock mutex")
			} else if !ok {
				h.logger.Warn().Str("aggregate_id", envelope.AggregateID).Msg("mutex was not unlocked")
			}
		}()

		events, err := mapper.MapUserHumanAdded(envelope)
		if err != nil {
			return oops.In("handler").Code("map_failed").With("event_type", "user.human.added").With("user_id", envelope.UserID).Wrap(err)
		}

		if err := h.pub.Publish(c.Context(), events); err != nil {
			return oops.In("handler").Code("publish_failed").With("event_type", "user.human.added").With("user_id", envelope.UserID).Wrap(err)
		}

		h.logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Int("published_events", len(events)).
			Msg("processed UserHumanAdded")

		return c.SendStatus(fiber.StatusOK)
	}
}
