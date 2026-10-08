package controller

import (
	"net/http"
	"strconv"
	"teamflow/internal/service"

	"github.com/gin-gonic/gin"
)

type CommentController struct {
	service service.CommentService
}

func NewCommentController(service service.CommentService) *CommentController {

	return &CommentController{service: service}
}

// createCommentRequest 发表评论的请求体
type createCommentRequest struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

// CreateCommentController POST /tasks/:id/comments
// @Summary CreateCommentController
// @Description POST /tasks/{id}/comments
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body createCommentRequest true "Request body"
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id}/comments [post]
func (h *CommentController) CreateCommentController(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	var req createCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 从认证中间件注入的上下文中获取当前用户ID
	userID := c.GetUint("userID")

	comment, err := h.service.Create(service.CreateCommentInput{
		TaskID:  uint(taskID),
		UserID:  userID,
		Content: req.Content,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "评论发表成功", "comment": comment})
}

// GetCommentsController GET /tasks/:id/comments
// @Summary GetCommentsController
// @Description GET /tasks/{id}/comments
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id}/comments [get]
func (h *CommentController) GetCommentsController(c *gin.Context) {
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.service.GetByTaskID(uint(taskID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取评论失败"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// DeleteCommentController DELETE /tasks/:id/comments/:commentId
// @Summary DeleteCommentController
// @Description DELETE /tasks/{id}/comments/{commentId}
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Param commentId path string true "commentId ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id}/comments/{commentId} [delete]
func (h *CommentController) DeleteCommentController(c *gin.Context) {
	commentID, err := strconv.ParseUint(c.Param("commentId"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的评论ID"})
		return
	}

	operatorID := c.GetUint("userID")
	isAdmin := c.GetBool("isAdmin")

	if err := h.service.Delete(uint(commentID), operatorID, isAdmin); err != nil {
		switch err {
		case service.ErrCommentNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrCommentForbidden:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "评论已删除"})
}
