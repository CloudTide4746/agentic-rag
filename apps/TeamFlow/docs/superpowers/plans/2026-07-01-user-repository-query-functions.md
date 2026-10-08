# UserRepository 查询函数实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `UserRepository` 添加 `GetByID`、`GetByEmail`、`GetByUsername` 三个查询方法和构造函数

**Architecture:** 三个方法直接挂在 `UserRepository` 上，使用 `r.db` 执行 GORM 查询，返回 `(*model.User, error)`，由调用方区分错误类型

**Tech Stack:** Go 1.26, GORM

---

### Task 1: 添加构造函数和三个查询方法

**Files:**
- Modify: `internal/repository/user_repository.go`

- [ ] **Step 1: 读取当前文件**

当前内容：

```go
package repository

import (
	"gorm.io/gorm"
)

// UserRepository 用户数据访问层
type UserRepository struct {
	db *gorm.DB
}
```

- [ ] **Step 2: 替换为完整实现**

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

- [ ] **Step 3: 运行编译验证**

```bash
go build ./...
```
Expected: 无错误

- [ ] **Step 4: 提交**

```bash
git add internal/repository/user_repository.go
git commit -m "feat: add UserRepository query methods (GetByID, GetByEmail, GetByUsername)"
```
