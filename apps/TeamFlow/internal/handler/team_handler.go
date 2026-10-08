package handler

import (
	"teamflow/internal/middleware"
	"teamflow/internal/service"
	"teamflow/pkg/apperr"

	"github.com/gin-gonic/gin"
)

type TeamHandler struct {
	teamService *service.TeamService
}

func (h *TeamHandler) AddMember(c *gin.Context) {

	currentRole, ok := c.Get(middleware.ContextKeyProjectRole)
	if !ok {
		err := apperr.ErrUnauthorized
		c.Error(err)
		c.Abort()
		return
	}
	role := currentRole.(string)
	if role != "member" {
		err := apperr.ErrUnauthorized
		c.Error(err)
		c.Abort()
		return
	}
	return
}
