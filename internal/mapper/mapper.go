// Package mapper converts Zitadel webhook payloads into the versioned
// protobuf messages emitted on NATS. Mappers are intentionally thin:
// they copy and enum-translate, never reshape or filter.
package mapper

import (
	"encoding/json"

	"github.com/samber/oops"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventsv1 "github.com/medincident/medincident-zitadel-actions/pkg/events/v1"
	sessionsv1 "github.com/medincident/medincident-zitadel-actions/pkg/sessions/v1"
	usersv1 "github.com/medincident/medincident-zitadel-actions/pkg/users/v1"

	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

const (
	ErrCodeMapperParseRawBodyFailed = "parse_raw_body_failed"
	ErrCodeMapperAnyPBNewFailed     = "anypb_new_failed"
)

// profileFieldOrder pairs each Zitadel camelCase JSON key with its
// proto snake_case field name in proto declaration order. Iterating it
// yields deterministic FieldMask.Paths regardless of JSON object order.
var profileFieldOrder = []struct {
	jsonKey, protoField string
}{
	{"firstName", "first_name"},
	{"lastName", "last_name"},
	{"displayName", "display_name"},
	{"preferredLanguage", "preferred_language"},
	{"gender", "gender"},
	{"nickName", "nick_name"},
}

func BuildUserHumanAdded(env *zitadel.Envelope[zitadel.UserHumanAdded]) (*usersv1.UserHumanAdded, error) {
	p := &env.EventPayload

	msg := &usersv1.UserHumanAdded{
		UserName:          p.UserName,
		FirstName:         p.FirstName,
		LastName:          p.LastName,
		DisplayName:       p.DisplayName,
		Email:             p.Email,
		PreferredLanguage: p.PreferredLanguage,
		Gender:            mapGender(p.Gender),
	}

	if p.NickName != "" {
		msg.NickName = proto.String(p.NickName)
	}

	return msg, nil
}

// BuildUserHumanProfileChanged maps a profile-changed envelope. rawBody
// is the untouched request body: it is reparsed here to detect which
// event_payload keys were present, so that the output FieldMask lists
// exactly the fields the webhook reported as changed. The decoded env
// is still the source of values — rawBody is only used for key
// presence, not for deserialization.
func BuildUserHumanProfileChanged(
	rawBody []byte,
	env *zitadel.Envelope[zitadel.UserHumanProfileChanged],
) (*usersv1.UserHumanProfileChanged, error) {
	var parsed struct {
		EventPayload map[string]json.RawMessage `json:"event_payload"`
	}

	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return nil, oops.In("mapper").Code(ErrCodeMapperParseRawBodyFailed).Wrap(err)
	}

	var paths []string

	for _, f := range profileFieldOrder {
		if _, ok := parsed.EventPayload[f.jsonKey]; ok {
			paths = append(paths, f.protoField)
		}
	}

	p := &env.EventPayload

	msg := &usersv1.UserHumanProfileChanged{
		FirstName:         p.FirstName,
		LastName:          p.LastName,
		DisplayName:       p.DisplayName,
		PreferredLanguage: p.PreferredLanguage,
		Gender:            mapGender(p.Gender),
		UpdatedFields:     &fieldmaskpb.FieldMask{Paths: paths},
	}

	if p.NickName != nil {
		msg.NickName = p.NickName
	}

	return msg, nil
}

func BuildUserHumanEmailChanged(env *zitadel.Envelope[zitadel.UserHumanEmailChanged]) (*usersv1.UserHumanEmailChanged, error) {
	return &usersv1.UserHumanEmailChanged{
		Email: env.EventPayload.Email,
	}, nil
}

func BuildUserHumanEmailVerified(env *zitadel.Envelope[zitadel.UserHumanEmailVerified]) (*usersv1.UserHumanEmailVerified, error) {
	_ = env
	return &usersv1.UserHumanEmailVerified{}, nil
}

func BuildSessionAdded(env *zitadel.Envelope[zitadel.SessionAdded]) (*sessionsv1.SessionAdded, error) {
	p := &env.EventPayload

	msg := &sessionsv1.SessionAdded{}

	if p.UserAgent != nil {
		ua := &sessionsv1.UserAgent{
			Ip:      p.UserAgent.IP,
			Headers: make(map[string]*sessionsv1.HeaderValues, len(p.UserAgent.Header)),
		}

		for k, vals := range p.UserAgent.Header {
			ua.Headers[k] = &sessionsv1.HeaderValues{Values: vals}
		}

		if p.UserAgent.FingerprintID != nil {
			ua.FingerprintId = p.UserAgent.FingerprintID
		}

		if p.UserAgent.Description != nil {
			ua.Description = p.UserAgent.Description
		}

		msg.UserAgent = ua
	}

	return msg, nil
}

func BuildSessionUserChecked(env *zitadel.Envelope[zitadel.SessionUserChecked]) (*sessionsv1.SessionUserChecked, error) {
	p := &env.EventPayload

	msg := &sessionsv1.SessionUserChecked{
		UserId:            p.UserID,
		UserResourceOwner: p.UserResourceOwner,
		CheckedAt:         timestamppb.New(p.CheckedAt),
	}

	if p.PreferredLanguage != nil {
		msg.PreferredLanguage = proto.String(*p.PreferredLanguage)
	}

	return msg, nil
}

// BuildEnvelope wraps payload in a zitadel.events.v1.Envelope and
// copies every passthrough field from src. Production publishing goes
// through [publish.Publisher]; this helper exists for tests and any
// future caller that needs the wrapped envelope without publishing.
func BuildEnvelope[T any](src *zitadel.Envelope[T], payload proto.Message) (*eventsv1.Envelope, error) {
	pb, err := anypb.New(payload)
	if err != nil {
		return nil, oops.In("mapper").Code(ErrCodeMapperAnyPBNewFailed).Wrap(err)
	}

	return &eventsv1.Envelope{
		AggregateId:   src.AggregateID,
		AggregateType: src.AggregateType,
		ResourceOwner: src.ResourceOwner,
		InstanceId:    src.InstanceID,
		Version:       src.Version,
		Sequence:      src.Sequence,
		EventType:     src.EventType,
		CreatedAt:     timestamppb.New(src.CreatedAt),
		UserId:        src.UserID,
		Payload:       pb,
	}, nil
}

// mapGender translates Zitadel's internal Gender integer into the
// versioned proto enum. Values follow Zitadel's domain.Gender:
// 1=female, 2=male, 3=diverse; anything else maps to UNSPECIFIED.
func mapGender(g int) usersv1.Gender {
	switch g {
	case 1:
		return usersv1.Gender_GENDER_FEMALE
	case 2:
		return usersv1.Gender_GENDER_MALE
	case 3:
		return usersv1.Gender_GENDER_DIVERSE
	default:
		return usersv1.Gender_GENDER_UNSPECIFIED
	}
}
