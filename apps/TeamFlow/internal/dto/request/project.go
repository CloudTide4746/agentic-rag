package request

import "time"

// CreateProjectRequest 创建项目请求。
type CreateProjectRequest struct {
	Name        string     `json:"name" binding:"required,min=2,max=100"`
	Description string     `json:"description" binding:"max=500"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
}

// UpdateProjectRequest 更新项目请求。
type UpdateProjectRequest struct {
	Name        *string    `json:"name" binding:"omitempty,min=2,max=100"`
	Description *string    `json:"description" binding:"omitempty,max=500"`
	Status      *string    `json:"status" binding:"omitempty,oneof=active archived"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
}

// AddProjectMemberRequest 添加项目成员请求。
type AddProjectMemberRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=admin member"`
}
