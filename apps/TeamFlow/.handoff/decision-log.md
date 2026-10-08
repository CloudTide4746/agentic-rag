# 决策注册表
记录所有重要的技术和架构决策。

---

## 2026-07-01 Service 层绕过 Repository 直连 storage.DB
**选择：** UserService.CreateUser/UpdateUser/DeleteUser 直接使用 `storage.DB`，不经过 `UserRepository`
**原因：** Create/Update/Delete 操作简单直接（无 JOIN、无关联预加载），Repository 抽象反而增加 boilerplate；Repository 聚焦查询封装
**放弃的备选：** 全部经 Repository — 因为简单写入不需要二次抽象

---

## 2026-07-01 UpdateUser 接收空密码不更新密码字段
**选择：** Service 层判断 `if password != ""` 才赋值；`User.BeforeUpdate` hook 用 `tx.Statement.Changed("Password")` 双重保护
**原因：** 避免"部分更新接口"把已哈希密码覆盖成空字符串或被二次哈希
**放弃的备选：** 一律要求传 password — 因为 DTO 难以区分"不改密码"和"清空密码"

---

## 2026-07-01 DeleteUser 返回 `用户不存在` 而非 silent success
**选择：** 检查 `result.Error` 优先，再看 `RowsAffected == 0`
**原因：** 业务上重复删除或删不存在的记录需要被调用方感知
**放弃的备选：** Silent success — 因为掩盖 bug、不利于幂等性排查
