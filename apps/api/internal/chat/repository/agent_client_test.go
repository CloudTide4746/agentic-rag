package repository

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"teamflow/internal/chat"
	"teamflow/internal/faults"
)

// newTestClient points the adapter at a stub Agent. The env vars are the only
// way the adapter is configured, so the test sets them per case (t.Setenv
// restores them afterwards).
func newTestClient(t *testing.T, handler http.HandlerFunc) *AgentClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	t.Setenv("AGENT_BASE_URL", server.URL)
	t.Setenv("INTERNAL_API_TOKEN", "test-token")
	return NewAgentClient()
}

func TestAgentClientSendsInternalTokenAndIdempotencyKey(t *testing.T) {
	var gotToken, gotKey, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Agent-Token")
		gotKey = r.Header.Get("Idempotency-Key")
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id": 42, "title": "x"}`))
	})

	id, err := client.CreateSession(context.Background(), "title", "idem-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "42" {
		t.Fatalf("agent session id = %q, want 42", id)
	}
	if gotToken != "test-token" {
		t.Fatalf("X-Agent-Token = %q, want the configured token", gotToken)
	}
	if gotKey != "idem-1" {
		t.Fatalf("Idempotency-Key = %q, want idem-1", gotKey)
	}
	if gotPath != "/internal/rag/chat/sessions" {
		t.Fatalf("path = %q, want the internal prefix", gotPath)
	}
}

func TestAgentClientRejectsMissingAgentSessionID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"title": "no id here"}`))
	})
	_, err := client.CreateSession(context.Background(), "title", "idem-1")
	if !errors.Is(err, ErrAgentUnavailable) {
		t.Fatalf("error = %v, want agent unavailable", err)
	}
}

func TestAgentClientMapsNotFound(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	_, err := client.GetMessages(context.Background(), "42")
	if !errors.Is(err, faults.ErrNotFound) {
		t.Fatalf("error = %v, want not found", err)
	}
}

func TestAgentClientMapsExplicitRejection(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	})
	_, err := client.CreateSession(context.Background(), "title", "idem-1")
	if !errors.Is(err, ErrAgentRejected) {
		t.Fatalf("error = %v, want rejected", err)
	}
}

func TestAgentClientMapsUnauthorizedAsUnavailable(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	// A bad internal credential is an operator problem, not a caller problem:
	// it must not surface as an authentication failure to the browser.
	_, err := client.CreateSession(context.Background(), "title", "idem-1")
	if !errors.Is(err, ErrAgentUnavailable) {
		t.Fatalf("error = %v, want agent unavailable", err)
	}
}

func TestAgentClientClassifiesDialFailureAsUnreachable(t *testing.T) {
	// Reserve a port and close it so the dial is refused rather than hanging.
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL
	server.Close()
	t.Setenv("AGENT_BASE_URL", baseURL)
	t.Setenv("INTERNAL_API_TOKEN", "test-token")

	_, err := NewAgentClient().CreateSession(context.Background(), "title", "idem-1")
	if !errors.Is(err, ErrAgentUnreachable) {
		t.Fatalf("error = %v, want unreachable", err)
	}
}

func TestAgentClientRequiresConfiguration(t *testing.T) {
	t.Setenv("AGENT_BASE_URL", "")
	t.Setenv("INTERNAL_API_TOKEN", "")
	_, err := NewAgentClient().CreateSession(context.Background(), "title", "idem-1")
	if !errors.Is(err, ErrAgentUnavailable) {
		t.Fatalf("error = %v, want agent unavailable", err)
	}
}

func TestAgentClientDecodesMessages(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/rag/chat/sessions/42/messages" {
			t.Errorf("path = %q, want the session messages path", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id": 7, "session_id": 42, "role": "user", "content": "hi", "images": ["/api/chat/images/a.png"], "created_at": "2026-10-16T10:00:00"},
			{"id": 8, "session_id": 42, "role": "assistant", "content": "hello",
			 "citations": [{"doc_name": "d", "chunk_no": 3}], "decision_log_id": 5, "created_at": "2026-10-16T10:00:05"}
		]`))
	})

	messages, err := client.GetMessages(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(messages))
	}
	if messages[0].Role != "user" || messages[0].Content != "hi" {
		t.Fatalf("first message = %+v, want the user turn", messages[0])
	}
	if len(messages[0].Images) != 1 {
		t.Fatalf("images = %v, want one entry", messages[0].Images)
	}
	// Citations must stay structured rather than collapsing to a string.
	citations, ok := messages[1].Citations.([]any)
	if !ok || len(citations) != 1 {
		t.Fatalf("citations = %#v, want a one-element array", messages[1].Citations)
	}
	if messages[1].DecisionLogID == nil || *messages[1].DecisionLogID != "5" {
		t.Fatalf("decision log id = %v, want 5", messages[1].DecisionLogID)
	}
	if messages[1].CreatedAt.IsZero() {
		t.Fatal("created_at was not parsed")
	}
}

func TestAgentClientOmitsZeroDecisionLogID(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"id": 7, "role": "user", "content": "hi", "decision_log_id": 0}]`))
	})
	messages, err := client.GetMessages(context.Background(), "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if messages[0].DecisionLogID != nil {
		t.Fatalf("decision log id = %v, want nil for the zero value", *messages[0].DecisionLogID)
	}
}

func TestAgentClientStartRunForwardsRunRequest(t *testing.T) {
	var body string
	var accept string
	var token string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		accept = r.Header.Get("Accept")
		token = r.Header.Get("X-Agent-Token")
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("event: done\ndata: {}\n\n"))
	})

	stream, err := client.StartRun(context.Background(), "42", chat.RunRequest{
		Question:       "hi",
		KnowledgeBases: []string{"3"},
		Images:         []string{"data:image/png;base64,AAA"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stream.Close()
	if accept != "text/event-stream" {
		t.Fatalf("Accept = %q, want text/event-stream", accept)
	}
	// StartRun builds its own client, so it must still attach the internal
	// credential; a missing token makes every ask fail at the Agent.
	if token != "test-token" {
		t.Fatalf("X-Agent-Token = %q, want the configured token", token)
	}
	for _, want := range []string{`"question":"hi"`, `"kb_ids":["3"]`, `"skip_user_persist":false`} {
		if !strings.Contains(body, want) {
			t.Fatalf("payload %s missing %s", body, want)
		}
	}
	payload, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if !strings.Contains(string(payload), "event: done") {
		t.Fatalf("stream = %q, want the forwarded frames", payload)
	}
}

// TestAgentClientStartRunRequiresToken guards the credential on the streaming
// path specifically: StartRun cannot reuse the shared do() helper.
func TestAgentClientStartRunRejectsMissingToken(t *testing.T) {
	t.Setenv("AGENT_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("INTERNAL_API_TOKEN", "")
	_, err := NewAgentClient().StartRun(context.Background(), "42", chat.RunRequest{Question: "hi"})
	if !errors.Is(err, ErrAgentUnavailable) {
		t.Fatalf("error = %v, want agent unavailable", err)
	}
}
