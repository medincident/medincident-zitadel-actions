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
	ErrCodeUserHumanEmailChangedBindFailed = "bind_failed"
	ErrCodeUserHumanEmailChangedMapFailed  = "map_failed"
)

const subjectUserHumanEmailChanged = "zitadel.users.v1.human.email.changed"

// PostHumanUserEmailChanged handles POST /events/user/human/email/changed.
func (h *EventHandler) PostHumanUserEmailChanged() fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.UserHumanEmailChanged])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code(ErrCodeUserHumanEmailChangedBindFailed).With("event_type", "user.human.email.changed").Wrap(err)
		}

		err := h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			payload, err := mapper.BuildUserHumanEmailChanged(envelope)
			if err != nil {
				return oops.In("handler").Code(ErrCodeUserHumanEmailChangedMapFailed).With("event_type", "user.human.email.changed").With("user_id", envelope.UserID).Wrap(err)
			}

			return h.pub.PublishZitadelEvent(c.Context(), subjectUserHumanEmailChanged, publish.FromZitadelEnvelope(envelope), payload)
		})
		if err != nil {
			return err
		}

		h.logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Msg("processed UserHumanEmailChanged")

		return c.SendStatus(fiber.StatusOK)
	}
}
