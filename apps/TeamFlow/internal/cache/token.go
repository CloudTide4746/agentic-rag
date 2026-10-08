package cache

import (
	"context"
	"crypto/sha256"
	"fmt"
	"teamflow/internal/database"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	blacklistPrefix = "blacklist:token:"
	userTokenPrefix = "user:token:"
)

// AddTokenToBlacklist 添加token到黑名单 ttl 为10分钟
func AddTokenToBlacklist(token string, ttl time.Duration) error {
	key := blacklistKey(token)
	return database.RDB.Set(context.Background(), key, token, ttl).Err()
}

// IsBlacklisted 检查token是否在黑名单中
func IsBlacklisted(token string) (bool, error) {
	key := blacklistKey(token)
	err := database.RDB.Get(context.Background(), key).Err()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// blacklistKey 生成黑名单key
func blacklistKey(token string) string {
	h := sha256.Sum256([]byte(token))
	return blacklistPrefix + fmt.Sprintf("%x", h[:8])
}
