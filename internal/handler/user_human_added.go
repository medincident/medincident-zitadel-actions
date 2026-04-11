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
	ErrCodeUserHumanAddedBindFailed = "bind_failed"
	ErrCodeUserHumanAddedMapFailed  = "map_failed"
)

const subjectUserHumanAdded = "zitadel.users.v1.human.added"

// PostHumanUserAdded handles POST /events/user/human/added.
func (h *EventHandler) PostHumanUserAdded() fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.UserHumanAdded])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code(ErrCodeUserHumanAddedBindFailed).With("event_type", "user.human.added").Wrap(err)
		}

		err := h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			payload, err := mapper.BuildUserHumanAdded(envelope)
			if err != nil {
				return oops.In("handler").Code(ErrCodeUserHumanAddedMapFailed).With("event_type", "user.human.added").With("user_id", envelope.UserID).Wrap(err)
			}

			return h.pub.PublishZitadelEvent(c.Context(), subjectUserHumanAdded, publish.FromZitadelEnvelope(envelope), payload)
		})
		if err != nil {
			return err
		}

		h.logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Msg("processed UserHumanAdded")

		return c.SendStatus(fiber.StatusOK)
	}
}
