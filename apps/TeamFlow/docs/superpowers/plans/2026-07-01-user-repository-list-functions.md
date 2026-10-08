# UserRepository 列表查询函数实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `UserRepository` 添加 `ListUsers`、`ListUsersByTeamID`、`ListUsersByProjectID` 三个分页查询方法

**Architecture:** 三个方法挂在 `UserRepository` 上，使用 `r.db` 执行 GORM 查询，JOIN 中间表获取团队成员/项目成员，返回 `([]model.User, int64, error)`

**Tech Stack:** Go 1.26, GORM

---

### Task 1: 添加三个列表查询方法

**Files:**
- Modify: `internal/repository/user_repository.go`

- [ ] **Step 1: 读取当前文件**

当前内容：

```go
package repository

import (
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
	var user model.User
	err := r.db.First(&user, id).Error
	if err != nil {
		return nil, err
	}
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
```

- [ ] **Step 2: 在 `GetByUsername` 方法之后追加三个新方法**

在文件末尾追加：

```go
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
```

- [ ] **Step 3: 运行编译验证**

```bash
go build ./...
```
Expected: 无错误

- [ ] **Step 4: 提交**

```bash
git add internal/repository/user_repository.go docs/superpowers/specs/2026-07-01-user-repository-list-functions-design.md docs/superpowers/plans/2026-07-01-user-repository-list-functions.md
git commit -m "feat: add UserRepository list query methods (ListUsers, ListUsersByTeamID, ListUsersByProjectID)"
```
