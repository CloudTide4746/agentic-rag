package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	onlineUsersKey = "teamflow:online:users"   // 全局在线用户集合
	userOnlineKey  = "teamflow:online:user:%d" // 单用户在线 key（用于超时）
	onlineTTL      = 90 * time.Second          // 90秒无心跳则离线
)

type OnlineService struct {
	rdb *redis.Client
}

func (m OnlineService) RemoveUserFromOnline(param any, userID uint) any {
	panic("unimplemented")
}

func NewOnlineService(rdb *redis.Client) *OnlineService {
	return &OnlineService{rdb: rdb}
}

// SetOnline 标记用户上线
func (m *OnlineService) SetOnline(ctx context.Context, userID uint) error {
	pipe := m.rdb.Pipeline()
	// 加入全局在线集合
	pipe.SAdd(ctx, onlineUsersKey, userID)
	// 设置用户独立 key，带过期时间（心跳刷新）
	pipe.Set(ctx, fmt.Sprintf(userOnlineKey, userID), "1", onlineTTL)
	_, err := pipe.Exec(ctx)
	return err
}

// SetOffline 标记用户下线
func (m *OnlineService) SetOffline(ctx context.Context, userID uint) error {
	pipe := m.rdb.Pipeline()
	pipe.SRem(ctx, onlineUsersKey, userID)
	pipe.Del(ctx, fmt.Sprintf(userOnlineKey, userID))
	_, err := pipe.Exec(ctx)
	return err
}

// Heartbeat 刷新用户在线状态（每30秒调用一次）
func (m *OnlineService) Heartbeat(ctx context.Context, userID uint) error {
	return m.rdb.Expire(ctx, fmt.Sprintf(userOnlineKey, userID), onlineTTL).Err()
}

// IsOnline 查询用户是否在线
func (m *OnlineService) IsOnline(ctx context.Context, userID uint) (bool, error) {
	exists, err := m.rdb.Exists(ctx, fmt.Sprintf(userOnlineKey, userID)).Result()
	return exists > 0, err
}

// GetOnlineUsers 获取所有在线用户 ID
func (m *OnlineService) GetOnlineUsers(ctx context.Context) ([]uint, error) {
	members, err := m.rdb.SMembers(ctx, onlineUsersKey).Result()
	if err != nil {
		return nil, err
	}
	var ids []uint
	for _, m := range members {
		id, _ := strconv.ParseUint(m, 10, 64)
		ids = append(ids, uint(id))
	}
	return ids, nil
}
