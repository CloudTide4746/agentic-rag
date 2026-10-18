package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"

	"teamflow/internal/chat"
	"teamflow/internal/chat/handler"
	"teamflow/internal/chat/repository"
	"teamflow/internal/faults"
)

const (
	// maxQuestionRunes bounds a single question before it reaches the Agent.
	maxQuestionRunes = 4000
	// maxTitleRunes bounds the platform's display title.
	maxTitleRunes = 128
	// maxImages matches the Agent's multimodal limit of four attachments.
	maxImages = 4
)

// SessionStore is the platform-owned mapping table port.
type SessionStore interface {
	Create(context.Context, chat.Session, string) (*chat.Session, error)
	GetByPublicID(context.Context, string, uint) (*chat.Session, error)
	ListByOwner(context.Context, uint, int, int) ([]chat.Session, int64, error)
	MarkActive(context.Context, string, string) (*chat.Session, error)
	MarkFailed(context.Context, string, string) error
	MarkPending(context.Context, string, string) error
}

// AgentClient is the internal Agent runtime port.
type AgentClient interface {
	CreateSession(ctx context.Context, title, idempotencyKey string) (string, error)
	GetMessages(ctx context.Context, agentSessionID string) ([]chat.Message, error)
	StartRun(ctx context.Context, agentSessionID string, req chat.RunRequest) (io.ReadCloser, error)
}

type ChatService struct {
	sessions SessionStore
	agent    AgentClient
}

var _ handler.ChatHandler = (*ChatService)(nil)

func NewChatService(sessions SessionStore, agent AgentClient) *ChatService {
	return &ChatService{sessions: sessions, agent: agent}
}

// CreateChat creates the platform mapping first, then the Agent session, so a
// confirmed Agent session is never left without platform ownership
// (ADR-0003 "创建会话流程").
func (s *ChatService) CreateChat(ctx context.Context, req *handler.CreateChatReq) (*handler.CreateChatResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "新会话"
	}
	if utf8.RuneCountInString(title) > maxTitleRunes {
		return nil, fmt.Errorf("title is too long: %w", faults.ErrValidation)
	}
	publicID, err := newPublicID()
	if err != nil {
		return nil, err
	}
	idempotencyKey, err := newPublicID()
	if err != nil {
		return nil, err
	}
	if _, err := s.sessions.Create(ctx, chat.Session{
		PublicID: publicID,
		OwnerID:  strconv.FormatUint(uint64(req.ActorID), 10),
		Title:    title,
		Status:   chat.StatusCreating,
	}, idempotencyKey); err != nil {
		return nil, err
	}

	agentSessionID, err := s.agent.CreateSession(ctx, title, idempotencyKey)
	if err != nil {
		s.recordCreationFailure(ctx, publicID, err)
		return nil, err
	}
	// Report the committed state rather than the pre-update snapshot.
	activated, err := s.sessions.MarkActive(ctx, publicID, agentSessionID)
	if err != nil {
		return nil, err
	}
	return &handler.CreateChatResp{Session: *activated}, nil
}

// GetChat returns one owned session and its Agent-owned message history.
func (s *ChatService) GetChat(ctx context.Context, req *handler.GetChatReq) (*handler.GetChatResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	session, err := s.requireSession(ctx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}
	if session.AgentSessionID == "" {
		// Creation never confirmed an Agent session; there is no history to
		// read, and reporting an empty list would look like a successful load.
		return nil, fmt.Errorf("chat session has no agent session yet: %w", faults.ErrConflict)
	}
	messages, err := s.agent.GetMessages(ctx, session.AgentSessionID)
	if err != nil {
		return nil, err
	}
	if messages == nil {
		messages = []chat.Message{}
	}
	return &handler.GetChatResp{Session: *session, Messages: messages}, nil
}

// ListSessions returns the caller's own sessions, newest first.
func (s *ChatService) ListSessions(ctx context.Context, req *handler.ListSessionsReq) (*handler.ListSessionsResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	page, size := normalizePage(req.Page, req.PageSize)
	items, total, err := s.sessions.ListByOwner(ctx, req.ActorID, page, size)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []chat.Session{}
	}
	return &handler.ListSessionsResp{Items: items, Total: total, Page: page, PageSize: size}, nil
}

