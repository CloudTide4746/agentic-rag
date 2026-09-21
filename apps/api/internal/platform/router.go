package platform

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func NewRouter(log *zap.Logger) *gin.Engine {
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.HandleMethodNotAllowed = true
	r.Use(func(c *gin.Context) {
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		requestID := hex.EncodeToString(id)
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		started := time.Now()
		c.Next()
		log.Info("http_request", zap.String("request_id", requestID), zap.String("method", c.Request.Method), zap.String("route", c.FullPath()), zap.Int("status", c.Writer.Status()), zap.Duration("duration", time.Since(started)))
	})
	r.Use(func(c *gin.Context) {
		defer func() {
			if recover() != nil {
				log.Error("request_panic", zap.String("request_id", c.GetString("request_id")))
				c.Abort()
				if !c.Writer.Written() {
					failure(c, 500, "INTERNAL_ERROR", "服务内部错误")
				}
			}
		}()
		c.Next()
	})
	r.GET("/api/v1/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"data": gin.H{"status": "ok", "service": "api"}, "request_id": c.GetString("request_id")})
	})
	r.GET("/api/v1/health/ready", func(c *gin.Context) {
		// Until dependencies and business services are wired, do not claim readiness.
		failure(c, 503, "NOT_READY", "工程骨架可运行，业务依赖尚未接入")
	})
	r.NoRoute(func(c *gin.Context) { failure(c, 404, "NOT_FOUND", "接口不存在") })
	r.NoMethod(func(c *gin.Context) { failure(c, 405, "METHOD_NOT_ALLOWED", "不支持的请求方法") })
	return r
}

func failure(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}, "request_id": c.GetString("request_id")})
}
