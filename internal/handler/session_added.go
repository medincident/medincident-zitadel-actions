package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

const (
	ErrCodeSessionAddedBindFailed = "bind_failed"
	ErrCodeSessionAddedMapFailed  = "map_failed"
)

const subjectSessionAdded = "zitadel.sessions.v1.added"

func (h *EventHandler) PostSessionAdded() fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.SessionAdded])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code(ErrCodeSessionAddedBindFailed).With("event_type", "session.added").Wrap(err)
		}

		err := h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			payload, err := mapper.BuildSessionAdded(envelope)
			if err != nil {
				return oops.In("handler").Code(ErrCodeSessionAddedMapFailed).With("event_type", "session.added").With("session_id", envelope.AggregateID).Wrap(err)
			}

			return h.pub.PublishZitadelEvent(c.Context(), subjectSessionAdded, publish.FromZitadelEnvelope(envelope), payload)
		})
		if err != nil {
			return err
		}

		h.logger.Info().
			Str("session_id", envelope.AggregateID).
			Str("event_type", envelope.EventType).
			Msg("processed SessionAdded")

		return c.SendStatus(fiber.StatusOK)
	}
}
