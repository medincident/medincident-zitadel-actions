package publish

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventsv1 "github.com/medincident/medincident-zitadel-actions/gen/zitadel/events/v1"
	usersv1 "github.com/medincident/medincident-zitadel-actions/gen/zitadel/users/v1"
	"github.com/medincident/medincident-zitadel-actions/internal/config"
)

func TestDedupMsgID(t *testing.T) {
	got := dedupMsgID("inst-1", "user", "agg-2", 42)
	assert.Equal(t, "inst-1:user:agg-2:42", got)
}

type fakeJetStream struct {
	jetstream.JetStream
	lastSubject  string
	lastData     []byte
	lastOpts     []jetstream.PublishOpt
	errSequence  []error
	publishCount int
}

func (f *fakeJetStream) Publish(_ context.Context, subject string, data []byte, opts ...jetstream.PublishOpt) (*jetstream.PubAck, error) {
	f.publishCount++
	f.lastSubject = subject
	f.lastData = data
	f.lastOpts = opts
	if f.publishCount-1 < len(f.errSequence) {
		if err := f.errSequence[f.publishCount-1]; err != nil {
			return nil, err
		}
	}
	return &jetstream.PubAck{Stream: "zitadel", Sequence: uint64(f.publishCount)}, nil //nolint:gosec // publishCount is always small in tests
}

func testConfig() config.PublishConfig {
	return config.PublishConfig{
		MaxRetries:     3,
		InitialBackoff: config.Duration(1 * time.Millisecond),
		MaxBackoff:     config.Duration(5 * time.Millisecond),
		MaxElapsedTime: config.Duration(200 * time.Millisecond),
	}
}

func testLogger() *zerolog.Logger {
	l := zerolog.New(io.Discard)
	return &l
}

func testMetadata() *ZitadelMetadata {
	return &ZitadelMetadata{
		AggregateID:   "agg-1",
		AggregateType: "user",
		ResourceOwner: "org-1",
		InstanceID:    "inst-1",
		Version:       "v1",
		Sequence:      42,
		EventType:     "user.human.email.changed",
		CreatedAt:     timestamppb.New(time.Date(2026, 4, 11, 12, 0, 0, 0, time.UTC)),
		UserID:        "user-1",
	}
}

func TestPublishZitadelEventCallsJetstream(t *testing.T) {
	js := &fakeJetStream{}
	p := NewPublisher(testLogger(), js, testConfig())
	meta := testMetadata()

	payload := &usersv1.UserHumanEmailChanged{Email: "x@example.com"}
	err := p.PublishZitadelEvent(context.Background(), "zitadel.users.v1.human.email.changed", meta, payload)
	require.NoError(t, err)

	// jetstream.PublishOpt manipulates unexported pubOpts so we cannot
	// inspect the Nats-Msg-Id value in a unit test — the dedup key
	// itself is covered by TestDedupMsgID, and the end-to-end dedup
	// contract is covered by TestDedupByMsgID in the integration suite.
	// Here we only verify the publisher called Publish with the
	// expected subject and exactly one PublishOpt (the WithMsgID call).
	assert.Equal(t, "zitadel.users.v1.human.email.changed", js.lastSubject)
	assert.Len(t, js.lastOpts, 1)
}

func TestPublishZitadelEventWrapsPayloadInEnvelope(t *testing.T) {
	js := &fakeJetStream{}
	p := NewPublisher(testLogger(), js, testConfig())
	meta := testMetadata()

	payload := &usersv1.UserHumanEmailChanged{Email: "x@example.com"}
	err := p.PublishZitadelEvent(context.Background(), "zitadel.users.v1.human.email.changed", meta, payload)
	require.NoError(t, err)

	var env eventsv1.Envelope
	require.NoError(t, proto.Unmarshal(js.lastData, &env))

	assert.Equal(t, "agg-1", env.GetAggregateId())
	assert.Equal(t, "user", env.GetAggregateType())
	assert.Equal(t, "org-1", env.GetResourceOwner())
	assert.Equal(t, "inst-1", env.GetInstanceId())
	assert.Equal(t, "v1", env.GetVersion())
	assert.Equal(t, uint64(42), env.GetSequence())
	assert.Equal(t, "user.human.email.changed", env.GetEventType())
	assert.Equal(t, meta.CreatedAt.AsTime().UTC(), env.GetCreatedAt().AsTime().UTC())
	assert.Equal(t, "user-1", env.GetUserId())

	require.NotNil(t, env.GetPayload())
	var decoded usersv1.UserHumanEmailChanged
	require.NoError(t, env.GetPayload().UnmarshalTo(&decoded))
	assert.Equal(t, "x@example.com", decoded.GetEmail())
}

func TestPublishZitadelEventRetriesOnTransientError(t *testing.T) {
	transient := errors.New("transient nats failure")
	js := &fakeJetStream{errSequence: []error{transient, transient, nil}}
	p := NewPublisher(testLogger(), js, testConfig())
	meta := testMetadata()

	payload := &usersv1.UserHumanEmailChanged{Email: "x@example.com"}
	err := p.PublishZitadelEvent(context.Background(), "zitadel.users.v1.human.email.changed", meta, payload)
	require.NoError(t, err)
	assert.Equal(t, 3, js.publishCount)
}
