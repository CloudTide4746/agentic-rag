package controller

import (
	"net/http"
	"path/filepath"
	"strconv"
	"teamflow/internal/service"

	"github.com/gin-gonic/gin"
)

type AttachmentController struct {
	attachmentSvc service.AttachmentServiceInterface
}

func NewAttachmentController(attachmentSvc service.AttachmentServiceInterface) *AttachmentController {
	return &AttachmentController{attachmentSvc: attachmentSvc}
}

// UploadAttachment 上传附件
// @Summary UploadAttachment
// @Description POST /tasks/{id}/attachments
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Param file formData file true "上传文件"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id}/attachments [post]
func (h *AttachmentController) UploadAttachment(ctx *gin.Context) {
	taskID, err := strconv.ParseUint(ctx.Param("taskID"), 10, 64)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "taskID is not a valid number"})
		return
	}

	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 10<<20) //位运算 10MB

	file, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}
	uploaderID := ctx.GetUint("uploaderID")

	attachment, err := h.attachmentSvc.UploadAttachment(uint(taskID), uploaderID, file)
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(201, gin.H{"message": "attachment uploaded successfully", "attachment": attachment})
}

// GetAttachments 获取任务下的所有附件
// @Summary GetAttachments
// @Description GET /tasks/{id}/attachments
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id}/attachments [get]
func (h *AttachmentController) GetAttachments(ctx *gin.Context) {
	taskID, err := strconv.ParseUint(ctx.Param("taskID"), 10, 64)
	if err != nil {
		ctx.JSON(400, gin.H{"error": "taskID is not a valid number"})
		return
	}

	attachments, err := h.attachmentSvc.GetAttachments(uint(taskID))
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{"attachments": attachments})
}

// DeleteAttachmentHandler DELETE /attachments/:id
// @Summary DeleteAttachmentHandler
// @Description DELETE /attachments/{id}
// @Tags attachments
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /attachments/{id} [delete]
func (h *AttachmentController) DeleteAttachmentHandler(c *gin.Context) {
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的附件ID"})
		return
	}

	operatorID := c.GetUint("userID")
	isAdmin := c.GetBool("isAdmin")

	if err := h.attachmentSvc.DeleteAttachment(uint(attachmentID), operatorID, isAdmin); err != nil {
		switch err {
		case service.ErrAttachmentNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		case service.ErrAttachmentForbidden:
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "附件已删除"})
}

// DownloadAttachmentHandler GET /attachments/:id/download
// 使用 Content-Disposition: attachment 触发浏览器下载，并以原始文件名命名
// @Summary DownloadAttachmentHandler
// @Description GET /attachments/{id}/download
// @Tags attachments
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /attachments/{id}/download [get]
func (h *AttachmentController) DownloadAttachmentHandler(c *gin.Context) {
	attachmentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的附件ID"})
		return
	}

	attachment, err := h.attachmentSvc.GetByID(uint(attachmentID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.Header("Content-Disposition",
		`attachment; filename="`+filepath.Base(attachment.OriginalName)+`"`)
	c.Header("Content-Type", "application/octet-stream")
	c.File(attachment.FilePath)
	// 标准下载实现：
	// 1. 查询附件信息（获取 FilePath 和 OriginalName）
	// 2. 设置 Content-Disposition 头，触发浏览器下载
	// 3. 使用 c.File() 发送文件
	//
	// 示例（假设已获取到 attachment 对象）：
	//   c.Header("Content-Disposition", `attachment; filename="`+attachment.OriginalName+`"`)
	//   c.File(attachment.FilePath)
}
