package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/handlers"
	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using environment variables")
	}

	logger := newLogger()
	defer logger.Sync() //nolint:errcheck

	// JWT
	secret := os.Getenv("JWT_SECRET")
	if len(secret) < 32 {
		logger.Fatal("JWT_SECRET must be at least 32 characters")
	}
	accessExpiry := parseDuration(logger, "JWT_ACCESS_EXPIRY", "15m")
	refreshExpiry := parseDuration(logger, "JWT_REFRESH_EXPIRY", "168h")
	jwtManager := pkgauth.NewJWTManager(secret, accessExpiry, refreshExpiry)

	// Database
	ctx := context.Background()
	pool, err := repositories.NewPostgresPool(ctx, mustEnv("DATABASE_URL"))
	if err != nil {
		logger.Fatal("failed to connect to postgres", zap.Error(err))
	}
	defer pool.Close()

	// Redis
	redisClient, err := repositories.NewRedisClient(mustEnv("REDIS_URL"))
	if err != nil {
		logger.Fatal("failed to connect to redis", zap.Error(err))
	}
	defer redisClient.Close()

	// Repositories
	userRepo := repositories.NewUserRepository(pool)

	// Services
	authSvc := services.NewAuthService(userRepo, redisClient, jwtManager)

	// Handlers
	authHandler := handlers.NewAuthHandler(authSvc)

	// Router
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(middleware.LoggerMiddleware(logger))
	r.Use(middleware.CORSMiddleware())
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}
	}

	port := getEnv("PORT", "8080")
	if err := r.Run(":" + port); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}

func newLogger() *zap.Logger {
	if os.Getenv("ENV") == "development" {
		l, _ := zap.NewDevelopment()
		return l
	}
	l, _ := zap.NewProduction()
	return l
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(logger *zap.Logger, key, fallback string) time.Duration {
	d, err := time.ParseDuration(getEnv(key, fallback))
	if err != nil {
		logger.Fatal("invalid duration", zap.String("key", key), zap.Error(err))
	}
	return d
}
