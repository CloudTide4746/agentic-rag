# UserService CRUD 设计文档

**日期**: 2026-07-01  
**状态**: 已批准

---

## 1. 需求

为 `UserService` 添加三个 CRUD 方法：

| 方法 | 用途 | 备注 |
|------|------|------|
| `CreateUser` | 创建用户 | 密码 bcrypt 加密（BeforeCreate hook） |
| `UpdateUser` | 更新用户信息 | 全量覆盖，密码加密（BeforeUpdate hook） |
| `DeleteUser` | 软删除用户 | GORM 自动 UPDATE deleted_at |

---

## 2. 设计方案

### 2.1 分层

Service 直接使用 `storage.DB`，不经过 Repository 层。

**理由**: Create/Update/Delete 操作简单直接，不需要 Repository 封装；Repository 聚焦查询。

### 2.2 接口

```go
func (s *UserService) CreateUser(username, email, password, avatar, nickname string) (*model.User, error)
func (s *UserService) UpdateUser(id uint, username, email, password, avatar, nickname string, status int8) (*model.User, error)
func (s *UserService) DeleteUser(id uint) error
```

### 2.3 CreateUser

- **必填**: username, email, password
- **可选**: avatar, nickname（空字符串 = BeforeCreate hook 设置 nickname=username）
- **加密**: `User.BeforeCreate` hook 自动 bcrypt
- **唯一性冲突**: 检查 MySQL 1062 错误码 → 返回 "用户名或邮箱已存在"
- **返回**: 创建后的 `*model.User`（含自增 ID、时间戳）

```go
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
```

### 2.4 UpdateUser

- **全量更新**: Username/Email/Password/Avatar/Nickname/Status 全部覆盖
- **密码加密**: 新增 `User.BeforeUpdate` hook — Password != "" 时 bcrypt
- **存在性检查**: 先 GetByID，不存在 → "用户不存在"
- **实现**: `db.Save(&user)`（Save 是 INSERT OR UPDATE，因为 user 有主键会走 UPDATE）

```go
func (s *UserService) UpdateUser(id uint, username, email, password, avatar, nickname string, status int8) (*model.User, error) {
    var user model.User
    if err := storage.DB.First(&user, id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, fmt.Errorf("用户不存在")
        }
        return nil, fmt.Errorf("查询用户失败: %w", err)
    }

    user.Username = username
    user.Email    = email
    user.Password = password
    user.Avatar   = avatar
    user.Nickname = nickname
    user.Status   = status

    if err := storage.DB.Save(&user).Error; err != nil {
        if isMySQLDuplicate(err) {
            return nil, fmt.Errorf("用户名或邮箱已存在")
        }
        return nil, fmt.Errorf("更新用户失败: %w", err)
    }
    return &user, nil
}
```

### 2.5 DeleteUser

- **软删除**: `db.Delete(&model.User{}, id)` → GORM 自动 SET deleted_at = NOW()
- **存在性**: `RowsAffected == 0` → "用户不存在"
- **已删除记录**: 再次 Delete 返回 RowsAffected 0（已软删），同样返回 "用户不存在"

```go
func (s *UserService) DeleteUser(id uint) error {
    result := storage.DB.Delete(&model.User{}, id)
    if result.RowsAffected == 0 {
        return fmt.Errorf("用户不存在")
    }
    return result.Error
}
```

### 2.6 BeforeUpdate Hook（新增，挂在 User 模型）

```go
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

### 2.7 isMySQLDuplicate 辅助函数

```go
func isMySQLDuplicate(err error) bool {
    var mysqlErr *mysql.MySQLError
    return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
```

---

## 3. 错误处理汇总

| 场景 | 返回错误 |
|------|----------|
| 创建成功 | nil |
| 用户名/邮箱重复 | `"用户名或邮箱已存在"` |
| 数据库错误 | `"创建用户失败: <err>"` |
| 更新成功 | nil |
| 用户不存在（更新） | `"用户不存在"` |
| 更新时唯一冲突 | `"用户名或邮箱已存在"` |
| 删除成功 | nil |
| 用户不存在（删除） | `"用户不存在"` |

---

## 4. 涉及文件

| 文件 | 操作 |
|------|------|
| `internal/service/user_service.go` | 修改 — 添加 CreateUser/UpdateUser/DeleteUser + isMySQLDuplicate |
| `internal/model/user.go` | 修改 — 添加 BeforeUpdate hook |

## 5. 依赖

- `database/sql` 已由 `mysql` driver 间接导入
- 需要 `go get github.com/go-sql-driver/mysql`（或已有）
- 需要 `errors` 标准库
- 需要 `gorm.io/gorm`
