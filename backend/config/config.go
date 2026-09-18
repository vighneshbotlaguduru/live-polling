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

	cfg := &Config{
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		RedisURL:    getEnv("REDIS_URL", "redis://localhost:6379"),
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
