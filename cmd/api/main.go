package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/andrespalacio/finapp-backend/db"
	"github.com/andrespalacio/finapp-backend/internal/handlers"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	_ "github.com/andrespalacio/finapp-backend/api/swagger"
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
	databaseURL := mustEnv("DATABASE_URL")
	if err := db.RunMigrations(databaseURL); err != nil {
		logger.Fatal("failed to run migrations", zap.Error(err))
	}

	ctx := context.Background()
	pool, err := repositories.NewPostgresPool(ctx, databaseURL)
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
	debtRepo        := repositories.NewDebtRepository(pool)
	savingsRepo     := repositories.NewSavingsRepository(pool)
	invitationRepo  := repositories.NewInvitationRepository(pool)

	// Services
	authSvc        := services.NewAuthService(userRepo, redisClient, jwtManager, bcryptCost)
	userSvc        := services.NewUserService(userRepo, bcryptCost)
	workspaceSvc   := services.NewWorkspaceService(workspaceRepo)
	categorySvc    := services.NewCategoryService(categoryRepo)
	transactionSvc := services.NewTransactionService(transactionRepo)
	budgetSvc      := services.NewBudgetService(budgetRepo)
	debtSvc        := services.NewDebtService(debtRepo)
	savingsSvc     := services.NewSavingsService(savingsRepo)
	invitationSvc  := services.NewInvitationService(invitationRepo, workspaceRepo, userRepo)

	// Handlers
	authHandler        := handlers.NewAuthHandler(authSvc)
	userHandler        := handlers.NewUserHandler(userSvc)
	workspaceHandler   := handlers.NewWorkspaceHandler(workspaceSvc)
	categoryHandler    := handlers.NewCategoryHandler(categorySvc)
	transactionHandler := handlers.NewTransactionHandler(transactionSvc)
	budgetHandler      := handlers.NewBudgetHandler(budgetSvc)
	debtHandler        := handlers.NewDebtHandler(debtSvc)
	savingsHandler     := handlers.NewSavingsHandler(savingsSvc)
	invitationHandler  := handlers.NewInvitationHandler(invitationSvc)
	importHandler      := handlers.NewImportHandler(transactionSvc, categorySvc)
	alertHandler       := handlers.NewAlertHandler(budgetSvc)

	// Router
	r := newRouter(logger, redisClient, jwtManager, workspaceRepo, handlerSet{
		auth:        authHandler,
		user:        userHandler,
		workspace:   workspaceHandler,
		category:    categoryHandler,
		transaction: transactionHandler,
		budget:      budgetHandler,
		debt:        debtHandler,
		savings:     savingsHandler,
		invitation:  invitationHandler,
		importH:     importHandler,
		alert:       alertHandler,
	})

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
