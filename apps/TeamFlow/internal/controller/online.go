package controller

import (
	"teamflow/internal/service"
	"teamflow/pkg/apperr"
	"teamflow/pkg/response"

	"github.com/gin-gonic/gin"
)

type OnlineController struct {
	onlineSvc *service.OnlineService
}

func NewOnlineController(onlineSvc *service.OnlineService) *OnlineController {
	return &OnlineController{onlineSvc: onlineSvc}
}

// GetOnlineUsers 获取当前在线用户列表。
// @Summary GetOnlineUsers
// @Description GET /online/users
// @Tags online
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /online/users [get]
func (c *OnlineController) GetOnlineUsers(ctx *gin.Context) {
	onlineUsers, err := c.onlineSvc.GetOnlineUsers(ctx)
	if err != nil {
		ctx.Error(apperr.ErrBadRequest)
		return
	}
	response.Success(ctx, onlineUsers)
}

// AddUserToOnline 添加用户到在线列表。
// @Summary AddUserToOnline
// @Description POST /online/users/{userId}
// @Tags online
// @Accept json
// @Produce json
// @Param userId path string true "userId ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /online/users/{userId} [post]
func (c *OnlineController) AddUserToOnline(ctx *gin.Context) {
	userID, ok := parseID(ctx, "userId")
	if !ok {
		return
	}
	if err := c.onlineSvc.SetOnline(ctx, userID); err != nil {
		ctx.Error(apperr.ErrBadRequest)
		return
	}
	response.Success(ctx, gin.H{"message": "用户已添加到在线列表"})
}

// RemoveUserFromOnline 从在线列表移除用户。
// @Summary RemoveUserFromOnline
// @Description DELETE /online/users/{userId}
// @Tags online
// @Accept json
// @Produce json
// @Param userId path string true "userId ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /online/users/{userId} [delete]
func (c *OnlineController) RemoveUserFromOnline(ctx *gin.Context) {
	userID, ok := parseID(ctx, "userId")
	if !ok {
		return
	}
	if err := c.onlineSvc.SetOffline(ctx, userID); err != nil {
		ctx.Error(apperr.ErrBadRequest)
		return
	}
	response.Success(ctx, gin.H{"message": "用户已从在线列表移除"})
}
