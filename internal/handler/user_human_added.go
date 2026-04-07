package handler

import (
	"fmt"

	"github.com/go-redsync/redsync/v4"
	"github.com/gofiber/fiber/v3"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// PostHumanUserAdded returns a handler for POST /events/user/human/added.
func PostHumanUserAdded(logger *zerolog.Logger, js jetstream.JetStream, rs *redsync.Redsync, cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.UserHumanAdded])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code("bind_failed").With("event_type", "user.human.added").Wrap(err)
		}

		mutex := rs.NewMutex(
			fmt.Sprintf("%s%s:%s", cfg.Redis.LockPrefix, envelope.AggregateType, envelope.AggregateID),
			redsync.WithExpiry(cfg.Redis.LockExpiry.Duration()),
		)

		if err := mutex.LockContext(c.Context()); err != nil {
			return oops.In("handler").Code("lock_failed").With("event_type", "user.human.added").With("aggregate_id", envelope.AggregateID).Wrap(err)
		}
		defer func() {
			if ok, err := mutex.Unlock(); err != nil {
				logger.Error().Err(err).Str("aggregate_id", envelope.AggregateID).Msg("failed to unlock mutex")
			} else if !ok {
				logger.Warn().Str("aggregate_id", envelope.AggregateID).Msg("mutex was not unlocked")
			}
		}()

		events, err := mapper.MapUserHumanAdded(envelope)
		if err != nil {
			return oops.In("handler").Code("map_failed").With("event_type", "user.human.added").With("user_id", envelope.UserID).Wrap(err)
		}

		if err := publish.PublishEvents(c.Context(), logger, js, cfg.Publish, events); err != nil {
			return oops.In("handler").Code("publish_failed").With("event_type", "user.human.added").With("user_id", envelope.UserID).Wrap(err)
		}

		logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Int("published_events", len(events)).
			Msg("processed UserHumanAdded")

		return c.SendStatus(fiber.StatusOK)
	}
}
