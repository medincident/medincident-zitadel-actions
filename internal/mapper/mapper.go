package mapper

import (
	"encoding/json"

	"github.com/samber/oops"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventsv1 "github.com/medincident/medincident-zitadel-actions/gen/zitadel/events/v1"
	sessionsv1 "github.com/medincident/medincident-zitadel-actions/gen/zitadel/sessions/v1"
	usersv1 "github.com/medincident/medincident-zitadel-actions/gen/zitadel/users/v1"

	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

// Error codes emitted by this mapper. Declared at file level so each emit site
// is grep-local; string values carry the mapper name so interceptors can
// distinguish per-mapper telemetry.
const (
	ErrCodeMapperParseRawBodyFailed = "parse_raw_body_failed"
	ErrCodeMapperAnyPBNewFailed     = "anypb_new_failed"
)

// profileFieldOrder maps Zitadel camelCase JSON keys to proto snake_case field names
// in proto declaration order, ensuring deterministic FieldMask.Paths across calls.
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

// BuildUserHumanAdded converts a Zitadel UserHumanAdded event into a proto message.
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

// BuildUserHumanProfileChanged converts a Zitadel UserHumanProfileChanged event into a proto message.
// rawBody is the raw HTTP request body used for FieldMask building via JSON key presence detection.
func BuildUserHumanProfileChanged(
	rawBody []byte,
	env *zitadel.Envelope[zitadel.UserHumanProfileChanged],
) (*usersv1.UserHumanProfileChanged, error) {
	// Parse raw body to detect which keys are actually present.
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

// BuildUserHumanEmailChanged converts a Zitadel UserHumanEmailChanged event into a proto message.
func BuildUserHumanEmailChanged(env *zitadel.Envelope[zitadel.UserHumanEmailChanged]) (*usersv1.UserHumanEmailChanged, error) {
	return &usersv1.UserHumanEmailChanged{
		Email: env.EventPayload.Email,
	}, nil
}

// BuildUserHumanEmailVerified converts a Zitadel UserHumanEmailVerified event into a proto message.
func BuildUserHumanEmailVerified(env *zitadel.Envelope[zitadel.UserHumanEmailVerified]) (*usersv1.UserHumanEmailVerified, error) {
	_ = env
	return &usersv1.UserHumanEmailVerified{}, nil
}

// BuildSessionAdded converts a Zitadel SessionAdded event into a proto message.
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

// BuildSessionUserChecked converts a Zitadel SessionUserChecked event into a proto message.
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

// BuildEnvelope wraps a proto payload in a zitadel.events.v1.Envelope,
// copying all fields from the Zitadel webhook envelope.
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

// mapGender converts a Zitadel gender integer (internal/domain.Gender) into a proto Gender enum.
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
