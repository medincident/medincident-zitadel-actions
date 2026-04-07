package handler

import (
	"context"
	"fmt"

	"github.com/go-redsync/redsync/v4"
	"github.com/rs/zerolog"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
)

// EventHandler handles Zitadel webhook events with distributed locking and publishing.
type EventHandler struct {
	logger *zerolog.Logger
	pub    *publish.Publisher
	rs     *redsync.Redsync
	cfg    *config.Config
}

// NewEventHandler creates an EventHandler.
func NewEventHandler(logger *zerolog.Logger, pub *publish.Publisher, rs *redsync.Redsync, cfg *config.Config) *EventHandler {
	return &EventHandler{logger: logger, pub: pub, rs: rs, cfg: cfg}
}

// withMutex acquires a distributed lock for the given aggregate, executes fn,
// and releases the lock. Guarantees per-aggregate ordering of publishes.
func (h *EventHandler) withMutex(ctx context.Context, aggregateType, aggregateID string, fn func() error) error {
	mutex := h.rs.NewMutex(
		fmt.Sprintf("%s%s:%s", h.cfg.Redis.LockPrefix, aggregateType, aggregateID),
		redsync.WithExpiry(h.cfg.Redis.LockExpiry.Duration()),
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
