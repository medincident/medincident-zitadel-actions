// Package handler holds the Fiber HTTP handlers for Zitadel Actions v2
// webhooks. Each handler binds a typed [zitadel.Envelope], maps the
// payload to a protobuf message and publishes it via [publish.Publisher].
package handler

import (
	"context"
	"fmt"

	"github.com/go-redsync/redsync/v4"
	"github.com/rs/zerolog"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
)

// EventHandler owns the cross-cutting dependencies shared by every
// per-event handler method: the logger, the NATS publisher, the Redlock
// client and the application config.
type EventHandler struct {
	logger *zerolog.Logger
	pub    *publish.Publisher
	rs     *redsync.Redsync
	cfg    *config.Config
}

func NewEventHandler(logger *zerolog.Logger, pub *publish.Publisher, rs *redsync.Redsync, cfg *config.Config) *EventHandler {
	return &EventHandler{logger: logger, pub: pub, rs: rs, cfg: cfg}
}

// withMutex serializes handling per aggregate so that concurrent
// webhooks for the same aggregate can never publish out of sequence.
// The lock key is scoped by aggregate type and ID; the NATS JetStream
// Nats-Msg-Id dedup header provides the final idempotency guarantee.
func (h *EventHandler) withMutex(ctx context.Context, aggregateType, aggregateID string, fn func() error) error {
	mutex := h.rs.NewMutex(
		fmt.Sprintf("%s%s:%s", h.cfg.Redis.LockPrefix, aggregateType, aggregateID),
		redsync.WithExpiry(h.cfg.Redis.LockExpiry),
	)

	if err := mutex.LockContext(ctx); err != nil {
		return err
	}
	defer func() {
		if ok, err := mutex.UnlockContext(ctx); err != nil {
			h.logger.Error().Err(err).Str("aggregate_id", aggregateID).Msg("failed to unlock mutex")
		} else if !ok {
			h.logger.Warn().Str("aggregate_id", aggregateID).Msg("mutex was not unlocked")
		}
	}()

	return fn()
}
