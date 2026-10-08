package database

import (
	"context"
	"fmt"
	"log"
	"teamflow/config"
	"time"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis(cfg config.RedisConfig) error {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return fmt.Errorf("ping Redis失败: %w", err)
	}
	RDB = rdb
	log.Println("Redis 连接初始化成功")
	return nil
}

//example:
// import(
// 	"context"
// 	"pkg/database"
// )
// func SetUserCache(userID uint,data string) error{
// 	ctx:=context.Background()
// 	return database.RDB.Set(ctx,fmt.Sprintf("user:%d",userID),data,time.Hour).Err()
// }
