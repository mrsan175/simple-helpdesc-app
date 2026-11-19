package handler

import (
	"github.com/gin-gonic/gin"

	"backend/internal/middleware"
)

type RouterConfig struct {
	JWTSecret     string
	AuthHandler   *AuthHandler
	TicketHandler *TicketHandler
	UserHandler   *UserHandler
}

func SetupRouter(cfg RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		api.POST("/auth/register", cfg.AuthHandler.Register)
		api.POST("/auth/login", cfg.AuthHandler.Login)
		api.GET("/me", middleware.Auth(cfg.JWTSecret), cfg.AuthHandler.Me)

		api.GET("/stats", middleware.Auth(cfg.JWTSecret), cfg.TicketHandler.Stats)
		api.GET("/tickets/stats", middleware.Auth(cfg.JWTSecret), cfg.TicketHandler.Stats)

		tickets := api.Group("/tickets", middleware.Auth(cfg.JWTSecret))
		{
			tickets.GET("", cfg.TicketHandler.List)
			tickets.POST("", cfg.TicketHandler.Create)
			tickets.GET("/:id", cfg.TicketHandler.Get)
			tickets.PATCH("/:id", cfg.TicketHandler.Update)
			tickets.POST("/:id/comments", cfg.TicketHandler.AddComment)
			tickets.POST("/:id/attachments", cfg.TicketHandler.UploadAttachment)
		}

		comments := api.Group("/comments", middleware.Auth(cfg.JWTSecret))
		{
			comments.PATCH("/:id", cfg.TicketHandler.UpdateComment)
			comments.DELETE("/:id", cfg.TicketHandler.DeleteComment)
			comments.POST("/:id/attachments", cfg.TicketHandler.UploadCommentAttachment)
		}

		attachments := api.Group("/attachments", middleware.Auth(cfg.JWTSecret))
		{
			attachments.GET("/:id/download", cfg.TicketHandler.DownloadAttachment)
			attachments.GET("/:id/preview", cfg.TicketHandler.PreviewAttachment)
			attachments.PATCH("/:id", cfg.TicketHandler.ReplaceAttachment)
			attachments.DELETE("/:id", cfg.TicketHandler.DeleteAttachment)
		}

		api.GET("/events", middleware.Auth(cfg.JWTSecret), cfg.TicketHandler.StreamEvents)
		api.GET("/admin/stats", middleware.Auth(cfg.JWTSecret), cfg.TicketHandler.AdminStats)

		users := api.Group("/users", middleware.Auth(cfg.JWTSecret))
		{
			users.GET("", cfg.UserHandler.List)
			users.PATCH("/:id/role", cfg.UserHandler.UpdateRole)
		}
	}

	return r
}
