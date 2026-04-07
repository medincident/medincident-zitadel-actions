package handler

import (
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
