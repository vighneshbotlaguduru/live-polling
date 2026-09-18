package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// Config holds all application configuration values.
type Config struct {
	MongoURI    string
	RedisURL    string
	JWTSecret   string
	Port        string
	FrontendURL string
}

// Load reads configuration from environment variables, with .env fallback.
func Load() *Config {
	// Try loading from .env in various common locations
	_ = godotenv.Load(".env", "backend/.env", "../backend/.env")
	if exe, err := os.Executable(); err == nil {
		_ = godotenv.Load(filepath.Join(filepath.Dir(exe), ".env"))
	}

	mongoURI := getEnv("MONGO_URI", "")
	if mongoURI == "" {
		mongoURI = getEnv("MONGODB_URI", "")
	}
	if mongoURI == "" {
		mongoURI = getEnv("MONGODB_URL", "mongodb://localhost:27017")
	}

	redisURL := getEnv("REDIS_URL", "")
	if redisURL == "" {
		redisURL = getEnv("REDIS_URI", "redis://localhost:6379")
	}

	cfg := &Config{
		MongoURI:    mongoURI,
		RedisURL:    redisURL,
		JWTSecret:   getEnv("JWT_SECRET", "change-me-in-production-please"),
		Port:        getEnv("PORT", "8080"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),
	}

	if cfg.MongoURI == "mongodb://localhost:27017" {
		log.Println("⚠️  WARNING: MONGO_URI is not set! Falling back to localhost:27017. If you are on Render, add MONGO_URI in Environment Variables!")
	} else {
		log.Println("ℹ️  MONGO_URI is configured")
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
