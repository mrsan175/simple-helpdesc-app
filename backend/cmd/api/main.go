package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/handler"
	"backend/internal/middleware"
	"backend/internal/repository"
	"backend/internal/service"
)

func main() {
	// Optional local .env; docker-compose injects env directly.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load error: %v", err)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	if cfg.SeedOnStart {
		if err := database.Seed(db); err != nil {
			log.Fatalf("seed error: %v", err)
		}
	}

	userRepo := repository.NewUserRepository(db)
	ticketRepo := repository.NewTicketRepository(db)
	commentRepo := repository.NewCommentRepository(db)

	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiry)
	ticketService := service.NewTicketService(ticketRepo, commentRepo, userRepo, cfg.SLAHours)
	userService := service.NewUserService(userRepo)

	authHandler := handler.NewAuthHandler(authService)
	ticketHandler := handler.NewTicketHandler(ticketService)
	userHandler := handler.NewUserHandler(userService)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

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
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
		api.GET("/me", middleware.Auth(cfg.JWTSecret), authHandler.Me)

		api.GET("/stats", middleware.Auth(cfg.JWTSecret), ticketHandler.Stats)
		api.GET("/tickets/stats", middleware.Auth(cfg.JWTSecret), ticketHandler.Stats)

		tickets := api.Group("/tickets", middleware.Auth(cfg.JWTSecret))
		{
			tickets.GET("", ticketHandler.List)
			tickets.POST("", ticketHandler.Create)
			tickets.GET("/:id", ticketHandler.Get)
			tickets.PATCH("/:id", ticketHandler.Update)
			tickets.POST("/:id/comments", ticketHandler.AddComment)
		}

		users := api.Group("/users", middleware.Auth(cfg.JWTSecret))
		{
			users.GET("", userHandler.List)
			users.PATCH("/:id/role", userHandler.UpdateRole)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
