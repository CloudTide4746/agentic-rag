package dto

import "time"

type AssignTaskRequest struct {
	AssigneeID uint `json:"assignee_id" binding:"required,gt=0"`
}
type TaskResponse struct {
	ID         uint      `json:"id"`
	Title      string    `json:"title"`
	AssigneeID *uint     `json:"assignee_id"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type TaskQuery struct {
	// 筛选条件
	Status       string `form:"status"`         // 任务状态
	Priority     string `form:"priority"`       // 优先级
	AssigneeID   uint   `form:"assignee_id"`    // 负责人 ID
	DueDateStart string `form:"due_date_start"` // 截止日期起始 YYYY-MM-DD
	DueDateEnd   string `form:"due_date_end"`   // 截止日期结束 YYYY-MM-DD

	// 排序参数
	SortBy  string `form:"sort_by"`  // 排序字段：created_at / priority / due_date
	SortDir string `form:"sort_dir"` // 排序方向：asc / desc

	// 分页参数
	Page     int `form:"page"`      // 页码，从 1 开始
	PageSize int `form:"page_size"` // 每页数量，默认 20
}

func (q *TaskQuery) SetDefaults() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20 // 默认每页 20 条，最多 100 条
	}
	if q.SortBy == "" {
		q.SortBy = "created_at"
	}
	if q.SortDir == "" {
		q.SortDir = "desc"
	}
}
