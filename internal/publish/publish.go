package publish

import (
	"context"

	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/oops"
	"google.golang.org/protobuf/proto"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
)

// PublishEvents serializes each MappedEvent's Envelope and publishes it to NATS JetStream.
func PublishEvents(ctx context.Context, js jetstream.JetStream, events []mapper.MappedEvent) error {
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

		if _, err := js.PublishMsg(ctx, msg); err != nil {
			return oops.
				In("publish").
				Code("nats_publish_failed").
				With("subject", event.Subject).
				With("aggregate_id", event.Envelope.GetAggregateId()).
				Wrap(err)
		}
	}

	return nil
}
