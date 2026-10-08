# UserService CRUD 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `UserService` 添加 CreateUser/UpdateUser/DeleteUser 方法，并为 `User` 模型添加 BeforeUpdate hook

**Architecture:** Service 直接使用 `storage.DB`（不经过 Repository），BeforeCreate/BeforeUpdate hook 处理密码 bcrypt 加密

**Tech Stack:** Go 1.26, GORM, `golang.org/x/crypto/bcrypt`

---

### Task 1: 添加 User.BeforeUpdate hook

**Files:**
- Modify: `internal/model/user.go`

- [ ] **Step 1: 在 BeforeCreate 之后添加 BeforeUpdate hook**

在 `internal/model/user.go` 的 `BeforeCreate` 方法后面追加：

```go
// BeforeUpdate GORM 钩子：更新用户前自动加密密码
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if u.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashed)
	}
	return nil
}
```

- [ ] **Step 2: 编译验证**

```bash
go build ./...
```
Expected: 无错误

- [ ] **Step 3: 提交**

```bash
git add internal/model/user.go
git commit -m "feat: add User.BeforeUpdate hook for password hashing"
```

---

### Task 2: 添加 UserService CRUD 方法

**Files:**
- Modify: `internal/service/user_service.go`

- [ ] **Step 1: 替换文件内容**

将 `internal/service/user_service.go` 替换为以下完整实现：

```go
package service

import (
	"errors"
	"fmt"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"teamflow/internal/model"
	"teamflow/internal/repository"
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

// CreateUser 创建用户，密码由 BeforeCreate hook 自动加密
func (s *UserService) CreateUser(username, email, password, avatar, nickname string) (*model.User, error) {
	user := model.User{
		Username: username,
		Email:    email,
		Password: password,
		Avatar:   avatar,
		Nickname: nickname,
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
func (s *UserService) UpdateUser(id uint, username, email, password, avatar, nickname string, status int8) (*model.User, error) {
	var user model.User
	if err := storage.DB.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	user.Username = username
	user.Email = email
	user.Password = password
	user.Avatar = avatar
	user.Nickname = nickname
	user.Status = status

	if err := storage.DB.Save(&user).Error; err != nil {
		if isMySQLDuplicate(err) {
			return nil, fmt.Errorf("用户名或邮箱已存在")
		}
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}
	return &user, nil
}

// DeleteUser 软删除用户，GORM 自动设置 deleted_at
func (s *UserService) DeleteUser(id uint) error {
	result := storage.DB.Delete(&model.User{}, id)
	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在")
	}
	return result.Error
}

// isMySQLDuplicate 检查是否为 MySQL 1062 唯一键冲突
func isMySQLDuplicate(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
```

- [ ] **Step 2: 编译验证**

```bash
go build ./...
```
Expected: 无错误

- [ ] **Step 3: 验证 go.mod（确保 go-sql-driver/mysql 从 indirect 变为 direct）**

```bash
go mod tidy
```
Expected: 无错误，`github.com/go-sql-driver/mysql` 在 go.mod require 块中变为 direct

- [ ] **Step 4: 提交**

```bash
git add internal/service/user_service.go go.mod go.sum
git commit -m "feat: add UserService CRUD methods (CreateUser, UpdateUser, DeleteUser)"
```
