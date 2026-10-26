package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPathIDRejectsNonCanonicalIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := map[string]uint{"7": 7, "0": 0, "01": 0, "-1": 0, "abc": 0}
	for raw, want := range cases {
		router := gin.New()
		var got uint
		router.GET("/items/:id", func(c *gin.Context) {
			id, ok := PathID(c, "id")
			if !ok {
				return
			}
			got = id
		})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/items/"+raw, nil))
		if got != want {
			t.Fatalf("PathID(%q)=%d, want %d", raw, got, want)
		}
		if want == 0 && (response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "INVALID_ARGUMENT")) {
			t.Fatalf("PathID(%q) status=%d body=%s, want 400 INVALID_ARGUMENT", raw, response.Code, response.Body.String())
		}
	}
}

func TestParsePageAppliesDefaultsAndBounds(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		query     string
		page      int
		size      int
		wantValid bool
		status    int
	}{
		{"", 1, 20, true, 0},
		{"?page=3&page_size=50", 3, 50, true, 0},
		{"?page=0", 0, 0, false, http.StatusBadRequest},
		{"?page=100001", 0, 0, false, http.StatusBadRequest},
		{"?page_size=101", 0, 0, false, http.StatusBadRequest},
		{"?page=x", 0, 0, false, http.StatusBadRequest},
	}
	for _, tc := range cases {
		router := gin.New()
		var page, size int
		var valid bool
		router.GET("/items", func(c *gin.Context) {
			page, size, valid = ParsePage(c)
		})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/items"+tc.query, nil))
		if valid != tc.wantValid || page != tc.page || size != tc.size {
			t.Fatalf("%q: page=%d size=%d valid=%v, want %d/%d/%v", tc.query, page, size, valid, tc.page, tc.size, tc.wantValid)
		}
		if tc.status != 0 && response.Code != tc.status {
			t.Fatalf("%q: status=%d, want %d", tc.query, response.Code, tc.status)
		}
	}
}
