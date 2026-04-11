package zitadel_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
)

func TestEnvelopeHappyPath(t *testing.T) {
	const raw = `{
		"aggregateID":   "agg-123",
		"aggregateType": "user",
		"resourceOwner": "ro-456",
		"instanceID":    "inst-789",
		"version":       "v1",
		"sequence":      99,
		"event_type":    "user.human.added",
		"created_at":    "2026-04-11T12:34:56.789Z",
		"userID":        "user-001",
		"event_payload": {
			"userName":  "jdoe",
			"firstName": "John",
			"lastName":  "Doe",
			"email":     "john@example.com",
			"gender":    2
		}
	}`

	var env zitadel.Envelope[zitadel.UserHumanAdded]
	require.NoError(t, json.Unmarshal([]byte(raw), &env))

	assert.Equal(t, "agg-123", env.AggregateID)
	assert.Equal(t, "user", env.AggregateType)
	assert.Equal(t, "ro-456", env.ResourceOwner)
	assert.Equal(t, "inst-789", env.InstanceID)
	assert.Equal(t, "v1", env.Version)
	assert.Equal(t, uint64(99), env.Sequence)
	assert.Equal(t, "user.human.added", env.EventType)
	assert.False(t, env.CreatedAt.IsZero())
	assert.Equal(t, time.Date(2026, 4, 11, 12, 34, 56, 789_000_000, time.UTC), env.CreatedAt)
	assert.Equal(t, "user-001", env.UserID)

	assert.Equal(t, "jdoe", env.EventPayload.UserName)
	assert.Equal(t, "John", env.EventPayload.FirstName)
	assert.Equal(t, "Doe", env.EventPayload.LastName)
	assert.Equal(t, "john@example.com", env.EventPayload.Email)
	assert.Equal(t, 2, env.EventPayload.Gender)
}

func TestEnvelopePayloadLessEvent(t *testing.T) {
	const raw = `{
		"aggregateID":   "agg-999",
		"aggregateType": "user",
		"resourceOwner": "ro-111",
		"instanceID":    "inst-222",
		"version":       "v1",
		"sequence":      7,
		"event_type":    "user.human.email.verified",
		"created_at":    "2026-04-11T09:00:00Z",
		"userID":        "user-042"
	}`

	var env zitadel.Envelope[zitadel.UserHumanEmailVerified]
	require.NoError(t, json.Unmarshal([]byte(raw), &env))

	assert.Equal(t, "agg-999", env.AggregateID)
	assert.Equal(t, "user.human.email.verified", env.EventType)
	assert.Equal(t, "user-042", env.UserID)
	assert.Equal(t, zitadel.UserHumanEmailVerified{}, env.EventPayload)
}

func TestEnvelopeGenericParametrization(t *testing.T) {
	t.Run("UserHumanAdded", func(t *testing.T) {
		const raw = `{
			"aggregateID":   "agg-u1",
			"aggregateType": "user",
			"resourceOwner": "ro-u1",
			"instanceID":    "inst-u1",
			"version":       "v1",
			"sequence":      1,
			"event_type":    "user.human.added",
			"created_at":    "2026-04-11T08:00:00Z",
			"userID":        "user-u1",
			"event_payload": {
				"userName": "alice",
				"email":    "alice@example.com"
			}
		}`

		var env zitadel.Envelope[zitadel.UserHumanAdded]
		require.NoError(t, json.Unmarshal([]byte(raw), &env))

		assert.Equal(t, "alice", env.EventPayload.UserName)
		assert.Equal(t, "alice@example.com", env.EventPayload.Email)
	})

	t.Run("SessionUserChecked", func(t *testing.T) {
		const raw = `{
			"aggregateID":   "agg-s1",
			"aggregateType": "session",
			"resourceOwner": "ro-s1",
			"instanceID":    "inst-s1",
			"version":       "v1",
			"sequence":      2,
			"event_type":    "session.user.checked",
			"created_at":    "2026-04-11T08:00:00Z",
			"userID":        "",
			"event_payload": {
				"userID":            "user-s1",
				"userResourceOwner": "ro-s1",
				"checkedAt":         "2026-04-11T07:55:00Z",
				"preferredLanguage": "en"
			}
		}`

		lang := "en"

		var env zitadel.Envelope[zitadel.SessionUserChecked]
		require.NoError(t, json.Unmarshal([]byte(raw), &env))

		assert.Equal(t, "user-s1", env.EventPayload.UserID)
		assert.Equal(t, "ro-s1", env.EventPayload.UserResourceOwner)
		assert.False(t, env.EventPayload.CheckedAt.IsZero())
		assert.Equal(t, &lang, env.EventPayload.PreferredLanguage)
	})
}
