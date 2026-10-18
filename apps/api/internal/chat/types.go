package chat

import "time"

// Session is the platform-side mapping record. The Agent session id stays
// internal: only PublicID is exposed through the public API (ADR-0003).
type Session struct {
	PublicID       string    `json:"id"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
	OwnerID        string    `json:"owner_id"`
	AgentSessionID string    `json:"-"`
	LastErrorCode  string    `json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// RunRequest is the platform's ask payload. It lives with the domain types so
// both the application port and the Agent adapter can reference it without the
// adapter depending on the service package.
//
// SkipUserPersist is decided by the platform, never by the browser: it exists
// so regeneration paths do not duplicate the user turn, and it is not part of
// the public request body (ADR-0003).
type RunRequest struct {
	Question        string
	KnowledgeBases  []string
	Images          []string
	SkipUserPersist bool
}

// Lifecycle states of a platform session mapping (ADR-0003).
const (
	StatusCreating = "creating"
	StatusActive   = "active"
	StatusPending  = "pending"
	StatusFailed   = "failed"
	StatusDeleting = "deleting"
)

// Message is one entry of the Agent-owned conversation history, mapped into
// the platform's public shape. The numeric Agent message id is not exposed.
type Message struct {
	Role          string    `json:"role"`
	Content       string    `json:"content"`
	Citations     any       `json:"citations,omitempty"`
	Images        []string  `json:"images,omitempty"`
	DecisionLogID *string   `json:"decision_log_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
