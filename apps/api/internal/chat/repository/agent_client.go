package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"teamflow/internal/chat"
	"teamflow/internal/faults"
)

// Agent-side error classification. The service maps these to public status
// codes; the raw upstream message is never forwarded to the browser.
//
// The unavailable/rejected split is not enough on its own: a connection that
// was never established is a known failure, while a request that was sent and
// then lost has an unknown outcome. CreateSession must distinguish them so it
// can mark the mapping failed instead of pending wherever that is justified.
var (
	ErrAgentUnavailable = errors.New("agent unavailable")
	ErrAgentTimeout     = errors.New("agent timeout")
	ErrAgentRejected    = errors.New("agent rejected the request")
	// ErrAgentUnreachable means the request provably never reached the Agent.
	ErrAgentUnreachable = errors.New("agent unreachable")
)

// maxAgentResponseBytes bounds every non-streaming Agent response body.
const maxAgentResponseBytes = 1 << 20

// AgentClient is the internal HTTP adapter in front of the FastAPI Agent
// runtime. It is the only place that knows Agent URLs, the internal token and
// the internal protocol shape (ADR-0003, backend standards §7).
type AgentClient struct {
	baseURL string
	token   string
	client  *http.Client
}

func NewAgentClient() *AgentClient {
	return &AgentClient{
		baseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("AGENT_BASE_URL")), "/"),
		token:   strings.TrimSpace(os.Getenv("INTERNAL_API_TOKEN")),
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
				TLSHandshakeTimeout: 5 * time.Second,
				MaxIdleConnsPerHost: 10,
			},
		},
	}
}

// AgentRunRequest is an alias kept for readability at call sites; the
// authoritative definition is the domain type in teamflow/internal/chat.
type AgentRunRequest = chat.RunRequest

func (c *AgentClient) CreateSession(ctx context.Context, title, idempotencyKey string) (string, error) {
	endpoint, err := c.endpoint("sessions")
	if err != nil {
		return "", err
	}
	body := strings.NewReader("{}")
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return "", fmt.Errorf("create agent session request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	response, err := c.do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if err := classify(response); err != nil {
		return "", err
	}
	var payload struct {
		ID json.Number `json:"id"`
	}
	if err := decodeLimited(response.Body, &payload); err != nil {
		return "", fmt.Errorf("decode agent session response: %w", err)
	}
	if payload.ID == "" {
		return "", fmt.Errorf("agent create session response has no id: %w", ErrAgentUnavailable)
	}
	return payload.ID.String(), nil
}

func (c *AgentClient) GetMessages(ctx context.Context, agentSessionID string) ([]chat.Message, error) {
	endpoint, err := c.endpoint("sessions", agentSessionID, "messages")
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("get agent messages request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if err := classify(response); err != nil {
		return nil, err
	}
	// The Agent serialises json columns itself; keep the payload verbatim so
	// citations stay structured instead of degrading to a string.
	var payload []struct {
		ID            json.Number `json:"id"`
		Role          string      `json:"role"`
		Content       *string     `json:"content"`
		Citations     any         `json:"citations"`
		Images        []string    `json:"images"`
		DecisionLogID json.Number `json:"decision_log_id"`
		CreatedAt     string      `json:"created_at"`
	}
	if err := decodeLimited(response.Body, &payload); err != nil {
		return nil, fmt.Errorf("decode agent messages response: %w", err)
	}
	messages := make([]chat.Message, 0, len(payload))
	for _, row := range payload {
		message := chat.Message{
			Role:      row.Role,
			Citations: row.Citations,
			Images:    row.Images,
		}
		if row.Content != nil {
			message.Content = *row.Content
		}
		// The Agent message id and decision log id are internal; only expose a
		// stable opaque reference for the decision log.
		if row.DecisionLogID != "" && row.DecisionLogID != "0" {
			id := row.DecisionLogID.String()
			message.DecisionLogID = &id
		}
		if row.CreatedAt != "" {
			if parsed, err := parseAgentTime(row.CreatedAt); err == nil {
				message.CreatedAt = parsed
			}
		}
		messages = append(messages, message)
	}
	return messages, nil
}

