package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	"teamflow/internal/faults"
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

	var authApp authhandler.AuthHandler = authservice.NewAuthService(authrepo.NewUserRepository(db), authrepo.NewRedisSessions(redisClient), tokens)
	teamService := teamservice.NewTeamService(teamrepo.NewTeamRepository(db))
	var teams teamhandler.TeamHandler = teamService
	projectService := projectservice.NewProjectService(projectrepo.NewProjectRepository(db), teamService)
	var projects projecthandler.ProjectHandler = projectService
	taskService := taskservice.NewTaskService(taskrepo.NewTaskRepository(db), projectService)
	var tasks taskhandler.TaskHandler = taskService
	// The chat module owns the platform session mapping only; Agent-owned
	// conversation data stays behind AgentClient (ADR-0003).
	var chats chathandler.ChatHandler = chatservice.NewChatService(chatrepo.NewSessionStore(db), chatrepo.NewAgentClient())

	router := gin.New()
	router.Use(recoveryMiddleware(logger), requestIDMiddleware(logger), jsonBodyLimitMiddleware(1<<20))
	router.GET("/healthz", func(c *gin.Context) { success(c, http.StatusOK, gin.H{"status": "ok"}) })
	api := router.Group("/api/v1")
	api.POST("/auth/register", func(c *gin.Context) {
		var req authhandler.RegisterReq
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		user, err := authApp.Register(c.Request.Context(), &req)
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusCreated, user)
	})
	api.POST("/auth/login", func(c *gin.Context) {
		var req authhandler.LoginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		resp, err := authApp.Login(c.Request.Context(), &req)
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected := api.Group("")
	protected.Use(authMiddleware(authApp))

	protected.POST("/teams", func(c *gin.Context) {
		var body teamhandler.CreateTeamReq
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		body.ActorID = actorID(c)
		resp, err := teams.CreateTeam(c.Request.Context(), &body)
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusCreated, resp)
	})
	protected.GET("/teams", func(c *gin.Context) {
		page, size, ok := parsePage(c)
		if !ok {
			return
		}
		resp, err := teams.ListTeams(c.Request.Context(), &teamhandler.ListTeamsReq{ActorID: actorID(c), Page: page, PageSize: size})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.GET("/teams/:id", func(c *gin.Context) {
		id, ok := pathID(c, "id")
		if !ok {
			return
		}
		resp, err := teams.GetTeam(c.Request.Context(), &teamhandler.GetTeamReq{ActorID: actorID(c), TeamID: id})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.PUT("/teams/:id", func(c *gin.Context) {
		id, ok := pathID(c, "id")
		if !ok {
			return
		}
		var body teamhandler.UpdateTeamReq
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		body.ActorID, body.TeamID = actorID(c), id
		resp, err := teams.UpdateTeam(c.Request.Context(), &body)
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.DELETE("/teams/:id", func(c *gin.Context) {
		id, ok := pathID(c, "id")
		if !ok {
			return
		}
		resp, err := teams.DeleteTeam(c.Request.Context(), &teamhandler.DeleteTeamReq{ActorID: actorID(c), TeamID: id})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})

	protected.POST("/teams/:id/projects", func(c *gin.Context) {
		teamID, ok := pathID(c, "id")
		if !ok {
			return
		}
		var body projecthandler.CreateProjectReq
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		body.ActorID, body.TeamID = actorID(c), teamID
		resp, err := projects.CreateProject(c.Request.Context(), &body)
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusCreated, resp)
	})
	protected.GET("/teams/:id/projects", func(c *gin.Context) {
		teamID, ok := pathID(c, "id")
		if !ok {
			return
		}
		page, size, ok := parsePage(c)
		if !ok {
			return
		}
		resp, err := projects.ListProjects(c.Request.Context(), &projecthandler.ListProjectsReq{ActorID: actorID(c), TeamID: teamID, Page: page, PageSize: size})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.GET("/projects/:id", func(c *gin.Context) {
		id, ok := pathID(c, "id")
		if !ok {
			return
		}
		resp, err := projects.GetProject(c.Request.Context(), &projecthandler.GetProjectReq{ActorID: actorID(c), ProjectID: id})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.PUT("/projects/:id", func(c *gin.Context) {
		id, ok := pathID(c, "id")
		if !ok {
			return
		}
		var body projecthandler.UpdateProjectReq
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		body.ActorID, body.ProjectID = actorID(c), id
		resp, err := projects.UpdateProject(c.Request.Context(), &body)
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.DELETE("/projects/:id", func(c *gin.Context) {
		id, ok := pathID(c, "id")
		if !ok {
			return
		}
		resp, err := projects.DeleteProject(c.Request.Context(), &projecthandler.DeleteProjectReq{ActorID: actorID(c), ProjectID: id})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})

	protected.POST("/tasks", func(c *gin.Context) {
		var body taskhandler.CreateTaskReq
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		body.ActorID = actorID(c)
		resp, err := tasks.CreateTask(c.Request.Context(), &body)
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusCreated, resp)
	})
	protected.GET("/tasks", func(c *gin.Context) {
		page, size, ok := parsePage(c)
		if !ok {
			return
		}
		resp, err := tasks.ListTasks(c.Request.Context(), &taskhandler.ListTasksReq{ActorID: actorID(c), ProjectID: c.Query("project_id"), Page: page, PageSize: size})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.GET("/tasks/:id", func(c *gin.Context) {
		id, ok := pathID(c, "id")
		if !ok {
			return
		}
		resp, err := tasks.GetTask(c.Request.Context(), &taskhandler.GetTaskReq{ActorID: actorID(c), TaskID: id})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.PUT("/tasks/:id", func(c *gin.Context) {
		id, ok := pathID(c, "id")
		if !ok {
			return
		}
		var body taskhandler.UpdateTaskReq
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		body.ActorID, body.TaskID = actorID(c), id
		resp, err := tasks.UpdateTask(c.Request.Context(), &body)
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.DELETE("/tasks/:id", func(c *gin.Context) {
		id, ok := pathID(c, "id")
		if !ok {
			return
		}
		resp, err := tasks.DeleteTask(c.Request.Context(), &taskhandler.DeleteTaskReq{ActorID: actorID(c), TaskID: id})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.POST("/chat/sessions", func(c *gin.Context) {
		var body chathandler.CreateChatReq
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		body.ActorID = actorID(c)
		resp, err := chats.CreateChat(c.Request.Context(), &body)
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusCreated, resp)
	})
	protected.GET("/chat/sessions", func(c *gin.Context) {
		page, size, ok := parsePage(c)
		if !ok {
			return
		}
		resp, err := chats.ListSessions(c.Request.Context(), &chathandler.ListSessionsReq{ActorID: actorID(c), Page: page, PageSize: size})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.GET("/chat/sessions/:id", func(c *gin.Context) {
		// The public id is an opaque platform string, not a numeric id.
		resp, err := chats.GetChat(c.Request.Context(), &chathandler.GetChatReq{ActorID: actorID(c), SessionID: c.Param("id")})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp)
	})
	protected.GET("/chat/sessions/:id/messages", func(c *gin.Context) {
		resp, err := chats.GetChat(c.Request.Context(), &chathandler.GetChatReq{ActorID: actorID(c), SessionID: c.Param("id")})
		if err != nil {
			writeError(c, err)
			return
		}
		success(c, http.StatusOK, resp.Messages)
	})
	protected.POST("/chat/sessions/:id/runs", func(c *gin.Context) {
		var body chathandler.AskReq
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
			return
		}
		body.ActorID, body.SessionID = actorID(c), c.Param("id")
		stream, err := chats.Ask(c.Request.Context(), &body)
		if err != nil {
			writeError(c, err)
			return
		}
		// Forward the Agent's SSE frames verbatim. The browser never reaches
		// the Agent directly (ADR-0003 "消息和问答流程").
		defer stream.Body.Close()
		c.Header("Content-Type", stream.ContentType)
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Status(http.StatusOK)
		c.Stream(func(w io.Writer) bool {
			buf := make([]byte, 4096)
			written, readErr := stream.Body.Read(buf)
			if written > 0 {
				if _, err := w.Write(buf[:written]); err != nil {
					return false
				}
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
			if readErr != nil {
				return false
			}
			return true
		})
	})
	return router, &RouterResources{SQLClose: sqlDB.Close, Redis: redisClient}, nil
}

func authMiddleware(app authhandler.AuthHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			fail(c, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录")
			return
		}
		identity, err := app.Authenticate(c.Request.Context(), parts[1])
		if err != nil {
			if errors.Is(err, faults.ErrUnauthenticated) {
				fail(c, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录")
			} else {
				requestLogger, _ := c.Get("logger")
				if requestLogger, ok := requestLogger.(*zap.Logger); ok {
					requestLogger.Error("authentication backend unavailable", zap.Any("request_id", c.GetString(requestIDKey)), zap.String("error_type", fmt.Sprintf("%T", err)))
				}
				fail(c, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "认证服务暂不可用")
			}
			return
		}
		c.Set("actor_id", identity.UserID)
		c.Next()
	}
}
func actorID(c *gin.Context) uint { id, _ := c.Get("actor_id"); value, _ := id.(uint); return value }
func pathID(c *gin.Context, name string) (uint, bool) {
	raw := c.Param(name)
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != raw {
		fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "无效的资源 ID")
		return 0, false
	}
	return uint(id), true
}
func parsePage(c *gin.Context) (int, int, bool) {
	page, size := 1, 20
	if raw := c.Query("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100000 {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "page 必须在 1 到 100000 之间")
			return 0, 0, false
		}
		page = parsed
	}
	if raw := c.Query("page_size"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "page_size 必须在 1 到 100 之间")
			return 0, 0, false
		}
		size = parsed
	}
	return page, size, true
}
