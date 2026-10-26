package handler

import (
	"io"
	"net/http"

	"teamflow/internal/httpx"

	"github.com/gin-gonic/gin"
)

// Handler adapts chat HTTP requests onto the chat use cases and owns the SSE
// pass-through to the browser.
type Handler struct{ app ChatHandler }

func New(app ChatHandler) *Handler { return &Handler{app: app} }

func (h *Handler) Register(group *gin.RouterGroup) {
	group.POST("/chat/sessions", h.create)
	group.GET("/chat/sessions", h.list)
	group.GET("/chat/sessions/:id", h.get)
	group.GET("/chat/sessions/:id/messages", h.messages)
	group.POST("/chat/sessions/:id/runs", h.ask)
}

func (h *Handler) create(c *gin.Context) {
	var body CreateChatReq
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	body.ActorID = httpx.ActorID(c)
	resp, err := h.app.CreateChat(c.Request.Context(), &body)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

func (h *Handler) list(c *gin.Context) {
	page, size, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	resp, err := h.app.ListSessions(c.Request.Context(), &ListSessionsReq{ActorID: httpx.ActorID(c), Page: page, PageSize: size})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) get(c *gin.Context) {
	// The public id is an opaque platform string, not a numeric id.
	resp, err := h.app.GetChat(c.Request.Context(), &GetChatReq{ActorID: httpx.ActorID(c), SessionID: c.Param("id")})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

func (h *Handler) messages(c *gin.Context) {
	resp, err := h.app.GetChat(c.Request.Context(), &GetChatReq{ActorID: httpx.ActorID(c), SessionID: c.Param("id")})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp.Messages)
}

func (h *Handler) ask(c *gin.Context) {
	var body AskReq
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	body.ActorID, body.SessionID = httpx.ActorID(c), c.Param("id")
	stream, err := h.app.Ask(c.Request.Context(), &body)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	// Forward the Agent's SSE frames verbatim. The browser never reaches
	// the Agent directly (ADR-0003 "消息和问答流程").
	defer stream.Body.Close()
	c.Header("Content-Type", stream.ContentType)
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	c.Stream(func(w io.Writer) bool {
		buf := make([]byte, 4096)
		written, readErr := stream.Body.Read(buf)
		if written > 0 {
			if _, err := w.Write(buf[:written]); err != nil {
				return false
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
		if readErr != nil {
			return false
		}
		return true
	})
}