// Ask starts a question run and hands the Agent's SSE stream back to the HTTP
// layer untouched. Validation and ownership happen before the stream opens, so
// a rejected request never emits a partial stream.
func (s *ChatService) Ask(ctx context.Context, req *handler.AskReq) (*handler.AskStream, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	question := strings.TrimSpace(req.Question)
	if question == "" || utf8.RuneCountInString(question) > maxQuestionRunes {
		return nil, fmt.Errorf("question is required and must not exceed %d characters: %w", maxQuestionRunes, faults.ErrValidation)
	}
	if len(req.Images) > maxImages {
		return nil, fmt.Errorf("at most %d images are allowed: %w", maxImages, faults.ErrValidation)
	}
	if err := validateKnowledgeBases(req.KnowledgeBases); err != nil {
		return nil, err
	}
	session, err := s.requireSession(ctx, req.SessionID, req.ActorID)
	if err != nil {
		return nil, err
	}
	if session.AgentSessionID == "" {
		return nil, fmt.Errorf("chat session has no agent session yet: %w", faults.ErrConflict)
	}
	body, err := s.agent.StartRun(ctx, session.AgentSessionID, chat.RunRequest{
		Question:       question,
		KnowledgeBases: req.KnowledgeBases,
		Images:         req.Images,
	})
	if err != nil {
		return nil, err
	}
	return &handler.AskStream{ContentType: "text/event-stream", Body: body}, nil
}

// requireSession resolves a public id to an owned mapping. A session owned by
// someone else is reported as not found to avoid leaking its existence.
func (s *ChatService) requireSession(ctx context.Context, publicID string, ownerID uint) (*chat.Session, error) {
	if strings.TrimSpace(publicID) == "" {
		return nil, fmt.Errorf("session id is required: %w", faults.ErrValidation)
	}
	return s.sessions.GetByPublicID(ctx, publicID, ownerID)
}

// recordCreationFailure persists the lifecycle outcome of a failed Agent
// creation. A timeout or a lost response has an unknown outcome, so it becomes
// pending for later reconciliation; retrying it here could create a second
// Agent session (ADR-0003). A dial failure or an explicit rejection is a known
// failure and is terminal.
func (s *ChatService) recordCreationFailure(ctx context.Context, publicID string, cause error) {
	code := "AGENT_UNAVAILABLE"
	switch {
	case errors.Is(cause, faults.ErrNotFound):
		code = "AGENT_NOT_FOUND"
	case errors.Is(cause, repository.ErrAgentRejected), errors.Is(cause, faults.ErrValidation):
		code = "AGENT_REJECTED"
	case errors.Is(cause, repository.ErrAgentTimeout):
		code = "AGENT_TIMEOUT"
	case errors.Is(cause, repository.ErrAgentUnreachable):
		code = "AGENT_UNREACHABLE"
	}
	// The outcome record is best effort: the original failure is what the
	// caller must see, so a bookkeeping error cannot replace it.
	if isUnknownOutcome(cause) {
		_ = s.sessions.MarkPending(ctx, publicID, code)
		return
	}
	_ = s.sessions.MarkFailed(ctx, publicID, code)
}

func validateKnowledgeBases(ids []string) error {
	seen := make(map[string]struct{}, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		parsed, err := strconv.ParseUint(id, 10, 64)
		if err != nil || parsed == 0 || strconv.FormatUint(parsed, 10) != id {
			return fmt.Errorf("invalid knowledge base id: %w", faults.ErrValidation)
		}
		if _, duplicate := seen[id]; duplicate {
			return fmt.Errorf("duplicate knowledge base id: %w", faults.ErrValidation)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// isUnknownOutcome reports whether a failed Agent creation may still have
// succeeded upstream. Only provable non-delivery and explicit rejections are
// safe to treat as terminal failures.
func isUnknownOutcome(err error) bool {
	return errors.Is(err, repository.ErrAgentTimeout) || errors.Is(err, repository.ErrAgentUnavailable)
}

func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

// newPublicID returns an opaque, unguessable platform identifier.
func newPublicID() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate chat session id: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}
