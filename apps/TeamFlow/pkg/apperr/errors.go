// pkg/apperr/errors.go
package apperr

import "net/http"

// 通用错误
var (
	ErrUnauthorized = New(1001, "请先登录", http.StatusUnauthorized)
	ErrForbidden    = New(1002, "权限不足", http.StatusForbidden)
	ErrNotFound     = New(1003, "资源不存在", http.StatusNotFound)
	ErrBadRequest   = New(1004, "请求参数错误", http.StatusBadRequest)
	ErrServerError  = New(1005, "服务器内部错误", http.StatusInternalServerError)
)

// 用户模块错误
var (
	ErrUserNotFound      = New(2001, "用户不存在", http.StatusNotFound)
	ErrPasswordIncorrect = New(2002, "用户名或密码错误", http.StatusUnauthorized)
	ErrUsernameExists    = New(2003, "用户名已存在", http.StatusConflict)
	ErrEmailExists       = New(2004, "邮箱已被注册", http.StatusConflict)
	ErrTokenExpired      = New(2005, "登录已过期，请重新登录", http.StatusUnauthorized)
	ErrTokenInvalid      = New(2006, "无效的认证令牌", http.StatusUnauthorized)
)

// 任务模块错误
var (
	ErrTaskNotFound      = New(3001, "任务不存在", http.StatusNotFound)
	ErrTaskAlreadyDone   = New(3002, "任务已完成", http.StatusConflict)
	ErrTaskAssignFailed  = New(3003, "任务分配失败", http.StatusBadRequest)
	ErrTaskStatusInvalid = New(3004, "无效的任务状态", http.StatusBadRequest)
)

// 评论模块错误
var (
	ErrCommentNotFound  = New(4001, "评论不存在", http.StatusNotFound)
	ErrCommentTooLong   = New(4002, "评论内容超出长度限制", http.StatusBadRequest)
	ErrCommentForbidden = New(4003, "无权操作他人评论", http.StatusForbidden)
)

// 文件模块错误
var (
	ErrFileNotFound       = New(5001, "文件不存在", http.StatusNotFound)
	ErrFileTooLarge       = New(5002, "文件大小超出限制", http.StatusBadRequest)
	ErrFileTypeNotAllowed = New(5003, "不支持的文件类型", http.StatusBadRequest)
	ErrFileUploadFailed   = New(5004, "文件上传失败", http.StatusInternalServerError)
)
