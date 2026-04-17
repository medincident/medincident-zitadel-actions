// Package publish wraps Zitadel webhook payloads in the versioned
// zitadel.events.v1.Envelope protobuf and publishes them on NATS
// JetStream with deduplication and exponential-backoff retry.
package publish

import (
	"context"
	"fmt"

	"github.com/cenkalti/backoff/v4"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/zitadel"
	eventsv1 "github.com/medincident/medincident-zitadel-actions/pkg/events/v1"
)

const (
	ErrCodePublishAnyPBNewFailed    = "anypb_new_failed"
	ErrCodePublishMarshalFailed     = "marshal_failed"
	ErrCodePublishNATSPublishFailed = "nats_publish_failed"
)

// Publisher publishes Zitadel events to NATS JetStream with
// exponential-backoff retry. It holds no mutable state and is safe for
// concurrent use.
type Publisher struct {
	logger *zerolog.Logger
	js     jetstream.JetStream
	cfg    config.PublishConfig
}

func NewPublisher(logger *zerolog.Logger, js jetstream.JetStream, cfg config.PublishConfig) *Publisher {
	return &Publisher{logger: logger, js: js, cfg: cfg}
}

// ZitadelMetadata is the envelope-level subset of a Zitadel webhook that
// Publisher copies onto the outbound protobuf envelope.
type ZitadelMetadata struct {
	AggregateID   string
	AggregateType string
	ResourceOwner string
	InstanceID    string
	Version       string
	Sequence      uint64
	EventType     string
	UserID        string
	CreatedAt     *timestamppb.Timestamp
}

func FromZitadelEnvelope[T any](e *zitadel.Envelope[T]) *ZitadelMetadata {
	return &ZitadelMetadata{
		AggregateID:   e.AggregateID,
		AggregateType: e.AggregateType,
		ResourceOwner: e.ResourceOwner,
		InstanceID:    e.InstanceID,
		Version:       e.Version,
		Sequence:      e.Sequence,
		EventType:     e.EventType,
		UserID:        e.UserID,
		CreatedAt:     timestamppb.New(e.CreatedAt),
	}
}

// dedupMsgID builds the Nats-Msg-Id used for JetStream deduplication:
// "{instance}:{aggregate_type}:{aggregate_id}:{sequence}". Zitadel
// guarantees sequence monotonicity per aggregate, so retries of the
// same event collapse to a single stored message within the stream's
// duplicate window.
func dedupMsgID(instanceID, aggregateType, aggregateID string, sequence uint64) string {
	return fmt.Sprintf("%s:%s:%s:%d", instanceID, aggregateType, aggregateID, sequence)
}

// PublishZitadelEvent wraps payload in a zitadel.events.v1.Envelope,
// marshals it and publishes it on subject with the dedup key set via
// [jetstream.WithMsgID]. Transient NATS errors are retried with
// exponential backoff bounded by config.PublishConfig.
//
// See https://docs.nats.io/using-nats/developer/develop_jetstream/model_deep_dive
func (p *Publisher) PublishZitadelEvent(ctx context.Context, subject string, meta *ZitadelMetadata, payload proto.Message) error {
	pb, err := anypb.New(payload)
	if err != nil {
		return oops.In("publish").Code(ErrCodePublishAnyPBNewFailed).With("subject", subject).Wrap(err)
	}

	envelope := &eventsv1.Envelope{
		AggregateId:   meta.AggregateID,
		AggregateType: meta.AggregateType,
		ResourceOwner: meta.ResourceOwner,
		InstanceId:    meta.InstanceID,
		Version:       meta.Version,
		Sequence:      meta.Sequence,
		EventType:     meta.EventType,
		CreatedAt:     meta.CreatedAt,
		UserId:        meta.UserID,
		Payload:       pb,
	}

	data, err := proto.Marshal(envelope)
	if err != nil {
		return oops.
			In("publish").
			Code(ErrCodePublishMarshalFailed).
			With("subject", subject).
			With("aggregate_id", meta.AggregateID).
			Wrap(err)
	}

	msgID := dedupMsgID(meta.InstanceID, meta.AggregateType, meta.AggregateID, meta.Sequence)

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = p.cfg.InitialBackoff
	b.MaxInterval = p.cfg.MaxBackoff
	b.MaxElapsedTime = p.cfg.MaxElapsedTime

	maxRetries := uint64(0)
	if p.cfg.MaxRetries > 0 {
		maxRetries = uint64(p.cfg.MaxRetries)
	}
	retryable := backoff.WithMaxRetries(b, maxRetries)

	err = backoff.Retry(func() error {
		_, pubErr := p.js.Publish(ctx, subject, data, jetstream.WithMsgID(msgID))
		return pubErr
	}, backoff.WithContext(retryable, ctx))
	if err != nil {
		p.logger.Warn().
			Str("subject", subject).
			Str("aggregate_id", meta.AggregateID).
			Uint64("sequence", meta.Sequence).
			Err(err).
			Msg("event publish failed after retries")

		return oops.
			In("publish").
			Code(ErrCodePublishNATSPublishFailed).
			With("subject", subject).
			With("aggregate_id", meta.AggregateID).
			With("sequence", meta.Sequence).
			Wrap(err)
	}

	p.logger.Info().
		Str("subject", subject).
		Str("aggregate_id", meta.AggregateID).
		Uint64("sequence", meta.Sequence).
		Msg("published zitadel event")

	return nil
}
