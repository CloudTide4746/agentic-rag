# UserRepository 查询函数设计文档

**日期**: 2026-07-01  
**状态**: 已批准

---

## 1. 需求

为 `UserRepository` 添加三个查询方法：

| 方法 | 用途 |
|------|------|
| `GetByID` | 根据主键 ID 查询用户 |
| `GetByEmail` | 根据邮箱查询用户 |
| `GetByUsername` | 根据用户名查询用户 |

---

## 2. 设计方案

**方案**: 直接方法（每个字段一个独立方法）

### 2.1 接口

```go
func (r *UserRepository) GetByID(id uint) (*model.User, error)
func (r *UserRepository) GetByEmail(email string) (*model.User, error)
func (r *UserRepository) GetByUsername(username string) (*model.User, error)
```

### 2.2 职责

- Repository 层只封装 GORM 查询，不做业务判断
- 所有方法使用 `r.db`，不接受额外 `tx` 参数
- 事务管理由 Service 层通过 `db.Begin()` 控制

### 2.3 错误处理

- 返回 `(*model.User, error)`，明确区分查询结果和错误
- 记录不存在时返回 `gorm.ErrRecordNotFound`
- 调用方通过 `errors.Is(err, gorm.ErrRecordNotFound)` 自行判断

---

## 3. 使用示例

### Repository 层实现

```go
func (r *UserRepository) GetByEmail(email string) (*model.User, error) {
    var user model.User
    err := r.db.Where("email = ?", email).First(&user).Error
    if err != nil {
        return nil, err
    }
    return &user, nil
}
```

### Service 层调用

```go
user, err := s.userRepo.GetByEmail(email)
if err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, errors.New("用户不存在")
    }
    return nil, fmt.Errorf("查询用户失败: %w", err)
}
```
