package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// PostSessionUserChecked handles POST /events/session/user/checked.
func (h *EventHandler) PostSessionUserChecked() fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.SessionUserChecked])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code("bind_failed").With("event_type", "session.user.checked").Wrap(err)
		}

		err := h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			events, err := mapper.MapSessionUserChecked(envelope)
			if err != nil {
				return oops.In("handler").Code("map_failed").With("event_type", "session.user.checked").With("session_id", envelope.AggregateID).Wrap(err)
			}

			return h.pub.Publish(c.Context(), events)
		})
		if err != nil {
			return oops.In("handler").Code("publish_failed").With("event_type", "session.user.checked").With("session_id", envelope.AggregateID).Wrap(err)
		}

		h.logger.Info().
			Str("session_id", envelope.AggregateID).
			Str("user_id", envelope.EventPayload.UserID).
			Str("event_type", envelope.EventType).
			Msg("processed SessionUserChecked")

		return c.SendStatus(fiber.StatusOK)
	}
}
