package response

// 错误码常量定义
// 按模块分段，便于快速定位问题所在模块
const (
	// CodeSuccess 成功
	CodeSuccess = 0

	// ==================== 1xxx 通用错误 ====================

	// CodeUnauthorized 未登录或 Token 失效
	CodeUnauthorized = 1001
	// CodeForbidden 无操作权限
	CodeForbidden = 1002
	// CodeNotFound 资源不存在
	CodeNotFound = 1003
	// CodeBadRequest 请求参数错误
	CodeBadRequest = 1004
	// CodeServerError 服务器内部错误
	CodeServerError = 1005
	// CodeTooManyRequests 请求过于频繁（限流）
	CodeTooManyRequests = 1006
	// CodeValidationFailed 参数校验失败
	CodeValidationFailed = 1007

	// ==================== 2xxx 用户模块 ====================

	// CodeUserNotFound 用户不存在
	CodeUserNotFound = 2001
	// CodePasswordIncorrect 密码错误
	CodePasswordIncorrect = 2002
	// CodeUsernameExists 用户名已存在
	CodeUsernameExists = 2003
	// CodeEmailExists 邮箱已被注册
	CodeEmailExists = 2004
	// CodeTokenExpired Token 已过期
	CodeTokenExpired = 2005
	// CodeTokenInvalid Token 无效
	CodeTokenInvalid = 2006

	// ==================== 3xxx 任务模块 ====================

	// CodeTaskNotFound 任务不存在
	CodeTaskNotFound = 3001
	// CodeTaskAlreadyDone 任务已完成，不可重复操作
	CodeTaskAlreadyDone = 3002
	// CodeTaskAssignFailed 任务分配失败
	CodeTaskAssignFailed = 3003
	// CodeTaskStatusInvalid 任务状态不合法
	CodeTaskStatusInvalid = 3004

	// ==================== 4xxx 评论模块 ====================

	// CodeCommentNotFound 评论不存在
	CodeCommentNotFound = 4001
	// CodeCommentTooLong 评论内容超出长度限制
	CodeCommentTooLong = 4002
	// CodeCommentForbidden 无权操作他人评论
	CodeCommentForbidden = 4003

	// ==================== 5xxx 文件模块 ====================

	// CodeFileNotFound 文件不存在
	CodeFileNotFound = 5001
	// CodeFileTooLarge 文件大小超出限制
	CodeFileTooLarge = 5002
	// CodeFileTypeNotAllowed 文件类型不允许上传
	CodeFileTypeNotAllowed = 5003
	// CodeFileUploadFailed 文件上传失败
	CodeFileUploadFailed = 5004
)
