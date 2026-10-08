package service

import (
	"errors"
	"fmt"
	"strconv"
	"teamflow/internal/cache"
	"teamflow/internal/dto"
	"teamflow/pkg/jwt"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"teamflow/internal/dto/request"
	"teamflow/internal/model"
	"teamflow/internal/repository"
	"teamflow/pkg/utils"
	"teamflow/storage"
)

// UserService 用户业务逻辑层
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService 创建用户服务实例
func NewUserService() *UserService {
	return &UserService{
		userRepo: repository.NewUserRepository(storage.DB),
	}
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

// Login 用户登录
func (s *UserService) Login(req *request.LoginRequest) (LoginResponse, error) {
	var user model.User
	result := storage.DB.Where("username = ?", req.Username).First(&user)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return LoginResponse{}, fmt.Errorf("用户名不存在")
	}
	if result.Error != nil {
		return LoginResponse{}, fmt.Errorf("查询用户失败: %w", result.Error)
	}
	//检验密码是否正确
	if err := utils.CheckPassword(user.Password, req.Password); err != nil {
		return LoginResponse{}, fmt.Errorf("密码错误")
	}
	token, err := jwt.GenerateToken(user.ID, user.Username, "user")
	if err != nil {
		return LoginResponse{}, fmt.Errorf("生成token失败: %w", err)
	}
	userCache := &cache.UserCache{}
	if err := userCache.SetUserToken(
		strconv.FormatUint(uint64(user.ID), 10),
		token,
		1*time.Hour,
	); err != nil {
		return LoginResponse{}, fmt.Errorf("保存登录会话失败: %w", err)
	}
	//清空密码
	user.Password = ""
	return LoginResponse{
		Token: token,
		User:  &user,
	}, nil
}

// Logout 用户注销
func (s *UserService) Logout(userID uint) error {
	// 检查用户是否存在
	var user model.User
	if err := storage.DB.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}
	// 从缓存中删除用户token
	userCache := &cache.UserCache{}
	return userCache.DeleteUserToken(strconv.FormatUint(uint64(userID), 10))
}

// CreateUser 创建用户，密码由 BeforeCreate hook 自动加密
func (s *UserService) CreateUser(req *request.RegisterRequest) (*model.User, error) {
	user := model.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Avatar:   "",
		Nickname: "",
	}
	if err := storage.DB.Create(&user).Error; err != nil {
		if isMySQLDuplicate(err) {
			return nil, fmt.Errorf("用户名或邮箱已存在")
		}
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	return &user, nil
}

// UpdateUser 全量更新用户信息，密码由 BeforeUpdate hook 自动加密
func (s *UserService) UpdateUser(id uint, username, email, currentPassword, avatar, nickname string, status int8) (*model.User, error) {
	// GetByID 内部负责“先缓存、未命中查库、回填 user:{id}”
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("用户不存在")
	}

	if err := utils.CheckPassword(user.Password, currentPassword); err != nil {
		return nil, fmt.Errorf("密码错误")
	}

	user.Username = username
	user.Email = email
	user.Avatar = avatar
	user.Nickname = nickname
	user.Status = status

	if err := storage.DB.Save(user).Error; err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	// 必须删 user:{id}，由 UserCache 内部统一组装键
	if err := (&cache.UserCache{}).DeleteUser(id); err != nil {
		// DB 已成功；这里更适合记录错误并异步重试，而非告诉客户端“更新失败”
	}

	return user, nil
}

// DeleteUser 软删除用户，GORM 自动设置 deleted_at
func (s *UserService) DeleteUser(id uint) error {
	result := storage.DB.Delete(&model.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除用户失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在")
	}
	// 缓存更新
	userCache := &cache.UserCache{}
	if err := userCache.DeleteUser(id); err != nil {
		return fmt.Errorf("删除缓存失败: %w", err)
	}
	return nil
}

// isMySQLDuplicate 检查是否为 MySQL 1062 唯一键冲突
func isMySQLDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

// GetProfile 获取用户个人信息
func (s *UserService) GetProfile(username string) (*dto.ProfileResponse, error) {
	var user model.User
	if err := storage.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &dto.ProfileResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Avatar:   user.Avatar,
		Bio:      user.Nickname,
	}, nil
}

// UpdateProfile 更新用户个人信息
func (s *UserService) UpdateProfile(username string, req *dto.UpdateProfileRequest) error {
	var user model.User
	if err := storage.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}
	//格式校验
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.Bio != "" {
		user.Nickname = req.Bio
	}
	if err := storage.DB.Save(&user).Error; err != nil {
		if isMySQLDuplicate(err) {
			return fmt.Errorf("用户名或邮箱已存在")
		}
		return fmt.Errorf("更新用户失败: %w", err)
	}
	return nil
}

// ChangePassword 更新用户密码
func (s *UserService) ChangePassword(username string, req *dto.ChangePasswordRequest) error {
	var user model.User
	if err := storage.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}
	//检验密码是否正确
	err := utils.CheckPassword(user.Password, req.OldPassword)
	if err != nil {
		return fmt.Errorf("旧密码错误")
	}
	//更新密码
	user.Password, err = utils.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}
	return storage.DB.Model(&user).Update("password", user.Password).Error
}

// ChangeAvatar 更新用户头像
func (s *UserService) ChangeAvatar(username string, avatar string) error {
	var user model.User
	if err := storage.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("用户不存在")
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}
	user.Avatar = avatar
	return storage.DB.Save(&user).Error
}
