package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	JWTExpiry      time.Duration
	SLAHours       int
	AllowedOrigins string
	SeedOnStart    bool
	UploadDir      string
}

func Load() (*Config, error) {
	port := getEnv("PORT", "8080")
	dbURL := getEnv("DATABASE_URL", "")
	jwtSecret := getEnv("JWT_SECRET", "")

	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	expiryHours, err := strconv.Atoi(getEnv("JWT_EXPIRY_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRY_HOURS: %w", err)
	}
	slaHours, err := strconv.Atoi(getEnv("SLA_HOURS", "2"))
	if err != nil {
		return nil, fmt.Errorf("invalid SLA_HOURS: %w", err)
	}

	return &Config{
		Port:           port,
		DatabaseURL:    dbURL,
		JWTSecret:      jwtSecret,
		JWTExpiry:      time.Duration(expiryHours) * time.Hour,
		SLAHours:       slaHours,
		AllowedOrigins: getEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		SeedOnStart:    getEnv("SEED_ON_START", "true") == "true",
		UploadDir:      getEnv("UPLOAD_DIR", "./uploads"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
