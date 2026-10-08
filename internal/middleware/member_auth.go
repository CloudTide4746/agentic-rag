package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"teamflow/internal/model"
	"teamflow/pkg/response"
	"teamflow/storage"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RequireTeamRole 校验当前用户在团队中的最低角色。
func RequireTeamRole(minRole string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		teamID, ok := parseResourceID(ctx, "id", "团队")
		if !ok {
			return
		}

		var member model.TeamMember
		err := storage.DB.WithContext(ctx).
			Where("team_id = ? AND user_id = ?", teamID, ctx.GetUint("userID")).
			First(&member).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(ctx, http.StatusForbidden, "你不是该团队成员")
			ctx.Abort()
			return
		}
		if err != nil {
			response.Error(ctx, http.StatusInternalServerError, "查询团队权限失败")
			ctx.Abort()
			return
		}
		if !model.RoleAtLeast(member.Role, minRole) {
			response.Error(ctx, http.StatusForbidden, "团队权限不足")
			ctx.Abort()
			return
		}

		ctx.Set("teamRole", member.Role)
		ctx.Next()
	}
}

// RequireProjectRole 校验当前用户在项目中的最低角色。
func RequireProjectRole(minRole string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		projectID, ok := parseResourceID(ctx, "id", "项目")
		if !ok {
			return
		}

		var member model.ProjectMember
		err := storage.DB.WithContext(ctx).
			Where("project_id = ? AND user_id = ?", projectID, ctx.GetUint("userID")).
			First(&member).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(ctx, http.StatusForbidden, "你不是该项目成员")
			ctx.Abort()
			return
		}
		if err != nil {
			response.Error(ctx, http.StatusInternalServerError, "查询项目权限失败")
			ctx.Abort()
			return
		}
		if !model.RoleAtLeast(member.Role, minRole) {
			response.Error(ctx, http.StatusForbidden, "项目权限不足")
			ctx.Abort()
			return
		}

		ctx.Set("projectRole", member.Role)
		ctx.Next()
	}
}

func parseResourceID(ctx *gin.Context, name, resource string) (uint, bool) {
	id, err := strconv.ParseUint(ctx.Param(name), 10, 64)
	if err != nil || id == 0 {
		response.Error(ctx, http.StatusBadRequest, "无效的"+resource+" ID")
		ctx.Abort()
		return 0, false
	}
	return uint(id), true
}
