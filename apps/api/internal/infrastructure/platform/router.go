package platform

import (
	"context"
	"fmt"
	"net/http"
	"time"

	authhandler "teamflow/internal/auth/handler"
	authrepo "teamflow/internal/auth/repository"
	authservice "teamflow/internal/auth/service"
	chathandler "teamflow/internal/chat/handler"
	chatrepo "teamflow/internal/chat/repository"
	chatservice "teamflow/internal/chat/service"
	"teamflow/internal/infrastructure/config"
	"teamflow/internal/infrastructure/httpx"
	"teamflow/internal/infrastructure/middleware"
	security "teamflow/internal/infrastructure/security"
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

// NewRouter 初始化基础设施并注入各模块依赖，注册业务路由及全局中间件。
// NewRouter is the composition root: it builds infrastructure, wires use cases
// and lets each module register its own routes.
func NewRouter(ctx context.Context, logger *zap.Logger, cfg config.Config) (*gin.Engine, *RouterResources, error) {
	//初始化数据库连接
	db, err := openDatabase(ctx, cfg.Database.DSN)
	if err != nil {
		return nil, nil, err
	}
	//初始化数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get database pool: %w", err)
	}
	//初始化Redis连接
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.Redis.Addr, Password: cfg.Redis.Password, DB: cfg.Redis.DB, DialTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second, ContextTimeoutEnabled: true})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = redisClient.Close()
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("connect redis: %w", err)
	}
	//初始化JWT签名器
	tokens, err := security.NewJWTSigner(cfg.JWT.Secret)
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
	var chats chathandler.ChatHandler = chatservice.NewChatService(chatrepo.NewSessionRepository(db), chatrepo.NewAgentClient(cfg.Agent.BaseURL, cfg.Agent.Token))

	//初始化路由
	router := gin.New()
	//错误处理
	router.HandleMethodNotAllowed = true //对不支持的HTTP方法返回405
	router.NoRoute(func(c *gin.Context) {
		httpx.Fail(c, http.StatusUnauthorized, "route not found", "route not found")
	}) //对不存在的路由返回404

	router.Use(middleware.Recovery(logger), middleware.RequestID(logger), middleware.BodyLimit(1<<20))
	router.GET("/healthz", func(c *gin.Context) { httpx.Success(c, http.StatusOK, gin.H{"status": "ok"}) })
	router.GET("/", func(c *gin.Context) { httpx.Success(c, http.StatusOK, gin.H{"status": "ok"}) })
	api := router.Group("/api/v1")

	//注册路由
	auth := authhandler.New(authApp)
	auth.Register(api)
	protected := api.Group("")
	protected.Use(middleware.Authenticate(authApp))

	teamhandler.New(teamService).Register(protected)
	projecthandler.New(projectService).Register(protected)
	taskhandler.New(taskService).Register(protected)
	chathandler.New(chats).Register(protected)

	return router, &RouterResources{SQLClose: sqlDB.Close, Redis: redisClient}, nil
}
