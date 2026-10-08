package router

import (
	"teamflow/internal/controller"
	"teamflow/internal/middleware"
	"teamflow/internal/model"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func SetupRouter(
	userController *controller.UserController,
	teamController *controller.TeamController,
	projectController *controller.ProjectController,
	taskController *controller.TaskController,
	commentController *controller.CommentController,
	attachmentController *controller.AttachmentController,
	logger *zap.Logger,
	onlineController *controller.OnlineController,
) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.ErrorHandler(logger))
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/swagger.json")))
	r.StaticFile("/", "./web/index.html")

	api := r.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", userController.Login)
			auth.POST("/register", userController.Register)
		}

		protected := api.Group("")
		protected.Use(middleware.JWTAuth())

		users := protected.Group("/users")
		{
			users.GET("/me", userController.GetProfile)
			users.PUT("/me", userController.UpdateProfile)
			users.PUT("/me/password", userController.ChangePassword)
			users.POST("/me/avatar", userController.UploadAvatar)
		}

		online := protected.Group("/online")
		{
			online.GET("/users", onlineController.GetOnlineUsers)
			online.POST("/users/:userId", onlineController.AddUserToOnline)
			online.DELETE("/users/:userId", onlineController.RemoveUserFromOnline)
		}

		team := protected.Group("/teams")
		{
			team.POST("", teamController.CreateTeam)
			team.GET("", teamController.GetTeamList)
			team.GET("/:id", teamController.GetTeamDetail)
			team.PUT("/:id", middleware.RequireTeamRole(model.RoleOwner), teamController.UpdateTeam)
			team.DELETE("/:id", middleware.RequireTeamRole(model.RoleOwner), teamController.DeleteTeam)
			team.POST("/:id/members", middleware.RequireTeamRole(model.RoleAdmin), teamController.AddMember)
			team.DELETE("/:id/members/me", teamController.LeaveTeam)
			team.DELETE("/:id/members/:userId", middleware.RequireTeamRole(model.RoleAdmin), teamController.RemoveMember)
			team.POST("/:id/projects", projectController.CreateProject)
			team.GET("/:id/projects", projectController.GetProjectList)
		}

		project := protected.Group("/projects")
		{
			project.GET("/:id", projectController.GetProjectDetail)
			project.PUT("/:id", middleware.RequireProjectRole(model.RoleOwner), projectController.UpdateProject)
			project.DELETE("/:id", middleware.RequireProjectRole(model.RoleOwner), projectController.DeleteProject)
			project.POST("/:id/members", middleware.RequireProjectRole(model.RoleAdmin), projectController.AddMember)
			project.DELETE("/:id/members/me", projectController.LeaveProject)
			project.DELETE("/:id/members/:userId", middleware.RequireProjectRole(model.RoleAdmin), projectController.RemoveMember)
			project.GET("/:id/stats", projectController.GetProjectStats)
		}

		tasks := protected.Group("/tasks")
		{
			tasks.POST("", taskController.CreateTask)
			tasks.GET("", taskController.GetTaskList)
			tasks.GET("/:id", taskController.GetTaskDetail)
			tasks.PUT("/:id", taskController.UpdateTask)
			tasks.DELETE("/:id", taskController.DeleteTask)
			tasks.PUT("/:id/status", taskController.ChangeTaskStatus)
			tasks.PATCH("/:id/assignee", taskController.AssignTask)
			tasks.GET("/:id/attachments", attachmentController.GetAttachments)
			tasks.POST("/:id/attachments", attachmentController.UploadAttachment)
			tasks.POST("/:id/comments", commentController.CreateCommentController)
			tasks.GET("/:id/comments", commentController.GetCommentsController)
			tasks.DELETE("/:id/comments/:commentId", commentController.DeleteCommentController)
		}

		attachments := protected.Group("/attachments")
		{
			attachments.DELETE("/:id", attachmentController.DeleteAttachmentHandler)
			attachments.GET("/:id/download", attachmentController.DownloadAttachmentHandler)
		}
	}
	return r
}
