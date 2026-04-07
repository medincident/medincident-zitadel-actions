package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// PostHumanUserEmailVerified handles POST /events/user/human/email/verified.
func (h *EventHandler) PostHumanUserEmailVerified() fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.UserHumanEmailVerified])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code("bind_failed").With("event_type", "user.human.email.verified").Wrap(err)
		}

		err := h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			events, err := mapper.MapUserEmailVerified(envelope)
			if err != nil {
				return oops.In("handler").Code("map_failed").With("event_type", "user.human.email.verified").With("user_id", envelope.UserID).Wrap(err)
			}

			return h.pub.Publish(c.Context(), events)
		})
		if err != nil {
			return oops.In("handler").Code("publish_failed").With("event_type", "user.human.email.verified").With("user_id", envelope.UserID).Wrap(err)
		}

		h.logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Msg("processed UserHumanEmailVerified")

		return c.SendStatus(fiber.StatusOK)
	}
}
