package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// Error codes emitted by this handler. Declared at file level so
// each emit site is grep-local; string values carry the handler name
// so interceptors can distinguish per-event telemetry.
const (
	ErrCodeUserHumanEmailVerifiedBindFailed = "bind_failed"
	ErrCodeUserHumanEmailVerifiedMapFailed  = "map_failed"
)

const subjectUserHumanEmailVerified = "zitadel.users.v1.human.email.verified"

// PostHumanUserEmailVerified handles POST /events/user/human/email/verified.
func (h *EventHandler) PostHumanUserEmailVerified() fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.UserHumanEmailVerified])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code(ErrCodeUserHumanEmailVerifiedBindFailed).With("event_type", "user.human.email.verified").Wrap(err)
		}

		err := h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			payload, err := mapper.BuildUserHumanEmailVerified(envelope)
			if err != nil {
				return oops.In("handler").Code(ErrCodeUserHumanEmailVerifiedMapFailed).With("event_type", "user.human.email.verified").With("user_id", envelope.UserID).Wrap(err)
			}

			return h.pub.PublishZitadelEvent(c.Context(), subjectUserHumanEmailVerified, publish.FromZitadelEnvelope(envelope), payload)
		})
		if err != nil {
			return err
		}

		h.logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Msg("processed UserHumanEmailVerified")

		return c.SendStatus(fiber.StatusOK)
	}
}
