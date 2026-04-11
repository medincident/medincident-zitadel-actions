package handler

import (
	"slices"

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
	ErrCodeUserHumanProfileChangedBindFailed = "user_human_profile_changed_bind_failed"
	ErrCodeUserHumanProfileChangedMapFailed  = "user_human_profile_changed_map_failed"
)

const subjectUserHumanProfileChanged = "zitadel.users.v1.human.profile.changed"

// PostHumanUserProfileChanged handles POST /events/user/human/profile/changed.
func (h *EventHandler) PostHumanUserProfileChanged() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Capture raw body before bind (bind may consume/modify the body buffer).
		rawBody := slices.Clone(c.Body())

		envelope := new(zitadel.Envelope[zitadel.UserHumanProfileChanged])
		if err := c.Bind().Body(envelope); err != nil {
			return oops.In("handler").Code(ErrCodeUserHumanProfileChangedBindFailed).With("event_type", "user.human.profile.changed").Wrap(err)
		}

		err := h.withMutex(c.Context(), envelope.AggregateType, envelope.AggregateID, func() error {
			payload, err := mapper.BuildUserHumanProfileChanged(rawBody, envelope)
			if err != nil {
				return oops.In("handler").Code(ErrCodeUserHumanProfileChangedMapFailed).With("event_type", "user.human.profile.changed").With("user_id", envelope.UserID).Wrap(err)
			}

			return h.pub.PublishZitadelEvent(c.Context(), subjectUserHumanProfileChanged, publish.FromZitadelEnvelope(envelope), payload)
		})
		if err != nil {
			return err
		}

		h.logger.Info().
			Str("user_id", envelope.UserID).
			Str("event_type", envelope.EventType).
			Msg("processed UserHumanProfileChanged")

		return c.SendStatus(fiber.StatusOK)
	}
}
