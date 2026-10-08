package handler

import (
	"strconv"
	"teamflow/internal/service"
	"teamflow/pkg/apperr"

	"github.com/gin-gonic/gin"
)

// type ProjectMemberHandler struct {
// 	memberSvc *service.ProjectMemberService
// 	permissionSvc *service.PermissionService
// }

//	func NewProjectMemberHandler(memberSvc *service.ProjectMemberService, permissionSvc *service.PermissionService) *ProjectMemberHandler {
//		return &ProjectMemberHandler{memberSvc: memberSvc, permissionSvc: permissionSvc}
//	}
type ProjectMemberHandler struct {
	memberService *service.ProjectService
	permissionSvc *service.PermissionService
}
type inviteMemberRequest struct {
	UserID uint   `json:"userID" binding:"required"`
	Role   string `json:"role" binding:"required" binding:"required,oneof=owner member"`
}

func NewProjectMemberHandler(memberservice service.ProjectService, permissionService service.PermissionService) *ProjectMemberHandler {
	return &ProjectMemberHandler{
		memberService: &memberservice,
		permissionSvc: &permissionService,
	}
}
func (h *ProjectMemberHandler) InviteMember(c *gin.Context) {
	projectID := c.Param("projectID")
	var req inviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.ErrBadRequest)
		return
	}
	// 检查项目权限
	projectIDUint, err := strconv.ParseUint(projectID, 10, 64)
	if err != nil {
		c.Error(apperr.ErrBadRequest)
		return
	}
	role, has, err := h.permissionSvc.CheckProjectPermission(uint(projectIDUint), uint(req.UserID), req.Role)
	if err != nil {
		c.Error(err)
		return
	}
	if has {
		c.Error(apperr.ErrServerError)
		return
	}
	if role != "" {
		c.Error(apperr.ErrServerError)
		return
	}
	c.JSON(200, gin.H{
		"message": "invite member success",
	})

}

// RemoveMember 移除项目成员
func (h *ProjectMemberHandler) RemoveMember(c *gin.Context) {
	projectIDUint, err := strconv.ParseUint(c.Param("projectID"), 10, 64)
	if err != nil {
		c.Error(apperr.ErrBadRequest)
		return
	}
	targetUserID, err := strconv.ParseUint(c.Param("targetUserID"), 10, 64)
	if err != nil {
		c.Error(apperr.ErrBadRequest)
		return
	}
	if targetUserID == 0 {
		c.Error(apperr.ErrBadRequest)
		return
	}
	currentUserID := c.GetUint("userID")
	if currentUserID == 0 {
		c.Error(apperr.ErrServerError)
		return
	}
	if currentUserID == uint(targetUserID) {
		c.Error(apperr.ErrServerError)
		return
	}
	var req inviteMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperr.ErrBadRequest)
		return
	}
	err = h.memberService.RemoveMember(c, uint(projectIDUint), uint(currentUserID), uint(targetUserID))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(200, gin.H{
		"message": "remove member success",
	})
}
