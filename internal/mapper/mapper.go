package mapper

import (
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventsv1 "github.com/medincident/medincident-zitadel-actions/pkg/medincident/events/v1"
	usersv1 "github.com/medincident/medincident-zitadel-actions/pkg/medincident/users/v1"

	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// MappedEvent pairs a NATS subject with a ready-to-publish proto Envelope.
type MappedEvent struct {
	Subject  string
	Envelope *eventsv1.Envelope
}

// MapUserHumanAdded converts a Zitadel UserHumanAdded event into domain proto events.
func MapUserHumanAdded(envelope *zitadel.Envelope[zitadel.UserHumanAdded]) ([]MappedEvent, error) {
	p := &envelope.EventPayload

	event, err := newUserMappedEvent("medincident.users.v1.created", envelope.UserID, &usersv1.UserCreated{
		Username:          p.UserName,
		FirstName:         p.FirstName,
		LastName:          p.LastName,
		DisplayName:       p.DisplayName,
		NickName:          p.NickName,
		Gender:            mapGender(p.Gender),
		Email:             p.Email,
		PreferredLanguage: p.PreferredLanguage,
	})
	if err != nil {
		return nil, err
	}

	return []MappedEvent{event}, nil
}

// MapUserHumanProfileChanged converts a Zitadel profile.changed event
// into zero or more domain proto events, based on which fields are non-nil.
func MapUserHumanProfileChanged(envelope *zitadel.Envelope[zitadel.UserHumanProfileChanged]) ([]MappedEvent, error) {
	p := &envelope.EventPayload
	var events []MappedEvent

	if p.FirstName != nil || p.LastName != nil || p.DisplayName != nil || p.NickName != nil {
		event, err := newUserMappedEvent("medincident.users.v1.name_changed", envelope.UserID, &usersv1.UserNameChanged{
			FirstName:   p.FirstName,
			LastName:    p.LastName,
			DisplayName: p.DisplayName,
			NickName:    p.NickName,
		})
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if p.PreferredLanguage != nil {
		event, err := newUserMappedEvent("medincident.users.v1.language_changed", envelope.UserID, &usersv1.UserLanguageChanged{
			PreferredLanguage: *p.PreferredLanguage,
		})
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	if p.Gender != nil {
		event, err := newUserMappedEvent("medincident.users.v1.gender_changed", envelope.UserID, &usersv1.UserGenderChanged{
			Gender: mapGender(*p.Gender),
		})
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, nil
}

// newUserMappedEvent constructs a MappedEvent with a fully populated eventsv1.Envelope.
func newUserMappedEvent(subject, aggregateID string, payload proto.Message) (MappedEvent, error) {
	pb, err := anypb.New(payload)
	if err != nil {
		return MappedEvent{}, err
	}

	return MappedEvent{
		Subject: subject,
		Envelope: &eventsv1.Envelope{
			EventId:       uuid.New().String(),
			OccurredAt:    timestamppb.Now(),
			AggregateType: "user",
			AggregateId:   aggregateID,
			Payload:       pb,
		},
	}, nil
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
