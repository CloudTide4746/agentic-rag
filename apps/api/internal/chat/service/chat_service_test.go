package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"teamflow/internal/chat"
	"teamflow/internal/chat/handler"
	"teamflow/internal/chat/repository"
	"teamflow/internal/faults"
)

type fakeStore struct {
	created       bool
	createdItem   chat.Session
	createdKey    string
	activeMarked  bool
	failedMarked  bool
	pendingMarked bool
	markedCode    string
	item          *chat.Session
	listItems     []chat.Session
	listTotal     int64
	listPage      int
	listSize      int
	createErr     error
	getErr        error
}

func (f *fakeStore) Create(_ context.Context, item chat.Session, idempotencyKey string) (*chat.Session, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.created = true
	f.createdItem = item
	f.createdKey = idempotencyKey
	copy := item
	return &copy, nil
}
func (f *fakeStore) GetByPublicID(context.Context, string, uint) (*chat.Session, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.item == nil {
		return nil, faults.ErrNotFound
	}
	copy := *f.item
	return &copy, nil
}
func (f *fakeStore) ListByOwner(_ context.Context, _ uint, page, size int) ([]chat.Session, int64, error) {
	f.listPage, f.listSize = page, size
	return f.listItems, f.listTotal, nil
}
func (f *fakeStore) MarkActive(_ context.Context, publicID, agentSessionID string) (*chat.Session, error) {
	f.activeMarked = true
	// Mirror the store: the caller reports the committed state, not the
	// pre-update snapshot it passed in.
	item := f.createdItem
	item.PublicID = publicID
	item.AgentSessionID = agentSessionID
	item.Status = chat.StatusActive
	return &item, nil
}
func (f *fakeStore) MarkFailed(_ context.Context, _ string, code string) error {
	f.failedMarked, f.markedCode = true, code
	return nil
}
func (f *fakeStore) MarkPending(_ context.Context, _ string, code string) error {
	f.pendingMarked, f.markedCode = true, code
	return nil
}

type fakeAgent struct {
	createdAgentID string
	createErr      error
	messages       []chat.Message
	messagesErr    error
	stream         io.ReadCloser
	streamErr      error
	ranSessionID   string
	ranRequest     chat.RunRequest
}

func (f *fakeAgent) CreateSession(context.Context, string, string) (string, error) {
	if f.createErr != nil {
		return "", f.createErr
	}
	if f.createdAgentID == "" {
		return "42", nil
	}
	return f.createdAgentID, nil
}
func (f *fakeAgent) GetMessages(context.Context, string) ([]chat.Message, error) {
	return f.messages, f.messagesErr
}
func (f *fakeAgent) StartRun(_ context.Context, agentSessionID string, req chat.RunRequest) (io.ReadCloser, error) {
	if f.streamErr != nil {
		return nil, f.streamErr
	}
	f.ranSessionID, f.ranRequest = agentSessionID, req
	if f.stream == nil {
		return io.NopCloser(strings.NewReader("event: done\n\ndata: {}\n\n")), nil
	}
	return f.stream, nil
}

func activeSession() *chat.Session {
	return &chat.Session{PublicID: "pub-1", OwnerID: "7", Title: "T", Status: chat.StatusActive, AgentSessionID: "42"}
}

func TestCreateChatRequiresAuthenticatedActor(t *testing.T) {
	store := &fakeStore{}
	svc := NewChatService(store, &fakeAgent{})
	_, err := svc.CreateChat(context.Background(), &handler.CreateChatReq{Title: "hello"})
	if !errors.Is(err, faults.ErrUnauthenticated) {
		t.Fatalf("error = %v, want unauthenticated", err)
	}
	if store.created {
		t.Fatal("session created without an authenticated actor")
	}
}

