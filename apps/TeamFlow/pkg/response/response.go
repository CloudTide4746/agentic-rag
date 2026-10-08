package response

import (
	"net/http"
	"teamflow/internal/dto"
	"teamflow/internal/model"
	"time"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code      int         `json:"code"`      // 业务状态码
	Message   string      `json:"message"`   // 说明信息
	Data      interface{} `json:"data"`      // 响应数据
	Timestamp int64       `json:"timestamp"` // Unix 时间戳
}

func newResponse(code int, message string, data interface{}) Response {
	return Response{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
	}
}

// Success 返回成功响应，HTTP 200
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, newResponse(0, "success", data))
}

// Error 返回业务错误响应，HTTP 200（业务层面的错误）
func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, newResponse(code, message, nil))
}

// Unauthorized 返回未认证响应，HTTP 401
func Unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, newResponse(1001, "请先登录", nil))
}

// Forbidden 返回无权限响应，HTTP 403
func Forbidden(c *gin.Context) {
	c.JSON(http.StatusForbidden, newResponse(1002, "权限不足", nil))
}

// NotFound 返回资源不存在响应，HTTP 404
func NotFound(c *gin.Context) {
	c.JSON(http.StatusNotFound, newResponse(1003, "资源不存在", nil))
}

// BadRequest 返回请求参数错误响应，HTTP 400
func BadRequest(c *gin.Context, message string) {
	c.JSON(http.StatusBadRequest, newResponse(1004, message, nil))
}

// ServerError 返回服务器内部错误响应，HTTP 500
func ServerError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, newResponse(1005, "服务器内部错误", nil))
}

func SuccessPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, newResponse(0, "success", dto.PageResult[model.Project]{
		List:  list.([]model.Project),
		Total: total,
		Page:  page,
		Size:  pageSize,
	}))
}
