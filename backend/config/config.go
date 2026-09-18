package config

import (
	"os"

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
	// Try loading from .env, backend/.env, or ../.env
	_ = godotenv.Load(".env", "backend/.env", "../backend/.env")

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

	return cfg
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return fallback
}
