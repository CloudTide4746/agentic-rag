package dto

type ProfileResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Bio      string `json:"bio"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required,min=6,max=20"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=72"`
}

type UpdateProfileRequest struct {
	Username string `json:"username"`
	Bio      string `json:"bio"`
}
