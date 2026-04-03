package events

import (
	"time"
)

// Envelope is the generic wrapper Zitadel sends for every Actions v2 event.
// The type parameter T represents the event-specific payload.
//
// Example raw JSON from Zitadel:
//
//	{
//	  "aggregateID":   "336494809936035843",
//	  "aggregateType": "user",
//	  "resourceOwner": "336392597046099971",
//	  "instanceID":    "336392597046034435",
//	  "version":       "v2",
//	  "sequence":      1,
//	  "event_type":    "user.human.added",
//	  "created_at":    "2025-09-05T08:55:36.156333Z",
//	  "userID":        "336392597046755331",
//	  "event_payload": { ... }
//	}
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
	// EventPayload is the event-specific payload, deserialized into T.
	EventPayload T `json:"event_payload"`
}
