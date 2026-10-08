package middleware

import (
	"errors"
	"teamflow/pkg/apperr"
	"teamflow/pkg/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ErrorHandler 统一拦截 Controller 通过 c.Error() 附加的错误
// 用法：在 Controller 中调用 c.Error(err) 后直接 return，由此中间件统一响应

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // 继续执行后续中间件
		if len(c.Errors) == 0 {
			return
		}
		err := c.Errors.Last().Err

		var apperr *apperr.AppError

		if errors.As(err, &apperr) {
			// 已知业务错误：返回对应 HTTP 状态码和业务码
			logger.Warn("business error",
				zap.Int("code", apperr.Code),
				zap.String("message", apperr.Message),
				zap.String("path", c.Request.URL.Path),
				zap.Error(apperr.Cause),
			)
			c.JSON(apperr.HTTPStatus, gin.H{
				"code":    apperr.Code,
				"message": apperr.Message,
				"data":    nil,
			})
			return
		}

		// 未知错误：记录完整日志，对外返回通用 500
		logger.Error("unexpected error",
			zap.String("path", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.Error(err),
		)
		response.ServerError(c)
	}
}
