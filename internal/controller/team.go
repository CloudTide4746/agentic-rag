package controller

import (
	"net/http"
	"strconv"
	"teamflow/internal/dto/request"
	"teamflow/internal/service"
	"teamflow/pkg/response"

	"github.com/gin-gonic/gin"
)

// TeamController 团队控制器。
type TeamController struct {
	teamService *service.TeamService
}

// NewTeamController 创建团队控制器。
func NewTeamController(teamService *service.TeamService) *TeamController {
	return &TeamController{teamService: teamService}
}

// CreateTeam 创建团队。
// @Summary CreateTeam
// @Description POST /teams
// @Tags teams
// @Accept json
// @Produce json
// @Param request body request.CreateTeamRequest true "Request body"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams [post]
func (c *TeamController) CreateTeam(ctx *gin.Context) {
	var req request.CreateTeamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}

	team, err := c.teamService.CreateTeam(ctx, ctx.GetUint("userID"), &req)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, team)
}

// GetTeamList 获取当前用户加入的团队列表。
// @Summary GetTeamList
// @Description GET /teams
// @Tags teams
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams [get]
func (c *TeamController) GetTeamList(ctx *gin.Context) {
	page := queryInt(ctx, "page", 1)
	pageSize := queryInt(ctx, "page_size", 10)
	if pageSize > 100 {
		pageSize = 10
	}

	teams, total, err := c.teamService.GetTeamList(ctx, ctx.GetUint("userID"), page, pageSize)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{
		"list":      teams,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetTeamDetail 获取团队详情。
// @Summary GetTeamDetail
// @Description GET /teams/{id}
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams/{id} [get]
func (c *TeamController) GetTeamDetail(ctx *gin.Context) {
	teamID, ok := parseID(ctx, "id")
	if !ok {
		return
	}

	team, err := c.teamService.GetTeamDetail(ctx, teamID, ctx.GetUint("userID"))
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, team)
}

// DeleteTeam 解散团队。
// @Summary DeleteTeam
// @Description DELETE /teams/{id}
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams/{id} [delete]
func (c *TeamController) DeleteTeam(ctx *gin.Context) {
	teamID, ok := parseID(ctx, "id")
	if !ok {
		return
	}

	if err := c.teamService.DeleteTeam(ctx, teamID, ctx.GetUint("userID")); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{"message": "团队已解散"})
}

// UpdateTeam 更新团队。
// @Summary UpdateTeam
// @Description PUT /teams/{id}
// @Tags teams
// @Accept json
// @Produce json
// @Param request body request.UpdateTeamRequest true "Request body"
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams/{id} [put]
func (c *TeamController) UpdateTeam(ctx *gin.Context) {
	teamID, ok := parseID(ctx, "id")
	if !ok {
		return
	}

	var req request.UpdateTeamRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	team, err := c.teamService.UpdateTeam(ctx, teamID, ctx.GetUint("userID"), &req)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, team)
}

// AddMember 添加团队成员。
// @Summary AddMember
// @Description POST /teams/{id}/members
// @Tags teams
// @Accept json
// @Produce json
// @Param request body request.AddTeamMemberRequest true "Request body"
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams/{id}/members [post]
func (c *TeamController) AddMember(ctx *gin.Context) {
	teamID, ok := parseID(ctx, "id")
	if !ok {
		return
	}

	var req request.AddTeamMemberRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.teamService.AddMember(ctx, teamID, ctx.GetUint("userID"), &req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{"message": "成员添加成功"})
}

// RemoveMember 移除团队成员。
// @Summary RemoveMember
// @Description DELETE /teams/{id}/members/{userId}
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Param userId path string true "userId ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams/{id}/members/{userId} [delete]
func (c *TeamController) RemoveMember(ctx *gin.Context) {
	teamID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	memberID, ok := parseID(ctx, "userId")
	if !ok {
		return
	}

	if err := c.teamService.RemoveMember(ctx, teamID, ctx.GetUint("userID"), memberID); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{"message": "成员已移除"})
}

// LeaveTeam 退出当前团队。
// @Summary LeaveTeam
// @Description DELETE /teams/{id}/members/me
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams/{id}/members/me [delete]
func (c *TeamController) LeaveTeam(ctx *gin.Context) {
	teamID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := c.teamService.LeaveTeam(ctx, teamID, ctx.GetUint("userID")); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{"message": "已退出团队"})
}

func parseID(ctx *gin.Context, name string) (uint, bool) {
	id, err := strconv.ParseUint(ctx.Param(name), 10, 64)
	if err != nil || id == 0 {
		response.Error(ctx, http.StatusBadRequest, "无效的 ID")
		return 0, false
	}
	return uint(id), true
}

func queryInt(ctx *gin.Context, name string, defaultValue int) int {
	value, err := strconv.Atoi(ctx.Query(name))
	if err != nil || value < 1 {
		return defaultValue
	}
	return value
}
