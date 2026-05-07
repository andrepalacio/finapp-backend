package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/handlers"
	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	_ "github.com/andrespalacio/finapp-backend/api/swagger"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

// @title           FinApp API
// @version         1.0
// @description     API REST de finanzas personales
// @host            localhost:8080
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
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
	bcryptCost := parseInt(logger, "BCRYPT_COST", 10)
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
	userRepo        := repositories.NewUserRepository(pool)
	workspaceRepo   := repositories.NewWorkspaceRepository(pool)
	categoryRepo    := repositories.NewCategoryRepository(pool)
	transactionRepo := repositories.NewTransactionRepository(pool)
	budgetRepo      := repositories.NewBudgetRepository(pool)

	// Services
	authSvc        := services.NewAuthService(userRepo, redisClient, jwtManager, bcryptCost)
	userSvc        := services.NewUserService(userRepo)
	workspaceSvc   := services.NewWorkspaceService(workspaceRepo)
	categorySvc    := services.NewCategoryService(categoryRepo)
	transactionSvc := services.NewTransactionService(transactionRepo)
	budgetSvc      := services.NewBudgetService(budgetRepo)

	// Handlers
	authHandler        := handlers.NewAuthHandler(authSvc)
	userHandler        := handlers.NewUserHandler(userSvc)
	workspaceHandler   := handlers.NewWorkspaceHandler(workspaceSvc)
	categoryHandler    := handlers.NewCategoryHandler(categorySvc)
	transactionHandler := handlers.NewTransactionHandler(transactionSvc)
	budgetHandler      := handlers.NewBudgetHandler(budgetSvc)

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
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		rateLimiter := middleware.RateLimitMiddleware(redisClient, 10, time.Minute)

		auth := v1.Group("/auth")
		{
			auth.POST("/register", rateLimiter, authHandler.Register)
			auth.POST("/login", rateLimiter, authHandler.Login)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
		}

		user := v1.Group("/user", middleware.AuthMiddleware(jwtManager))
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
		}

		authRequired := middleware.AuthMiddleware(jwtManager)
		wsMW := middleware.WorkspaceMiddleware(workspaceRepo)

		// Workspaces
		ws := v1.Group("/workspaces", authRequired)
		{
			ws.POST("", workspaceHandler.Create)
			ws.GET("", workspaceHandler.List)
		}
		wsMember := v1.Group("/workspaces/:workspace_id", authRequired, wsMW)
		{
			wsMember.GET("", workspaceHandler.Get)
			wsMember.PUT("", workspaceHandler.Update)
			wsMember.DELETE("", workspaceHandler.Delete)

			// Categories
			wsMember.GET("/categories", categoryHandler.List)
			wsMember.POST("/categories", categoryHandler.Create)
			wsMember.PUT("/categories/:category_id", categoryHandler.Update)
			wsMember.DELETE("/categories/:category_id", categoryHandler.Delete)

			// Transactions
			wsMember.GET("/transactions", transactionHandler.List)
			wsMember.POST("/transactions", transactionHandler.Create)
			wsMember.POST("/transactions/transfer", transactionHandler.CreateTransfer)
			wsMember.GET("/transactions/summary", transactionHandler.DailySummary)
			wsMember.GET("/transactions/by-date/:date", transactionHandler.ListByDate)
			wsMember.GET("/transactions/:transaction_id", transactionHandler.Get)
			wsMember.PUT("/transactions/:transaction_id", transactionHandler.Update)
			wsMember.DELETE("/transactions/:transaction_id", transactionHandler.Delete)

			// Budgets
			wsMember.GET("/budgets", budgetHandler.List)
			wsMember.PUT("/budgets/:year/:month", budgetHandler.Upsert)
			wsMember.GET("/budgets/:year/:month", budgetHandler.Get)
			wsMember.DELETE("/budgets/:year/:month", budgetHandler.Delete)
			wsMember.PUT("/budgets/:year/:month/categories/:category_id", budgetHandler.UpsertCategory)
			wsMember.DELETE("/budgets/:year/:month/categories/:category_id", budgetHandler.DeleteCategory)
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

func parseInt(logger *zap.Logger, key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		logger.Fatal("invalid integer env var", zap.String("key", key), zap.Error(err))
	}
	return i
}

func parseDuration(logger *zap.Logger, key, fallback string) time.Duration {
	d, err := time.ParseDuration(getEnv(key, fallback))
	if err != nil {
		logger.Fatal("invalid duration", zap.String("key", key), zap.Error(err))
	}
	return d
}
