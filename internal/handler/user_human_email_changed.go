package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// PostHumanUserEmailChanged handles POST /events/user/human/email/changed.
func (h *EventHandler) PostHumanUserEmailChanged() fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.UserHumanEmailChanged])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code("bind_failed").With("event_type", "user.human.email.changed").Wrap(err)
		}

		err := h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			events, err := mapper.MapUserEmailChanged(envelope)
			if err != nil {
				return oops.In("handler").Code("map_failed").With("event_type", "user.human.email.changed").With("user_id", envelope.UserID).Wrap(err)
			}

			return h.pub.Publish(c.Context(), events)
		})
		if err != nil {
			return oops.In("handler").Code("publish_failed").With("event_type", "user.human.email.changed").With("user_id", envelope.UserID).Wrap(err)
		}

		h.logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Msg("processed UserHumanEmailChanged")

		return c.SendStatus(fiber.StatusOK)
	}
}
