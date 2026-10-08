package request

// CreateTeamRequest 创建团队请求。
type CreateTeamRequest struct {
	Name        string `json:"name" binding:"required,min=2,max=100"`
	Description string `json:"description" binding:"max=500"`
}

// UpdateTeamRequest 更新团队请求。指针用于区分“未传”与“传空字符串”。
type UpdateTeamRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=100"`
	Description *string `json:"description" binding:"omitempty,max=500"`
	Avatar      *string `json:"avatar" binding:"omitempty,max=500"`
}

// AddTeamMemberRequest 添加团队成员请求。
type AddTeamMemberRequest struct {
	UserID uint   `json:"user_id" binding:"required"`
	Role   string `json:"role" binding:"required,oneof=admin member"`
}
