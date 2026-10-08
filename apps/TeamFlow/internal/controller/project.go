package controller

import (
	"net/http"
	"teamflow/internal/dto/request"
	"teamflow/internal/service"
	"teamflow/pkg/response"

	"github.com/gin-gonic/gin"
)

// ProjectController 项目控制器。
type ProjectController struct {
	projectService *service.ProjectService
}

func NewProjectController(projectService *service.ProjectService) *ProjectController {
	return &ProjectController{projectService: projectService}
}

// @Summary CreateProject
// @Description POST /teams/{id}/projects
// @Tags teams
// @Accept json
// @Produce json
// @Param request body request.CreateProjectRequest true "Request body"
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams/{id}/projects [post]
func (c *ProjectController) CreateProject(ctx *gin.Context) {
	teamID, ok := parseID(ctx, "id")
	if !ok {
		return
	}

	var req request.CreateProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	project, err := c.projectService.CreateProject(ctx, teamID, ctx.GetUint("userID"), &req)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, project)
}

// @Summary GetProjectList
// @Description GET /teams/{id}/projects
// @Tags teams
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /teams/{id}/projects [get]
func (c *ProjectController) GetProjectList(ctx *gin.Context) {
	teamID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	page := queryInt(ctx, "page", 1)
	pageSize := queryInt(ctx, "page_size", 10)
	if pageSize > 100 {
		pageSize = 100
	}

	projects, total, err := c.projectService.GetProjectList(
		ctx,
		teamID,
		ctx.GetUint("userID"),
		page,
		pageSize,
	)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{
		"list":      projects,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// @Summary GetProjectDetail
// @Description GET /projects/{id}
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /projects/{id} [get]
func (c *ProjectController) GetProjectDetail(ctx *gin.Context) {
	projectID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	project, err := c.projectService.GetProjectDetail(ctx, projectID, ctx.GetUint("userID"))
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, project)
}

// @Summary UpdateProject
// @Description PUT /projects/{id}
// @Tags projects
// @Accept json
// @Produce json
// @Param request body request.UpdateProjectRequest true "Request body"
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /projects/{id} [put]
func (c *ProjectController) UpdateProject(ctx *gin.Context) {
	projectID, ok := parseID(ctx, "id")
	if !ok {
		return
	}

	var req request.UpdateProjectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	project, err := c.projectService.UpdateProject(ctx, projectID, ctx.GetUint("userID"), &req)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, project)
}

// @Summary DeleteProject
// @Description DELETE /projects/{id}
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /projects/{id} [delete]
func (c *ProjectController) DeleteProject(ctx *gin.Context) {
	projectID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := c.projectService.DeleteProject(ctx, projectID, ctx.GetUint("userID")); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{"message": "项目已删除"})
}

// @Summary AddMember
// @Description POST /projects/{id}/members
// @Tags projects
// @Accept json
// @Produce json
// @Param request body request.AddProjectMemberRequest true "Request body"
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /projects/{id}/members [post]
func (c *ProjectController) AddMember(ctx *gin.Context) {
	projectID, ok := parseID(ctx, "id")
	if !ok {
		return
	}

	var req request.AddProjectMemberRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := c.projectService.AddMember(ctx, projectID, ctx.GetUint("userID"), &req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{"message": "项目成员添加成功"})
}

// @Summary RemoveMember
// @Description DELETE /projects/{id}/members/{userId}
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Param userId path string true "userId ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /projects/{id}/members/{userId} [delete]
func (c *ProjectController) RemoveMember(ctx *gin.Context) {
	projectID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	userID, ok := parseID(ctx, "userId")
	if !ok {
		return
	}

	if err := c.projectService.RemoveMember(ctx, projectID, ctx.GetUint("userID"), userID); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{"message": "项目成员已移除"})
}

// LeaveProject 退出当前项目。
// @Summary LeaveProject
// @Description DELETE /projects/{id}/members/me
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /projects/{id}/members/me [delete]
func (c *ProjectController) LeaveProject(ctx *gin.Context) {
	projectID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := c.projectService.LeaveProject(ctx, projectID, ctx.GetUint("userID")); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{"message": "已退出项目"})
}

// @Summary GetProjectStats
// @Description GET /projects/{id}/stats
// @Tags projects
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /projects/{id}/stats [get]
func (c *ProjectController) GetProjectStats(ctx *gin.Context) {
	projectID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	stats, err := c.projectService.GetProjectStats(ctx, projectID, ctx.GetUint("userID"))
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, stats)
}
