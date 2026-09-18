package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/livepolling/backend/config"
	"github.com/livepolling/backend/handlers"
	"github.com/livepolling/backend/middleware"
	"github.com/livepolling/backend/services"
)

func main() {
	cfg := config.Load()

	// ── MongoDB ─────────────────────────────────────────────
	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer mongoCancel()

	mongoClient, err := mongo.Connect(mongoCtx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}
	defer mongoClient.Disconnect(context.Background())

	if err := mongoClient.Ping(mongoCtx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}
	log.Println("✓ Connected to MongoDB")

	// ── Validate JWT secret ────────────────────────────────
	if cfg.JWTSecret == "change-me-in-production-please" {
		log.Println("⚠ WARNING: Using default JWT_SECRET. Set a secure value in production!")
	}

	db := mongoClient.Database("livepolling")

	// ── Create MongoDB indexes ─────────────────────────────
	createIndexes(db)

	// ── Redis ───────────────────────────────────────────────
	redisSvc, err := services.NewRedisService(cfg.RedisURL)
	if err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}
	defer redisSvc.Close()
	log.Println("✓ Connected to Redis")

	// ── Background sync (Redis → MongoDB) ───────────────────
	syncSvc := services.NewSyncService(db, redisSvc)
	syncSvc.Start()
	defer syncSvc.Stop()

	// ── Handlers ────────────────────────────────────────────
	authHandler := handlers.NewAuthHandler(db, cfg)
	pollHandler := handlers.NewPollHandler(db, redisSvc)
	voteHandler := handlers.NewVoteHandler(db, redisSvc)
	liveHandler := handlers.NewLiveHandler(db, redisSvc)

	// ── Router ──────────────────────────────────────────────
	router := gin.Default()

	// CORS middleware — validate origin against allowlist
	allowedOrigins := buildAllowedOrigins(cfg.FrontendURL)
	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if isAllowedOrigin(origin, allowedOrigins) {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.Header("Access-Control-Allow-Credentials", "true")
		}
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	api := router.Group("/api")

	// Auth routes (public)
	api.POST("/auth/signup", authHandler.Signup)
	api.POST("/auth/login", authHandler.Login)
	api.GET("/auth/me", middleware.AuthMiddleware(cfg.JWTSecret), authHandler.Me)

	// Poll routes — public
	api.GET("/polls/:id", pollHandler.Get)
	api.GET("/polls/:id/live", liveHandler.Stream)
	api.POST("/polls/:id/vote",
		middleware.RateLimitMiddleware(redisSvc.Client(), 10, time.Minute),
		voteHandler.Vote,
	)

	// Poll routes — authenticated
	authMW := middleware.AuthMiddleware(cfg.JWTSecret)
	api.POST("/polls", authMW, pollHandler.Create)
	api.GET("/polls", authMW, pollHandler.List)
	api.PATCH("/polls/:id/close", authMW, pollHandler.Close)
	api.DELETE("/polls/:id", authMW, pollHandler.Delete)

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().UTC()})
	})

	// ── Start server ────────────────────────────────────────
	port := cfg.Port
	log.Printf("🚀 Server starting on :%s", port)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed:", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited cleanly")
}

// createIndexes ensures required MongoDB indexes exist.
func createIndexes(db *mongo.Database) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	users := db.Collection("users")
	polls := db.Collection("polls")

	userIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "username", Value: 1}}, Options: options.Index().SetUnique(true)},
	}
	if _, err := users.Indexes().CreateMany(ctx, userIndexes); err != nil {
		log.Printf("Warning: failed to create user indexes: %v", err)
	}

	pollIndexes := []mongo.IndexModel{
		{Keys: bson.D{{Key: "share_code", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "creator_id", Value: 1}}},
		{Keys: bson.D{{Key: "is_active", Value: 1}}},
	}
	if _, err := polls.Indexes().CreateMany(ctx, pollIndexes); err != nil {
		log.Printf("Warning: failed to create poll indexes: %v", err)
	}

	log.Println("✓ MongoDB indexes ensured")
}

// buildAllowedOrigins creates a set of allowed CORS origins.
func buildAllowedOrigins(frontendURL string) map[string]bool {
	origins := map[string]bool{
		"http://localhost:5173": true,
		"http://localhost:4173": true, // vite preview
	}
	// Add configured frontend URL and variants
	if frontendURL != "" {
		origins[strings.TrimRight(frontendURL, "/")] = true
	}
	return origins
}

// isAllowedOrigin checks if the given origin is in the allowlist.
func isAllowedOrigin(origin string, allowed map[string]bool) bool {
	if origin == "" {
		return false
	}
	return allowed[strings.TrimRight(origin, "/")]
}
