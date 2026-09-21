package platform

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestPublicResponses(t *testing.T) {
	r := NewRouter(zap.NewNop())
	for _, tc := range []struct {
		method, path string
		status       int
		key          string
	}{
		{"GET", "/api/v1/health/live", 200, "data"},
		{"GET", "/api/v1/health/ready", 503, "error"},
		{"GET", "/api/v1/missing", 404, "error"},
		{"POST", "/api/v1/health/live", 405, "error"},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil))
			if w.Code != tc.status {
				t.Fatalf("status = %d, want %d", w.Code, tc.status)
			}
			var body map[string]json.RawMessage
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if _, ok := body[tc.key]; !ok {
				t.Fatalf("missing %s", tc.key)
			}
			var id string
			if err := json.Unmarshal(body["request_id"], &id); err != nil {
				t.Fatal(err)
			}
			if len(id) != 32 || id != w.Header().Get("X-Request-ID") {
				t.Fatal("invalid request correlation")
			}
		})
	}
}
