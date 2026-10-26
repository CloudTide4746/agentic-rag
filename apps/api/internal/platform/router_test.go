package platform

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"teamflow/internal/httpx"

	"github.com/gin-gonic/gin"
)

func TestRequestIDRejectsHeaderInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(requestIDMiddleware())
	router.GET("/", func(c *gin.Context) { httpx.Success(c, http.StatusOK, nil) })
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
	router.GET("/", func(c *gin.Context) { httpx.WriteError(c, errors.New("sql: password=secret")) })
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
