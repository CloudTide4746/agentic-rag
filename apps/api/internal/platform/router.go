package platform

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	authhandler "teamflow/internal/auth/handler"
	authrepo "teamflow/internal/auth/repository"
	authservice "teamflow/internal/auth/service"
	chathandler "teamflow/internal/chat/handler"
	chatrepo "teamflow/internal/chat/repository"
	chatservice "teamflow/internal/chat/service"
	"teamflow/internal/httpx"
	projecthandler "teamflow/internal/project/handler"
	projectrepo "teamflow/internal/project/repository"
	projectservice "teamflow/internal/project/service"
	taskhandler "teamflow/internal/task/handler"
	taskrepo "teamflow/internal/task/repository"
	taskservice "teamflow/internal/task/service"
	teamhandler "teamflow/internal/team/handler"
	teamrepo "teamflow/internal/team/repository"
	teamservice "teamflow/internal/team/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type RouterResources struct {
	SQLClose func() error
	Redis    *redis.Client
}

// NewRouter is the composition root: it builds infrastructure, wires use cases
// and lets each module register its own routes.
func NewRouter(ctx context.Context, logger *zap.Logger) (*gin.Engine, *RouterResources, error) {
	db, err := openDatabase(ctx)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get database pool: %w", err)
	}
	redisAddr := strings.TrimSpace(os.Getenv("REDIS_ADDR"))
	if redisAddr == "" {
		_ = sqlDB.Close()
		return nil, nil, errors.New("REDIS_ADDR must be set")
	}
	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil && os.Getenv("REDIS_DB") != "" {
		_ = sqlDB.Close()
		return nil, nil, errors.New("REDIS_DB must be an integer")
	}
	redisClient := redis.NewClient(&redis.Options{Addr: redisAddr, Password: os.Getenv("REDIS_PASSWORD"), DB: redisDB, DialTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, ContextTimeoutEnabled: true})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("connect redis: %w", err)
	}
	tokens, err := authrepo.NewJWTSignerFromEnv()
	if err != nil {
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, nil, err
	}

	//初始化服务层, 并注册路由, 并初始化中间件
	var authApp authhandler.AuthHandler = authservice.NewAuthService(authrepo.NewUserRepository(db), authrepo.NewRedisSessions(redisClient), tokens)
	teamService := teamservice.NewTeamService(teamrepo.NewTeamRepository(db))
	projectService := projectservice.NewProjectService(projectrepo.NewProjectRepository(db), teamService)
	taskService := taskservice.NewTaskService(taskrepo.NewTaskRepository(db), projectService)
	// The chat module owns the platform session mapping only; Agent-owned
	// conversation data stays behind AgentClient (ADR-0003).
	var chats chathandler.ChatHandler = chatservice.NewChatService(chatrepo.NewSessionRepository(db), chatrepo.NewAgentClient())

	//初始化路由
	router := gin.New()
	router.Use(recoveryMiddleware(logger), requestIDMiddleware(logger), jsonBodyLimitMiddleware(1<<20))
	router.GET("/healthz", func(c *gin.Context) { httpx.Success(c, http.StatusOK, gin.H{"status": "ok"}) })
	api := router.Group("/api/v1")

	//注册路由
	auth := authhandler.New(authApp)
	auth.Register(api)
	protected := api.Group("")
	protected.Use(auth.Middleware())

	teamhandler.New(teamService).Register(protected)
	projecthandler.New(projectService).Register(protected)
	taskhandler.New(taskService).Register(protected)
	chathandler.New(chats).Register(protected)

	return router, &RouterResources{SQLClose: sqlDB.Close, Redis: redisClient}, nil
}
