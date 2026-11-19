package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/handler"
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
	attachmentRepo := repository.NewAttachmentRepository(db)

	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTExpiry)
	ticketService := service.NewTicketService(ticketRepo, commentRepo, userRepo, attachmentRepo, cfg.SLAHours)
	userService := service.NewUserService(userRepo)

	authHandler := handler.NewAuthHandler(authService)
	ticketHandler := handler.NewTicketHandler(ticketService, cfg.UploadDir)
	userHandler := handler.NewUserHandler(userService)

	r := handler.SetupRouter(handler.RouterConfig{
		JWTSecret:     cfg.JWTSecret,
		AuthHandler:   authHandler,
		TicketHandler: ticketHandler,
		UserHandler:   userHandler,
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = cfg.Port
	}
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
