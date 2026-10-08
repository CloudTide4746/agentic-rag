package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"teamflow/internal/cache"
	"teamflow/pkg/apperr"
	"teamflow/pkg/jwt"
	"teamflow/pkg/response"

	"github.com/gin-gonic/gin"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头中获取token
		token := c.GetHeader("Authorization")
		if token == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		blacklisted, err := cache.IsBlacklisted(token)
		if err != nil {
			c.Error(apperr.ErrUnauthorized)
			return
		}
		if blacklisted {
			c.Error(apperr.New(1001, "token is blacklisted", http.StatusUnauthorized))
			return
		}
		// 验证token
		parts := strings.SplitN(token, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		// 验证token
		claims, err := jwt.ParseToken(parts[1])

		rawToken := parts[1]
		userCache := &cache.UserCache{}
		cachedToken, err := userCache.GetUserToken(strconv.FormatUint(uint64(claims.UserID), 10))
		if err != nil || cachedToken != rawToken {
			c.Error(apperr.ErrUnauthorized)
			c.Abort()
			return
		}
		if err != nil {
			c.Error(apperr.ErrUnauthorized)
			return
		}
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func GetCurrentUsername(c *gin.Context) (string, bool) {
	username, ok := c.Get("username")
	if !ok {
		return "", false
	}
	return username.(string), true
}

// GetCurrentUserID 获取当前登录用户的ID
func GetCurrentUserID(c *gin.Context) (uint, bool) {
	userID, ok := c.Get("userID")
	if !ok {
		return 0, false
	}
	return userID.(uint), true
}
