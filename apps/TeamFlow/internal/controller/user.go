package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"teamflow/internal/dto"
	"teamflow/internal/dto/request"
	"teamflow/internal/middleware"
	"teamflow/internal/service"
	"teamflow/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

type UserController struct {
	userService *service.UserService
	onlineSvc   *service.OnlineService
}

func NewUserController(userService *service.UserService, onlineSvc *service.OnlineService) *UserController {
	return &UserController{
		userService: userService,
		onlineSvc:   onlineSvc,
	}
}

// @Summary Log in
// @Description Authenticate with username and password and receive a JWT.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body request.LoginRequest true "Login credentials"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Router /auth/login [post]
func (uc *UserController) Login(ctx *gin.Context) {
	var req request.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := uc.userService.Login(&req)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	// 登录成功后标记用户在线（Redis）
	if resp.User != nil {
		_ = uc.onlineSvc.SetOnline(ctx, resp.User.ID)
	}
	response.Success(ctx, resp)
}

// UploadAvatar 上传用户头像
// @Summary UploadAvatar
// @Description POST /users/me/avatar
// @Tags users
// @Accept json
// @Produce json
// @Param file formData file true "上传文件"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /users/me/avatar [post]
func (uc *UserController) UploadAvatar(ctx *gin.Context) {
	userName, ok := middleware.GetCurrentUsername(ctx)
	if !ok {
		response.Unauthorized(ctx)
		return
	}
	file, err := ctx.FormFile("avatar")
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if file.Size > 1024*1024*5 {
		response.Error(ctx, http.StatusBadRequest, "头像大小不能超过5MB")
		return
	}
	ext := filepath.Ext(file.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
		response.Error(ctx, http.StatusBadRequest, "头像格式错误")
		return
	}
	filename := fmt.Sprintf("avatar%s_%d%s", userName, time.Now().Unix(), ext)
	savepath := filepath.Join("uploads", "avatar", filename)
	if err := os.MkdirAll(filepath.Dir(savepath), 0755); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := ctx.SaveUploadedFile(file, savepath); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := uc.userService.ChangeAvatar(userName, filename); err != nil {
		_ = os.Remove(savepath)
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, gin.H{"message": "上传头像成功", "data": filename})
}

// GetProfile 获取用户信息
// @Summary GetProfile
// @Description GET /users/me
// @Tags users
// @Accept json
// @Produce json
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /users/me [get]
func (c *UserController) GetProfile(ctx *gin.Context) {
	userName, ok := middleware.GetCurrentUsername(ctx)
	if !ok {
		response.Unauthorized(ctx)
		return
	}
	profile, err := c.userService.GetProfile(userName)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, profile)
}

// UpdateProfile 更新用户个人信息
// @Summary UpdateProfile
// @Description PUT /users/me
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.UpdateProfileRequest true "Request body"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /users/me [put]
func (uc *UserController) UpdateProfile(ctx *gin.Context) {
	var req dto.UpdateProfileRequest
	userName, ok := middleware.GetCurrentUsername(ctx)
	if !ok {
		response.Unauthorized(ctx)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := uc.userService.UpdateProfile(userName, &req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, nil)
}

// ChangePassword 更新用户密码
// @Summary ChangePassword
// @Description PUT /users/me/password
// @Tags users
// @Accept json
// @Produce json
// @Param request body dto.ChangePasswordRequest true "Request body"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /users/me/password [put]
func (uc *UserController) ChangePassword(ctx *gin.Context) {
	var req dto.ChangePasswordRequest
	userName, ok := middleware.GetCurrentUsername(ctx)
	if !ok {
		response.Unauthorized(ctx)
		return
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	if err := uc.userService.ChangePassword(userName, &req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, nil)
}

// Register 注册
// @Summary Register
// @Description POST /auth/register
// @Tags auth
// @Accept json
// @Produce json
// @Param request body request.RegisterRequest true "Request body"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Router /auth/register [post]
func (c *UserController) Register(ctx *gin.Context) {
	var req request.RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	userResp, err := c.userService.CreateUser(&req)
	if err != nil {
		response.Error(ctx, http.StatusBadRequest, err.Error())
		return
	}
	response.Success(ctx, userResp)
}
