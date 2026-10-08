package event

import (
	"encoding/json"
	"errors"
	"time"
)

const TaskAssignedV1 = "task.assigned.v1"

var ErrPermanent = errors.New("permanent error")

// Envelope is the stable wire format for all events.
type Envelope struct {
	Payload   json.RawMessage `json:"payLoad"`
	EventID   string          `json:"eventID"`
	EventType string          `json:"eventType"`
	CreatedAt time.Time       `json:"createdAt"`
	// OccurredAt is retained for compatibility with the original event model.
	OccurredAt    time.Time `json:"occurredAt,omitempty"`
	SchemaVersion string    `json:"schemaVersion"`
}
