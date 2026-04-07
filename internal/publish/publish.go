package publish

import (
	"context"

	"github.com/cenkalti/backoff/v4"
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"google.golang.org/protobuf/proto"

	"github.com/medincident/medincident-zitadel-actions/internal/config"
	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
)

// PublishEvents serializes each MappedEvent's Envelope and publishes it to
// NATS JetStream with exponential backoff retry.
func PublishEvents(ctx context.Context, logger *zerolog.Logger, js jetstream.JetStream, cfg config.PublishConfig, events []mapper.MappedEvent) error {
	for _, event := range events {
		data, err := proto.Marshal(event.Envelope)
		if err != nil {
			return oops.
				In("publish").
				Code("marshal_failed").
				With("subject", event.Subject).
				With("event_id", event.Envelope.GetEventId()).
				Wrap(err)
		}

		msg := &nats.Msg{
			Subject: event.Subject,
			Data:    data,
			Header: nats.Header{
				"Nats-Msg-Id":  {event.Envelope.GetEventId()},
				"Aggregate-Id": {event.Envelope.GetAggregateId()},
			},
		}

		b := backoff.NewExponentialBackOff()
		b.InitialInterval = cfg.InitialBackoff
		b.MaxInterval = cfg.MaxBackoff
		b.MaxElapsedTime = 0

		maxRetries := uint64(0)
		if cfg.MaxRetries > 0 {
			maxRetries = uint64(cfg.MaxRetries) //nolint:gosec // MaxRetries is validated positive by config defaults
		}
		retryable := backoff.WithMaxRetries(b, maxRetries)

		err = backoff.Retry(func() error {
			_, pubErr := js.PublishMsg(ctx, msg)
			return pubErr
		}, backoff.WithContext(retryable, ctx))

		if err != nil {
			logger.Warn().
				Str("subject", event.Subject).
				Str("event_id", event.Envelope.GetEventId()).
				Str("aggregate_id", event.Envelope.GetAggregateId()).
				Err(err).
				Msg("event publish failed after retries")

			return oops.
				In("publish").
				Code("nats_publish_failed").
				With("subject", event.Subject).
				With("event_id", event.Envelope.GetEventId()).
				With("aggregate_id", event.Envelope.GetAggregateId()).
				Wrap(err)
		}
	}

	return nil
}
