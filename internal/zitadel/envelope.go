// Package zitadel defines Go types that mirror the JSON shape of
// Zitadel Actions v2 webhook payloads. These types are the untyped
// transport layer; mapper converts them into the versioned protobuf
// messages that this service publishes on NATS.
//
// See https://zitadel.com/docs/guides/integrate/actions/testing-event
package zitadel

import (
	"time"
)

// Envelope is the generic wrapper Zitadel sends for every Actions v2
// event. T is the event-specific payload type, deserialized from the
// nested "event_payload" JSON object. All envelope-level fields (IDs,
// sequence, timestamps) are passthrough — this service does not rename
// or transform them before publishing.
type Envelope[T any] struct {
	AggregateID   string    `json:"aggregateID"`
	AggregateType string    `json:"aggregateType"`
	ResourceOwner string    `json:"resourceOwner"`
	InstanceID    string    `json:"instanceID"`
	Version       string    `json:"version"`
	Sequence      uint64    `json:"sequence"`
	EventType     string    `json:"event_type"`
	CreatedAt     time.Time `json:"created_at"`
	UserID        string    `json:"userID"`
	EventPayload  T         `json:"event_payload"`
}
