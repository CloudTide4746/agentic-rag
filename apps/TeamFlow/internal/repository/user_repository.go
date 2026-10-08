package repository

import (
	"errors"
	"time"

	"teamflow/internal/cache"
	"teamflow/internal/model"

	"gorm.io/gorm"
)

// UserRepository 用户数据访问层

type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓储实例
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByID 根据主键 ID 查询用户
func (r *UserRepository) GetByID(id uint) (*model.User, error) {
	// cached：优先读缓存，未命中则查库
	userCache := &cache.UserCache{}
	var user model.User
	ok, err := userCache.GetUser(id, &user)
	if err != nil {
		// 防止缓存穿透
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if ok {
		return &user, nil
	}
	if err := r.db.First(&user, id).Error; err != nil {
		// 防止缓存穿透
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 缓存穿透，返回空值
			userCache.SetUser(id, nil, 5*time.Minute)
			return nil, nil
		}
		return nil, err
	}
	// 缓存用户信息
	userCache.SetUser(id, user, 5*time.Minute)
	return &user, nil
}

// GetByEmail 根据邮箱查询用户
func (r *UserRepository) GetByEmail(email string) (*model.User, error) {

	var user model.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByUsername 根据用户名查询用户
func (r *UserRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers 分页查询所有用户，返回用户列表和总数
func (r *UserRepository) ListUsers(offset, limit int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	if err := r.db.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// ListUsersByTeamID 根据团队 ID 分页查询成员
func (r *UserRepository) ListUsersByTeamID(teamID uint, offset, limit int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	query := r.db.Joins("JOIN team_members ON team_members.user_id = users.id").
		Where("team_members.team_id = ?", teamID)

	if err := query.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// ListUsersByProjectID 根据项目 ID 分页查询成员
func (r *UserRepository) ListUsersByProjectID(projectID uint, offset, limit int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	query := r.db.Joins("JOIN project_members ON project_members.user_id = users.id").
		Where("project_members.project_id = ?", projectID)

	if err := query.Model(&model.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}
