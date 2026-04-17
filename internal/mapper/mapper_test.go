package mapper_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
	usersv1 "github.com/medincident/medincident-zitadel-actions/pkg/users/v1"
)

var testCreatedAt = time.Date(2026, 4, 11, 10, 0, 0, 0, time.UTC)

func baseEnvelope[T any](payload T) *zitadel.Envelope[T] {
	return &zitadel.Envelope[T]{
		AggregateID:   "agg-id",
		AggregateType: "user",
		ResourceOwner: "ro-id",
		InstanceID:    "inst-id",
		Version:       "v1",
		Sequence:      42,
		EventType:     "user.human.added",
		CreatedAt:     testCreatedAt,
		UserID:        "user-id",
		EventPayload:  payload,
	}
}

func TestBuildUserHumanAdded(t *testing.T) {
	env := baseEnvelope(zitadel.UserHumanAdded{
		UserName:          "jdoe",
		FirstName:         "John",
		LastName:          "Doe",
		NickName:          "JD",
		DisplayName:       "John Doe",
		PreferredLanguage: "en",
		Gender:            2,
		Email:             "john@example.com",
	})

	msg, err := mapper.BuildUserHumanAdded(env)
	require.NoError(t, err)

	assert.Equal(t, "jdoe", msg.GetUserName())
	assert.Equal(t, "John", msg.GetFirstName())
	assert.Equal(t, "Doe", msg.GetLastName())
	assert.Equal(t, "John Doe", msg.GetDisplayName())
	assert.Equal(t, "john@example.com", msg.GetEmail())
	assert.Equal(t, "en", msg.GetPreferredLanguage())
	assert.Equal(t, usersv1.Gender_GENDER_MALE, msg.GetGender())
	assert.Equal(t, "JD", msg.GetNickName())
	require.NotNil(t, msg.NickName)
}

func TestBuildUserHumanProfileChangedFullJSON(t *testing.T) {
	nick := "JD"
	env := baseEnvelope(zitadel.UserHumanProfileChanged{
		FirstName:         "John",
		LastName:          "Doe",
		NickName:          &nick,
		DisplayName:       "John Doe",
		PreferredLanguage: "en",
		Gender:            1,
	})
	env.EventType = "user.human.profile.changed"

	rawBody, err := json.Marshal(map[string]any{
		"event_payload": map[string]any{
			"firstName":         "John",
			"lastName":          "Doe",
			"nickName":          "JD",
			"displayName":       "John Doe",
			"preferredLanguage": "en",
			"gender":            1,
		},
	})
	require.NoError(t, err)

	msg, err := mapper.BuildUserHumanProfileChanged(rawBody, env)
	require.NoError(t, err)

	assert.Equal(t, "John", msg.GetFirstName())
	assert.Equal(t, "Doe", msg.GetLastName())
	assert.Equal(t, "John Doe", msg.GetDisplayName())
	assert.Equal(t, "en", msg.GetPreferredLanguage())
	assert.Equal(t, usersv1.Gender_GENDER_FEMALE, msg.GetGender())
	assert.Equal(t, "JD", msg.GetNickName())

	require.NotNil(t, msg.UpdatedFields)
	paths := msg.UpdatedFields.GetPaths()
	assert.Len(t, paths, 6)
	assert.ElementsMatch(t, []string{
		"first_name", "last_name", "display_name",
		"preferred_language", "gender", "nick_name",
	}, paths)
}

func TestBuildUserHumanProfileChangedPartial(t *testing.T) {
	nick := "JD"
	env := baseEnvelope(zitadel.UserHumanProfileChanged{
		NickName: &nick,
	})
	env.EventType = "user.human.profile.changed"

	rawBody, err := json.Marshal(map[string]any{
		"event_payload": map[string]any{
			"nickName": "JD",
		},
	})
	require.NoError(t, err)

	msg, err := mapper.BuildUserHumanProfileChanged(rawBody, env)
	require.NoError(t, err)

	require.NotNil(t, msg.UpdatedFields)
	assert.Equal(t, []string{"nick_name"}, msg.UpdatedFields.GetPaths())
}

func TestBuildUserHumanProfileChangedNickNameCleared(t *testing.T) {
	env := baseEnvelope(zitadel.UserHumanProfileChanged{
		NickName: nil, // JSON null
	})
	env.EventType = "user.human.profile.changed"

	rawBody, err := json.Marshal(map[string]any{
		"event_payload": map[string]any{
			"nickName": nil,
		},
	})
	require.NoError(t, err)

	msg, err := mapper.BuildUserHumanProfileChanged(rawBody, env)
	require.NoError(t, err)

	require.NotNil(t, msg.UpdatedFields)
	assert.Contains(t, msg.UpdatedFields.GetPaths(), "nick_name")
	assert.Nil(t, msg.NickName)
}

