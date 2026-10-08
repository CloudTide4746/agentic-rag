# UserRepository 列表查询函数设计文档

**日期**: 2026-07-01  
**状态**: 已批准

---

## 1. 需求

为 `UserRepository` 添加三个列表查询方法：

| 方法 | 用途 |
|------|------|
| `ListUsers` | 分页查询所有用户 |
| `ListUsersByTeamID` | 根据团队 ID 分页查询成员 |
| `ListUsersByProjectID` | 根据项目 ID 分页查询成员 |

---

## 2. 设计方案

### 2.1 接口

```go
func (r *UserRepository) ListUsers(offset, limit int) ([]model.User, int64, error)
func (r *UserRepository) ListUsersByTeamID(teamID uint, offset, limit int) ([]model.User, int64, error)
func (r *UserRepository) ListUsersByProjectID(projectID uint, offset, limit int) ([]model.User, int64, error)
```

- `offset` / `limit` 分页参数
- 返回用户切片 + 总数 + 错误
- `limit <= 0` 时默认 10，`offset < 0` 时默认 0

### 2.2 查询方式

- **ListUsers**: 直接 `Find` + `Count`
- **ListUsersByTeamID**: JOIN `team_members` 中间表，`WHERE team_members.team_id = ?`
- **ListUsersByProjectID**: JOIN `project_members` 中间表，`WHERE project_members.project_id = ?`

### 2.3 职责

- Repository 层只封装 GORM 查询，不做业务判断
- 事务由 Service 层管理
- 软删除用户自动排除（GORM 默认行为）

### 2.4 错误处理

- 返回 `([]model.User, int64, error)`
- 空结果返回空切片，不返回 error
- 数据库错误直接返回

---

## 3. 使用示例

### Repository 层实现

```go
func (r *UserRepository) ListUsers(offset, limit int) ([]model.User, int64, error) {
	var users []model.User
	var total int64

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	r.db.Model(&model.User{}).Count(&total)

	err := r.db.Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

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

	query.Model(&model.User{}).Count(&total)

	err := query.Offset(offset).Limit(limit).Find(&users).Error
	if err != nil {
		return nil, 0, err
	}
	return users, total, nil
}
```

### Service 层调用

```go
users, total, err := s.userRepo.ListUsers(0, 20)
if err != nil {
    return nil, fmt.Errorf("查询用户列表失败: %w", err)
}
// users 可能为空切片，total 为 0
```
