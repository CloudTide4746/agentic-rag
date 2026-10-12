package platform

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"teamflow/internal/auth"
	authhandler "teamflow/internal/auth/handler"

	"github.com/gin-gonic/gin"
)

type fakeAuthHandler struct {
	identity *auth.Identity
	err      error
}

func (f fakeAuthHandler) Register(context.Context, *authhandler.RegisterReq) (*auth.User, error) {
	return nil, nil
}
func (f fakeAuthHandler) Login(context.Context, *authhandler.LoginReq) (*authhandler.AuthResp, error) {
	return nil, nil
}
func (f fakeAuthHandler) Authenticate(context.Context, string) (*auth.Identity, error) {
	return f.identity, f.err
}

func TestAuthMiddlewareAbortsForMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	router := gin.New()
	router.Use(requestIDMiddleware(), authMiddleware(fakeAuthHandler{}))
	router.GET("/protected", func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", response.Code)
	}
	if called {
		t.Fatal("protected handler ran without a token")
	}
	if !strings.Contains(response.Body.String(), `"error"`) || !strings.Contains(response.Body.String(), `"request_id"`) {
		t.Fatalf("unexpected error envelope: %s", response.Body.String())
	}
}

func TestAuthMiddlewareMapsSessionBackendFailureToUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestIDMiddleware(), authMiddleware(fakeAuthHandler{err: errors.New("redis unavailable")}))
	router.GET("/protected", func(c *gin.Context) { c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer token")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", response.Code)
	}
	if strings.Contains(response.Body.String(), "redis unavailable") {
		t.Fatal("internal dependency error leaked to client")
	}
}

func TestAuthMiddlewareSetsVerifiedActorAndContinues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(authMiddleware(fakeAuthHandler{identity: &auth.Identity{UserID: 9}}))
	router.GET("/protected", func(c *gin.Context) { success(c, http.StatusOK, gin.H{"actor_id": actorID(c)}) })
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer valid")
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", response.Code)
	}
	var payload struct {
		Data struct {
			ActorID uint `json:"actor_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.ActorID != 9 {
		t.Fatalf("actor=%d, want verified actor 9", payload.Data.ActorID)
	}
}

func TestRequestIDRejectsHeaderInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.GET("/", func(c *gin.Context) { success(c, http.StatusOK, nil) })
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-ID", "safe\r\nX-Evil: yes")
	router.ServeHTTP(response, request)
	if strings.Contains(response.Header().Get("X-Request-ID"), "\n") {
		t.Fatal("request id contains a newline")
	}
	if response.Header().Get("X-Request-ID") == "safe\r\nX-Evil: yes" {
		t.Fatal("untrusted request id was accepted")
	}
}

func TestErrorEnvelopeDoesNotExposeInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.GET("/", func(c *gin.Context) { writeError(c, errors.New("sql: password=secret")) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	if strings.Contains(response.Body.String(), "secret") {
		t.Fatal("internal error details leaked")
	}
	if !strings.Contains(response.Body.String(), "INTERNAL") {
		t.Fatalf("missing stable error code: %s", response.Body.String())
	}
}

func TestRecoveryMiddlewareWritesUnifiedErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(recoveryMiddleware(nil), requestIDMiddleware())
	router.GET("/", func(*gin.Context) { panic("do not expose this value") })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", response.Code)
	}
	if strings.Contains(response.Body.String(), "do not expose") {
		t.Fatal("panic details leaked")
	}
	if !strings.Contains(response.Body.String(), `"request_id"`) || !strings.Contains(response.Body.String(), `"INTERNAL"`) {
		t.Fatalf("unexpected error envelope: %s", response.Body.String())
	}
}
