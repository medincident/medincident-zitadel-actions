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
	ErrCodeSessionUserCheckedBindFailed = "session_user_checked_bind_failed"
	ErrCodeSessionUserCheckedMapFailed  = "session_user_checked_map_failed"
)

const subjectSessionUserChecked = "zitadel.sessions.v1.user.checked"

// PostSessionUserChecked handles POST /events/session/user/checked.
func (h *EventHandler) PostSessionUserChecked() fiber.Handler {
	return func(c fiber.Ctx) error {
		envelope := new(zitadel.Envelope[zitadel.SessionUserChecked])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code(ErrCodeSessionUserCheckedBindFailed).With("event_type", "session.user.checked").Wrap(err)
		}

		err := h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			payload, err := mapper.BuildSessionUserChecked(envelope)
			if err != nil {
				return oops.In("handler").Code(ErrCodeSessionUserCheckedMapFailed).With("event_type", "session.user.checked").With("session_id", envelope.AggregateID).Wrap(err)
			}

			return h.pub.PublishZitadelEvent(c.Context(), subjectSessionUserChecked, publish.FromZitadelEnvelope(envelope), payload)
		})
		if err != nil {
			return err
		}

		h.logger.Info().
			Str("session_id", envelope.AggregateID).
			Str("user_id", envelope.EventPayload.UserID).
			Str("event_type", envelope.EventType).
			Msg("processed SessionUserChecked")

		return c.SendStatus(fiber.StatusOK)
	}
}
