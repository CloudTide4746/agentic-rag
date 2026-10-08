package middleware

import (
	"strconv"
	"teamflow/internal/service"
	"teamflow/pkg/apperr"

	"github.com/gin-gonic/gin"
)

// ProjectPermissionMiddleware 检查项目是否有权限访问
// requiredRole 为项目角色，默认值为 "member"
func ProjectPermissionMiddleware(permSvc service.PermissionService, requiredRole string) gin.HandlerFunc {
	if requiredRole == "" {
		requiredRole = "member"
	}
	return func(c *gin.Context) {
		//1.取出UserID
		userID, ok := c.Get(ContextKeyUserID)
		if !ok {
			err := apperr.ErrUnauthorized
			c.Error(err)
			c.Abort()
			return
		}
		//2.取出ProjectID
		projectIDstr := c.Param("projectID")
		projectID, err := strconv.ParseUint(projectIDstr, 10, 64)
		if err != nil {
			err := apperr.ErrUnauthorized
			c.Error(err)
			c.Abort()
			return
		}

		actualRole, ok, err := permSvc.CheckProjectPermission(uint(userID.(uint)), uint(projectID), requiredRole)
		if err != nil {
			err := apperr.ErrServerError
			c.Error(err)
			c.Abort()
			return
		}
		if !ok {
			err := apperr.ErrUnauthorized
			c.Error(err)
			c.Abort()
			return
		}
		c.Set(ContextKeyProjectRole, actualRole)
		c.Next()
	}
}
