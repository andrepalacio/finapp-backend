package main

import (
	"os"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/handlers"
	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

type handlerSet struct {
	auth        *handlers.AuthHandler
	user        *handlers.UserHandler
	workspace   *handlers.WorkspaceHandler
	category    *handlers.CategoryHandler
	transaction *handlers.TransactionHandler
	budget      *handlers.BudgetHandler
	debt        *handlers.DebtHandler
	savings     *handlers.SavingsHandler
	invitation  *handlers.InvitationHandler
	importH     *handlers.ImportHandler
	alert       *handlers.AlertHandler
}

func newRouter(logger *zap.Logger, redisClient *redis.Client, jwtManager *pkgauth.JWTManager, workspaceRepo *repositories.WorkspaceRepository, h handlerSet) *gin.Engine {
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
			auth.POST("/register", rateLimiter, h.auth.Register)
			auth.POST("/login", rateLimiter, h.auth.Login)
			auth.POST("/refresh", h.auth.Refresh)
			auth.POST("/logout", h.auth.Logout)
		}

		user := v1.Group("/user", middleware.AuthMiddleware(jwtManager))
		{
			user.GET("/profile", h.user.GetProfile)
			user.PUT("/profile", h.user.UpdateProfile)
			user.PUT("/password", h.user.ChangePassword)
			user.DELETE("", h.user.Delete)
		}

		authRequired := middleware.AuthMiddleware(jwtManager)
		wsMW := middleware.WorkspaceMiddleware(workspaceRepo)

		// Invitation accept — auth required, no workspace middleware
		v1.GET("/invitations/accept", authRequired, h.invitation.Accept)

		// Workspaces
		ws := v1.Group("/workspaces", authRequired)
		{
			ws.POST("", h.workspace.Create)
			ws.GET("", h.workspace.List)
		}
		wsMember := v1.Group("/workspaces/:workspace_id", authRequired, wsMW)
		{
			wsMember.GET("", h.workspace.Get)
			wsMember.GET("/summary", h.transaction.WorkspaceSummary)
			wsMember.PUT("", h.workspace.Update)
			wsMember.DELETE("", h.workspace.Delete)

			// Members
			wsMember.GET("/members", h.workspace.ListMembers)
			wsMember.PUT("/members/:user_id/role", h.workspace.UpdateMemberRole)
			wsMember.DELETE("/members/:user_id", h.workspace.RemoveMember)

			// Invitations
			wsMember.GET("/invitations", h.invitation.ListPending)
			wsMember.POST("/invitations", h.invitation.Send)
			wsMember.DELETE("/invitations/:invitation_id", h.invitation.Cancel)

			// Alerts
			wsMember.GET("/alerts", h.alert.GetAlerts)

			// Categories
			wsMember.GET("/categories", h.category.List)
			wsMember.POST("/categories", h.category.Create)
			wsMember.PUT("/categories/:category_id", h.category.Update)
			wsMember.DELETE("/categories/:category_id", h.category.Delete)

			// Transactions
			wsMember.GET("/transactions", h.transaction.List)
			wsMember.POST("/transactions", h.transaction.Create)
			wsMember.POST("/transactions/transfer", h.transaction.CreateTransfer)
			wsMember.GET("/transactions/summary", h.transaction.DailySummary)
			wsMember.GET("/transactions/by-date/:date", h.transaction.ListByDate)
			wsMember.GET("/transactions/:transaction_id", h.transaction.Get)
			wsMember.PUT("/transactions/:transaction_id", h.transaction.Update)
			wsMember.DELETE("/transactions/:transaction_id", h.transaction.Delete)
			wsMember.GET("/transactions/import/template", h.importH.Template)
			wsMember.POST("/transactions/import", h.importH.Import)

			// Budgets
			wsMember.GET("/budgets", h.budget.List)
			wsMember.PUT("/budgets/:year/:month", h.budget.Upsert)
			wsMember.GET("/budgets/:year/:month", h.budget.Get)
			wsMember.DELETE("/budgets/:year/:month", h.budget.Delete)
			wsMember.PUT("/budgets/:year/:month/categories/:category_id", h.budget.UpsertCategory)
			wsMember.DELETE("/budgets/:year/:month/categories/:category_id", h.budget.DeleteCategory)

			// Debts
			wsMember.GET("/debts", h.debt.List)
			wsMember.POST("/debts", h.debt.Create)
			wsMember.GET("/debts/:debt_id", h.debt.Get)
			wsMember.PUT("/debts/:debt_id", h.debt.Update)
			wsMember.DELETE("/debts/:debt_id", h.debt.Delete)
			wsMember.GET("/debts/:debt_id/schedule", h.debt.GetSchedule)
			wsMember.GET("/debts/:debt_id/payments", h.debt.ListPayments)
			wsMember.POST("/debts/:debt_id/payments", h.debt.RecordPayment)
			wsMember.PUT("/debts/:debt_id/payments/:payment_id", h.debt.UpdatePayment)
			wsMember.DELETE("/debts/:debt_id/payments/:payment_id", h.debt.DeletePayment)

			// Savings Goals
			wsMember.GET("/savings", h.savings.List)
			wsMember.POST("/savings", h.savings.Create)
			wsMember.GET("/savings/:goal_id", h.savings.Get)
			wsMember.PUT("/savings/:goal_id", h.savings.Update)
			wsMember.DELETE("/savings/:goal_id", h.savings.Delete)
			wsMember.GET("/savings/:goal_id/contributions", h.savings.ListContributions)
			wsMember.POST("/savings/:goal_id/contributions", h.savings.AddContribution)
			wsMember.DELETE("/savings/:goal_id/contributions/:contribution_id", h.savings.DeleteContribution)
		}
	}

	return r
}
