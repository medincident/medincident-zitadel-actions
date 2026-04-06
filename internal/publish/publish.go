package publish

import (
	"context"

	"github.com/google/uuid"
	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/samber/oops"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	eventsv1 "github.com/medincident/medincident-zitadel-actions/pkg/medincident/events/v1"

	"github.com/medincident/medincident-zitadel-actions/internal/mapper"
)

// PublishEvents wraps each MappedEvent in a domain Envelope and publishes it to NATS JetStream.
func PublishEvents(ctx context.Context, js jetstream.JetStream, events []mapper.MappedEvent, aggregateID string) error {
	for _, event := range events {
		payload, err := anypb.New(event.Message)
		if err != nil {
			return oops.
				In("publish").
				Code("any_pack_failed").
				With("subject", event.Subject).
				Wrap(err)
		}

		envelope := &eventsv1.Envelope{
			EventId:       uuid.New().String(),
			OccurredAt:    timestamppb.Now(),
			AggregateType: "user",
			AggregateId:   aggregateID,
			CorrelationId: "",
			Payload:       payload,
		}

		data, err := proto.Marshal(envelope)
		if err != nil {
			return oops.
				In("publish").
				Code("marshal_failed").
				With("subject", event.Subject).
				With("event_id", envelope.EventId).
				Wrap(err)
		}

		msg := &nats.Msg{
			Subject: event.Subject,
			Data:    data,
			Header:  nats.Header{"Aggregate-Id": {aggregateID}},
		}

		if _, err := js.PublishMsg(ctx, msg); err != nil {
			return oops.
				In("publish").
				Code("nats_publish_failed").
				With("subject", event.Subject).
				With("aggregate_id", aggregateID).
				Wrap(err)
		}
	}

	return nil
}
