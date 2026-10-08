package apperr

import (
	"fmt"
)

// AppError 业务错误类型，携带业务码、说明和 HTTP 状态码
type AppError struct {
	Code       int    // 业务错误码（对应响应包中的 code 字段）
	Message    string // 面向用户的错误说明
	HTTPStatus int    // 对应的 HTTP 状态码
	Cause      error  // 原始错误（内部日志用，不暴露给用户）
}

// Error 实现 error 接口
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Unwrap 支持 errors.Is / errors.As 链式解包
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New 创建新的业务错误
func New(code int, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Wrap 包裹原始错误，保留调用链信息
func Wrap(base *AppError, cause error) *AppError {
	return &AppError{
		Code:       base.Code,
		Message:    base.Message,
		HTTPStatus: base.HTTPStatus,
		Cause:      cause,
	}
}
