package mapper

import (
	"google.golang.org/protobuf/proto"

	usersv1 "github.com/medincident/medincident-zitadel-actions/pkg/medincident/users/v1"

	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// MappedEvent pairs a NATS subject with a proto message to publish.
type MappedEvent struct {
	Subject string
	Message proto.Message
}

// MapUserHumanAdded converts a Zitadel UserHumanAdded event into domain proto events.
func MapUserHumanAdded(envelope *zitadel.Envelope[zitadel.UserHumanAdded]) []MappedEvent {
	p := &envelope.EventPayload

	return []MappedEvent{
		{
			Subject: "medincident.users.v1.created",
			Message: &usersv1.UserCreated{
				Username:          p.UserName,
				FirstName:         p.FirstName,
				LastName:          p.LastName,
				DisplayName:       p.DisplayName,
				NickName:          p.NickName,
				Gender:            mapGender(p.Gender),
				Email:             p.Email,
				PreferredLanguage: p.PreferredLanguage,
			},
		},
	}
}

// MapUserHumanProfileChanged converts a Zitadel profile.changed event
// into zero or more domain proto events, based on which fields are non-nil.
func MapUserHumanProfileChanged(envelope *zitadel.Envelope[zitadel.UserHumanProfileChanged]) []MappedEvent {
	p := &envelope.EventPayload
	var events []MappedEvent

	if p.FirstName != nil || p.LastName != nil || p.DisplayName != nil || p.NickName != nil {
		events = append(events, MappedEvent{
			Subject: "medincident.users.v1.name_changed",
			Message: &usersv1.UserNameChanged{
				FirstName:   p.FirstName,
				LastName:    p.LastName,
				DisplayName: p.DisplayName,
				NickName:    p.NickName,
			},
		})
	}

	if p.PreferredLanguage != nil {
		events = append(events, MappedEvent{
			Subject: "medincident.users.v1.language_changed",
			Message: &usersv1.UserLanguageChanged{
				PreferredLanguage: *p.PreferredLanguage,
			},
		})
	}

	if p.Gender != nil {
		events = append(events, MappedEvent{
			Subject: "medincident.users.v1.gender_changed",
			Message: &usersv1.UserGenderChanged{
				Gender: mapGender(*p.Gender),
			},
		})
	}

	return events
}

func mapGender(g int) usersv1.Gender {
	switch g {
	case 1:
		return usersv1.Gender_GENDER_MALE
	case 2:
		return usersv1.Gender_GENDER_FEMALE
	case 3:
		return usersv1.Gender_GENDER_DIVERSE
	default:
		return usersv1.Gender_GENDER_UNSPECIFIED
	}
}
