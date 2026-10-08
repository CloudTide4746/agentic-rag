package handler

// TaskHandler 处理任务相关的请求和响应
import (
	"net/http"
	"teamflow/internal/dto"
	"teamflow/internal/repository"
	"teamflow/internal/service"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskService *service.TaskService
	repo        repository.TaskRepository
}

func NewTaskHandler(taskService *service.TaskService, repo repository.TaskRepository) *TaskHandler {

	return &TaskHandler{taskService: taskService, repo: repo}
}

// ListTasks 处理任务列表查询请求
// GET /api/v1/tasks?status=todo&priority=high&sort_by=due_date&sort_dir=asc&page=1&page_size=20
func (h *TaskHandler) ListTasks(c *gin.Context) {
	var query dto.TaskQuery

	// 使用 form 标签从 Query String 绑定参数
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 填充默认值
	query.SetDefaults()

	tasks, total, err := h.repo.ListTasks(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      tasks,
		"total":     total,
		"page":      query.Page,
		"page_size": query.PageSize,
	})
}