func TestCreateChatMarksActiveOnSuccess(t *testing.T) {
	store := &fakeStore{}
	agent := &fakeAgent{createdAgentID: "42"}
	svc := NewChatService(store, agent)
	resp, err := svc.CreateChat(context.Background(), &handler.CreateChatReq{ActorID: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.activeMarked {
		t.Fatal("session was not marked active after agent creation")
	}
	if resp.Session.Status != chat.StatusActive {
		t.Fatalf("status = %q, want active", resp.Session.Status)
	}
	// The response must carry the confirmed Agent mapping, not a pre-update
	// snapshot that still looks unlinked.
	if resp.Session.AgentSessionID != "42" {
		t.Fatalf("agent session id = %q, want 42 in the committed response", resp.Session.AgentSessionID)
	}
	if resp.Session.PublicID == "" {
		t.Fatal("committed response lost the platform id")
	}
	// A blank title must fall back to the default rather than reach the Agent.
	if store.createdItem.Title != "新会话" {
		t.Fatalf("title = %q, want default", store.createdItem.Title)
	}
	// The platform id is generated, never the Agent's numeric id.
	if resp.Session.PublicID == "42" || resp.Session.PublicID == "" {
		t.Fatalf("public id = %q, want an opaque platform id", resp.Session.PublicID)
	}
}

func TestCreateChatRecordsPendingWhenOutcomeUnknown(t *testing.T) {
	store := &fakeStore{}
	agent := &fakeAgent{createErr: repository.ErrAgentTimeout}
	svc := NewChatService(store, agent)
	if _, err := svc.CreateChat(context.Background(), &handler.CreateChatReq{ActorID: 7}); err == nil {
		t.Fatal("expected error when agent creation times out")
	}
	if !store.pendingMarked {
		t.Fatal("timed-out creation must be recorded as pending for reconciliation")
	}
	if store.failedMarked {
		t.Fatal("unknown outcome must not be recorded as a terminal failure")
	}
}

func TestCreateChatRecordsFailedOnDialFailure(t *testing.T) {
	store := &fakeStore{}
	agent := &fakeAgent{createErr: repository.ErrAgentUnreachable}
	svc := NewChatService(store, agent)
	if _, err := svc.CreateChat(context.Background(), &handler.CreateChatReq{ActorID: 7}); err == nil {
		t.Fatal("expected error when agent is unreachable")
	}
	if !store.failedMarked {
		t.Fatal("provable non-delivery must be recorded as a terminal failure")
	}
	if store.pendingMarked {
		t.Fatal("provable non-delivery must not be recorded as pending")
	}
}

func TestCreateChatRejectsOversizedTitle(t *testing.T) {
	store := &fakeStore{}
	svc := NewChatService(store, &fakeAgent{})
	_, err := svc.CreateChat(context.Background(), &handler.CreateChatReq{ActorID: 7, Title: strings.Repeat("字", 200)})
	if !errors.Is(err, faults.ErrValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
	if store.created {
		t.Fatal("session created with an invalid title")
	}
}

func TestGetChatRequiresAuthenticatedActor(t *testing.T) {
	svc := NewChatService(&fakeStore{}, &fakeAgent{})
	_, err := svc.GetChat(context.Background(), &handler.GetChatReq{SessionID: "pub-1"})
	if !errors.Is(err, faults.ErrUnauthenticated) {
		t.Fatalf("error = %v, want unauthenticated", err)
	}
}

func TestGetChatRejectsMissingSession(t *testing.T) {
	svc := NewChatService(&fakeStore{getErr: faults.ErrNotFound}, &fakeAgent{})
	_, err := svc.GetChat(context.Background(), &handler.GetChatReq{ActorID: 7, SessionID: "pub-1"})
	if !errors.Is(err, faults.ErrNotFound) {
		t.Fatalf("error = %v, want not found", err)
	}
}

func TestGetChatRejectsSessionWithoutAgentSession(t *testing.T) {
	store := &fakeStore{item: &chat.Session{PublicID: "pub-1", OwnerID: "7", Status: chat.StatusPending}}
	agent := &fakeAgent{messages: []chat.Message{{Role: "user", Content: "hi"}}}
	svc := NewChatService(store, agent)
	// A session whose Agent creation never confirmed must not look like an
	// empty-but-successful history.
	_, err := svc.GetChat(context.Background(), &handler.GetChatReq{ActorID: 7, SessionID: "pub-1"})
	if !errors.Is(err, faults.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
}

func TestGetChatReturnsMessages(t *testing.T) {
	store := &fakeStore{item: activeSession()}
	agent := &fakeAgent{messages: []chat.Message{{Role: "user", Content: "hi"}}}
	svc := NewChatService(store, agent)
	resp, err := svc.GetChat(context.Background(), &handler.GetChatReq{ActorID: 7, SessionID: "pub-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(resp.Messages))
	}
	if resp.Session.PublicID != "pub-1" {
		t.Fatalf("session id = %q, want pub-1", resp.Session.PublicID)
	}
}

func TestGetChatReturnsEmptySliceNotNil(t *testing.T) {
	store := &fakeStore{item: activeSession()}
	svc := NewChatService(store, &fakeAgent{})
	resp, err := svc.GetChat(context.Background(), &handler.GetChatReq{ActorID: 7, SessionID: "pub-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Messages == nil {
		t.Fatal("messages must serialise as [] rather than null")
	}
}

func TestListSessionsRequiresAuthenticatedActor(t *testing.T) {
	svc := NewChatService(&fakeStore{}, &fakeAgent{})
	_, err := svc.ListSessions(context.Background(), &handler.ListSessionsReq{})
	if !errors.Is(err, faults.ErrUnauthenticated) {
		t.Fatalf("error = %v, want unauthenticated", err)
	}
}

func TestListSessionsNormalizesPaging(t *testing.T) {
	store := &fakeStore{listItems: []chat.Session{{PublicID: "pub-1"}}, listTotal: 1}
	svc := NewChatService(store, &fakeAgent{})
	resp, err := svc.ListSessions(context.Background(), &handler.ListSessionsReq{ActorID: 7, Page: 0, PageSize: 500})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Page != 1 || resp.PageSize != 100 {
		t.Fatalf("paging = %d/%d, want 1/100", resp.Page, resp.PageSize)
	}
	if store.listPage != 1 || store.listSize != 100 {
		t.Fatalf("store paging = %d/%d, want 1/100", store.listPage, store.listSize)
	}
}

func TestListSessionsReturnsEmptySliceNotNil(t *testing.T) {
	svc := NewChatService(&fakeStore{}, &fakeAgent{})
	resp, err := svc.ListSessions(context.Background(), &handler.ListSessionsReq{ActorID: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Items == nil {
		t.Fatal("items must serialise as [] rather than null")
	}
}

func TestAskRequiresQuestion(t *testing.T) {
	store := &fakeStore{item: activeSession()}
	svc := NewChatService(store, &fakeAgent{})
	_, err := svc.Ask(context.Background(), &handler.AskReq{ActorID: 7, SessionID: "pub-1", Question: "   "})
	if !errors.Is(err, faults.ErrValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
}

func TestAskRejectsOversizedQuestion(t *testing.T) {
	store := &fakeStore{item: activeSession()}
	svc := NewChatService(store, &fakeAgent{})
	_, err := svc.Ask(context.Background(), &handler.AskReq{ActorID: 7, SessionID: "pub-1", Question: strings.Repeat("字", maxQuestionRunes+1)})
	if !errors.Is(err, faults.ErrValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
}

func TestAskRejectsTooManyImages(t *testing.T) {
	store := &fakeStore{item: activeSession()}
	svc := NewChatService(store, &fakeAgent{})
	_, err := svc.Ask(context.Background(), &handler.AskReq{
		ActorID: 7, SessionID: "pub-1", Question: "hi",
		Images: []string{"a", "b", "c", "d", "e"},
	})
	if !errors.Is(err, faults.ErrValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
}

func TestAskRejectsInvalidKnowledgeBaseID(t *testing.T) {
	store := &fakeStore{item: activeSession()}
	agent := &fakeAgent{}
	svc := NewChatService(store, agent)
	_, err := svc.Ask(context.Background(), &handler.AskReq{
		ActorID: 7, SessionID: "pub-1", Question: "hi", KnowledgeBases: []string{"../etc"},
	})
	if !errors.Is(err, faults.ErrValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
	if agent.ranSessionID != "" {
		t.Fatal("agent run started with an invalid knowledge base id")
	}
}

func TestAskRejectsDuplicateKnowledgeBaseIDs(t *testing.T) {
	store := &fakeStore{item: activeSession()}
	svc := NewChatService(store, &fakeAgent{})
	_, err := svc.Ask(context.Background(), &handler.AskReq{
		ActorID: 7, SessionID: "pub-1", Question: "hi", KnowledgeBases: []string{"1", "1"},
	})
	if !errors.Is(err, faults.ErrValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
}

func TestAskRejectsUnconfirmedAgentSession(t *testing.T) {
	store := &fakeStore{item: &chat.Session{PublicID: "pub-1", OwnerID: "7", Status: chat.StatusPending}}
	agent := &fakeAgent{}
	svc := NewChatService(store, agent)
	_, err := svc.Ask(context.Background(), &handler.AskReq{ActorID: 7, SessionID: "pub-1", Question: "hi"})
	if !errors.Is(err, faults.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
	if agent.ranSessionID != "" {
		t.Fatal("agent run started without a confirmed agent session")
	}
}

func TestAskUsesInternalAgentSessionIDAndPropagatesStream(t *testing.T) {
	store := &fakeStore{item: activeSession()}
	agent := &fakeAgent{}
	svc := NewChatService(store, agent)
	stream, err := svc.Ask(context.Background(), &handler.AskReq{
		ActorID: 7, SessionID: "pub-1", Question: "  hi  ", KnowledgeBases: []string{"3"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stream.Body.Close()
	if agent.ranSessionID != "42" {
		t.Fatalf("agent session id = %q, want the internal id 42", agent.ranSessionID)
	}
	if stream.ContentType != "text/event-stream" {
		t.Fatalf("content type = %q, want text/event-stream", stream.ContentType)
	}
	if agent.ranRequest.Question != "hi" {
		t.Fatalf("question = %q, want trimmed input", agent.ranRequest.Question)
	}
	// The browser must not be able to control user-turn persistence.
	if agent.ranRequest.SkipUserPersist {
		t.Fatal("skip_user_persist must not be settable from the public request")
	}
}

func TestAskPropagatesAgentFailure(t *testing.T) {
	store := &fakeStore{item: activeSession()}
	agent := &fakeAgent{streamErr: repository.ErrAgentUnavailable}
	svc := NewChatService(store, agent)
	_, err := svc.Ask(context.Background(), &handler.AskReq{ActorID: 7, SessionID: "pub-1", Question: "hi"})
	if !errors.Is(err, repository.ErrAgentUnavailable) {
		t.Fatalf("error = %v, want agent unavailable", err)
	}
}
