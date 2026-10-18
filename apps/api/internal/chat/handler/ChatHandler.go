package handler

import (
	"context"
	"io"

	"teamflow/internal/chat"
)

type CreateChatReq struct {
	ActorID uint   `json:"-"`
	Title   string `json:"title,omitempty"`
}

type CreateChatResp struct {
	Session chat.Session `json:"session"`
}

type GetChatReq struct {
	ActorID   uint   `json:"-"`
	SessionID string `json:"-"`
}

type GetChatResp struct {
	Session  chat.Session   `json:"session"`
	Messages []chat.Message `json:"messages"`
}

type ListSessionsReq struct {
	ActorID  uint `json:"-"`
	Page     int  `form:"page"`
	PageSize int  `form:"page_size"`
}

type ListSessionsResp struct {
	Items    []chat.Session `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// AskReq is the body of POST /api/v1/chat/sessions/{id}/runs.
//
// skip_user_persist is deliberately absent: whether the Agent persists the
// user turn is a platform decision, not a browser-controlled flag (ADR-0003).
type AskReq struct {
	ActorID        uint     `json:"-"`
	SessionID      string   `json:"-"`
	Question       string   `json:"question"`
	KnowledgeBases []string `json:"knowledge_base_ids,omitempty"`
	Images         []string `json:"images,omitempty"`
}

// AskStream carries the Agent's SSE frames straight through to the caller.
type AskStream struct {
	ContentType string
	Body        io.ReadCloser
}

type ChatHandler interface {
	CreateChat(ctx context.Context, req *CreateChatReq) (*CreateChatResp, error)
	GetChat(ctx context.Context, req *GetChatReq) (*GetChatResp, error)
	ListSessions(ctx context.Context, req *ListSessionsReq) (*ListSessionsResp, error)
	Ask(ctx context.Context, req *AskReq) (*AskStream, error)
}