func TestBuildUserHumanEmailChanged(t *testing.T) {
	env := baseEnvelope(zitadel.UserHumanEmailChanged{Email: "new@example.com"})
	env.EventType = "user.human.email.changed"

	msg, err := mapper.BuildUserHumanEmailChanged(env)
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", msg.GetEmail())
}

func TestBuildUserHumanEmailVerified(t *testing.T) {
	env := baseEnvelope(zitadel.UserHumanEmailVerified{})
	env.EventType = "user.human.email.verified"

	msg, err := mapper.BuildUserHumanEmailVerified(env)
	require.NoError(t, err)
	require.NotNil(t, msg)
}

func TestBuildSessionAddedWithUserAgent(t *testing.T) {
	fp := "fp-abc"
	desc := "Chrome on macOS"
	env := baseEnvelope(zitadel.SessionAdded{
		UserAgent: &zitadel.SessionUserAgent{
			FingerprintID: &fp,
			IP:            "1.2.3.4",
			Description:   &desc,
			Header: map[string][]string{
				"Accept":          {"text/html", "application/json"},
				"Accept-Language": {"en-US"},
			},
		},
	})
	env.AggregateType = "session"
	env.EventType = "session.added"

	msg, err := mapper.BuildSessionAdded(env)
	require.NoError(t, err)
	require.NotNil(t, msg.UserAgent)

	assert.Equal(t, "1.2.3.4", msg.UserAgent.GetIp())
	assert.Equal(t, "fp-abc", msg.UserAgent.GetFingerprintId())
	assert.Equal(t, "Chrome on macOS", msg.UserAgent.GetDescription())

	hdrs := msg.UserAgent.GetHeaders()
	require.Contains(t, hdrs, "Accept")
	assert.ElementsMatch(t, []string{"text/html", "application/json"}, hdrs["Accept"].GetValues())
	require.Contains(t, hdrs, "Accept-Language")
	assert.ElementsMatch(t, []string{"en-US"}, hdrs["Accept-Language"].GetValues())
}

func TestBuildSessionAddedWithoutUserAgent(t *testing.T) {
	env := baseEnvelope(zitadel.SessionAdded{UserAgent: nil})
	env.AggregateType = "session"
	env.EventType = "session.added"

	msg, err := mapper.BuildSessionAdded(env)
	require.NoError(t, err)
	assert.Nil(t, msg.UserAgent)
}

func TestBuildSessionUserChecked(t *testing.T) {
	lang := "de"
	checkedAt := time.Date(2026, 4, 11, 9, 30, 0, 0, time.UTC)
	env := baseEnvelope(zitadel.SessionUserChecked{
		UserID:            "user-42",
		UserResourceOwner: "ro-42",
		CheckedAt:         checkedAt,
		PreferredLanguage: &lang,
	})
	env.AggregateType = "session"
	env.EventType = "session.user.checked"

	msg, err := mapper.BuildSessionUserChecked(env)
	require.NoError(t, err)

	assert.Equal(t, "user-42", msg.GetUserId())
	assert.Equal(t, "ro-42", msg.GetUserResourceOwner())
	require.NotNil(t, msg.CheckedAt)
	assert.Equal(t, checkedAt.Unix(), msg.CheckedAt.AsTime().Unix())
	assert.Equal(t, "de", msg.GetPreferredLanguage())
	require.NotNil(t, msg.PreferredLanguage)
}

func TestBuildEnvelope(t *testing.T) {
	env := &zitadel.Envelope[zitadel.UserHumanAdded]{
		AggregateID:   "agg-111",
		AggregateType: "user",
		ResourceOwner: "ro-222",
		InstanceID:    "inst-333",
		Version:       "v1",
		Sequence:      99,
		EventType:     "user.human.added",
		CreatedAt:     testCreatedAt,
		UserID:        "user-444",
	}

	payload := &usersv1.UserHumanAdded{
		UserName:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
	}

	envelope, err := mapper.BuildEnvelope(env, payload)
	require.NoError(t, err)
	require.NotNil(t, envelope)

	assert.Equal(t, "agg-111", envelope.GetAggregateId())
	assert.Equal(t, "user", envelope.GetAggregateType())
	assert.Equal(t, "ro-222", envelope.GetResourceOwner())
	assert.Equal(t, "inst-333", envelope.GetInstanceId())
	assert.Equal(t, "v1", envelope.GetVersion())
	assert.Equal(t, uint64(99), envelope.GetSequence())
	assert.Equal(t, "user.human.added", envelope.GetEventType())
	assert.Equal(t, "user-444", envelope.GetUserId())
	require.NotNil(t, envelope.GetCreatedAt())
	assert.Equal(t, testCreatedAt.Unix(), envelope.GetCreatedAt().AsTime().Unix())

	require.NotNil(t, envelope.GetPayload())
	unpacked := &usersv1.UserHumanAdded{}
	require.NoError(t, envelope.GetPayload().UnmarshalTo(unpacked))
	assert.Equal(t, "testuser", unpacked.GetUserName())

	// Verify it is a valid proto.Message
	_ = proto.Size(envelope)
}
