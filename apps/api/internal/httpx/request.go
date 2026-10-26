package httpx

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const ActorIDKey = "actor_id"

// ActorID returns the actor id injected by the auth middleware. It is zero
// when no verified identity is present; use cases treat that as unauthenticated.
func ActorID(c *gin.Context) uint { id, _ := c.Get(ActorIDKey); value, _ := id.(uint); return value }

// PathID parses a canonical positive decimal path parameter.
func PathID(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != raw {
		Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "无效的资源 ID")
		return 0, false
	}
	return uint(id), true
}

// ParsePage reads the bounded page/page_size query parameters with defaults.
func ParsePage(c *gin.Context) (int, int, bool) {
	page, size := 1, 20
	if raw := c.Query("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100000 {
			Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "page 必须在 1 到 100000 之间")
			return 0, 0, false
		}
		page = parsed
	}
	if raw := c.Query("page_size"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "page_size 必须在 1 到 100 之间")
			return 0, 0, false
		}
		size = parsed
	}
	return page, size, true
}
