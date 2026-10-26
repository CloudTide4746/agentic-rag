package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"teamflow/internal/auth"
	"teamflow/internal/httpx"

	"github.com/gin-gonic/gin"
)

type fakeAuthHandler struct {
	identity *auth.Identity
	err      error
}

func (f fakeAuthHandler) Register(context.Context, *RegisterReq) (*auth.User, error) {
	return nil, nil
}
func (f fakeAuthHandler) Login(context.Context, *LoginReq) (*AuthResp, error) {
	return nil, nil
}
func (f fakeAuthHandler) Authenticate(context.Context, string) (*auth.Identity, error) {
	return f.identity, f.err
}

func TestMiddlewareAbortsForMissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	called := false
	router := gin.New()
	router.Use(New(fakeAuthHandler{}).Middleware())
	router.GET("/protected", func(c *gin.Context) { called = true; c.Status(http.StatusNoContent) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/protected", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", response.Code)
	}
	if called {
		t.Fatal("protected handler ran without a token")
	}
	if !strings.Contains(response.Body.String(), `"error"`) {
		t.Fatalf("unexpected error envelope: %s", response.Body.String())
	}
}

func TestMiddlewareMapsSessionBackendFailureToUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(New(fakeAuthHandler{err: errors.New("redis unavailable")}).Middleware())
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

func TestMiddlewareSetsVerifiedActorAndContinues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(New(fakeAuthHandler{identity: &auth.Identity{UserID: 9}}).Middleware())
	router.GET("/protected", func(c *gin.Context) { httpx.Success(c, http.StatusOK, gin.H{"actor_id": httpx.ActorID(c)}) })
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
