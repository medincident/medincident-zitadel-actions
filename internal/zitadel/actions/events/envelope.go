package events

import (
	"encoding/json"
	"time"
)

// Envelope is the wrapper Zitadel sends for every Actions v2 event.
// See: https://zitadel.com/docs/guides/integrate/actions/usage#sent-information-event
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
//	  "event_payload": "<base64-encoded JSON>"
//	}
type Envelope struct {
	AggregateID   string    `json:"aggregateID"`
	AggregateType string    `json:"aggregateType"`
	ResourceOwner string    `json:"resourceOwner"`
	InstanceID    string    `json:"instanceID"`
	Version       string    `json:"version"`
	Sequence      uint64    `json:"sequence"`
	EventType     string    `json:"event_type"`
	CreatedAt     time.Time `json:"created_at"`
	UserID        string    `json:"userID"`
	// EventPayload holds the raw JSON body of the event.
	// Use Unmarshal to decode it into a typed struct.
	EventPayload json.RawMessage `json:"event_payload"`
}

// Unmarshal decodes the JSON EventPayload into output.
func Unmarshal[T any](envelope *Envelope, output *T) error {
	return json.Unmarshal(envelope.EventPayload, output)
}
