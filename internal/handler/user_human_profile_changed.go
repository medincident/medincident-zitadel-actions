package handler

import (
	"slices"

	"github.com/gofiber/fiber/v3"
	"github.com/samber/oops"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/publish"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

const (
	ErrCodeUserHumanProfileChangedBindFailed = "bind_failed"
	ErrCodeUserHumanProfileChangedMapFailed  = "map_failed"
)

const subjectUserHumanProfileChanged = "zitadel.users.v1.human.profile.changed"

// PostHumanUserProfileChanged handles profile updates. The raw body is
// cloned before binding so the mapper can rebuild a FieldMask from the
// original JSON key set (Zitadel only sends the fields that changed, and
// value-type Go fields cannot distinguish "zero" from "absent").
func (h *EventHandler) PostHumanUserProfileChanged() fiber.Handler {
	return func(c fiber.Ctx) error {
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