// StartRun opens the Agent SSE stream. The caller owns the returned body and
// must close it; cancellation propagates to the Agent through ctx.
func (c *AgentClient) StartRun(ctx context.Context, agentSessionID string, req AgentRunRequest) (io.ReadCloser, error) {
	endpoint, err := c.endpoint("sessions", agentSessionID, "ask")
	if err != nil {
		return nil, err
	}
	payload, err := json.Marshal(map[string]any{
		"question":          req.Question,
		"kb_ids":            req.KnowledgeBases,
		"images":            req.Images,
		"skip_user_persist": req.SkipUserPersist,
	})
	if err != nil {
		return nil, fmt.Errorf("encode agent run request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("create agent run request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream")
	// The internal token must be attached on this path too. StartRun cannot use
	// c.do because the streaming response needs a client without an overall
	// timeout, so the credential is set explicitly here.
	request.Header.Set("X-Agent-Token", c.token)
	streamClient := &http.Client{Transport: c.client.Transport}
	response, err := streamClient.Do(request)
	if err != nil {
		return nil, classifyTransport(err)
	}
	if err := classify(response); err != nil {
		response.Body.Close()
		return nil, err
	}
	return response.Body, nil
}

func (c *AgentClient) DeleteSession(ctx context.Context, agentSessionID string) error {
	endpoint, err := c.endpoint("sessions", agentSessionID)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, http.NoBody)
	if err != nil {
		return fmt.Errorf("create agent delete request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := c.do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if err := classify(response); err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, maxAgentResponseBytes))
	return nil
}

func (c *AgentClient) endpoint(segments ...string) (string, error) {
	if c.baseURL == "" {
		return "", fmt.Errorf("AGENT_BASE_URL is not configured: %w", ErrAgentUnavailable)
	}
	if c.token == "" {
		return "", fmt.Errorf("INTERNAL_API_TOKEN is not configured: %w", ErrAgentUnavailable)
	}
	parts := append([]string{c.baseURL, "/internal/rag/chat"}, segments...)
	endpoint, err := url.JoinPath(parts[0], parts[1:]...)
	if err != nil {
		return "", fmt.Errorf("build agent URL: %w", err)
	}
	return endpoint, nil
}

func (c *AgentClient) do(request *http.Request) (*http.Response, error) {
	request.Header.Set("X-Agent-Token", c.token)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, classifyTransport(err)
	}
	return response, nil
}

func classify(response *http.Response) error {
	switch {
	case response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices:
		return nil
	case response.StatusCode == http.StatusNotFound:
		return faults.ErrNotFound
	case response.StatusCode == http.StatusUnauthorized || response.StatusCode == http.StatusForbidden:
		return fmt.Errorf("agent rejected internal credentials (status %d): %w", response.StatusCode, ErrAgentUnavailable)
	case response.StatusCode >= http.StatusInternalServerError:
		return fmt.Errorf("agent returned %d: %w", response.StatusCode, ErrAgentUnavailable)
	default:
		return fmt.Errorf("agent rejected request (status %d): %w", response.StatusCode, ErrAgentRejected)
	}
}

// classifyTransport separates "never reached the Agent" from "response lost".
// A connection that was never established is a known failure; anything after
// the request was written may have been processed upstream, which the caller
// records as pending rather than retrying.
func classifyTransport(err error) error {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("agent request timed out: %w", ErrAgentTimeout)
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("agent request timed out: %w", ErrAgentTimeout)
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return fmt.Errorf("agent request dial failed: %w", ErrAgentUnreachable)
	}
	return fmt.Errorf("agent request failed: %w", ErrAgentUnavailable)
}

func decodeLimited(body io.Reader, target any) error {
	decoder := json.NewDecoder(io.LimitReader(body, maxAgentResponseBytes))
	return decoder.Decode(target)
}

// parseAgentTime accepts both RFC3339 and the Agent's MySQL datetime format.
func parseAgentTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported agent timestamp %q", value)
}
