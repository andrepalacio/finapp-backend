package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockTransactionServiceForHandler struct {
	createFn         func(ctx context.Context, p services.CreateTransactionParams) (services.TransactionView, error)
	createTransferFn func(ctx context.Context, p services.CreateTransferParams) (services.TransferResult, error)
	getByIDFn        func(ctx context.Context, id uuid.UUID) (services.TransactionView, error)
	listFn           func(ctx context.Context, p services.ListTransactionsParams) (services.TransactionListResult, error)
	dailySummaryFn   func(ctx context.Context, p services.DailySummaryParams) (services.DailySummaryResult, error)
	monthSummaryFn   func(ctx context.Context, p repositories.MonthSummaryParams) (repositories.MonthSummaryResult, error)
	listByDateFn     func(ctx context.Context, p services.ListByDateParams) (services.CursorListResult, error)
	updateFn         func(ctx context.Context, p services.UpdateTransactionParams) (services.TransactionView, error)
	deleteFn         func(ctx context.Context, id, workspaceID uuid.UUID) error
}

func (m *mockTransactionServiceForHandler) Create(ctx context.Context, p services.CreateTransactionParams) (services.TransactionView, error) {
	return m.createFn(ctx, p)
}
func (m *mockTransactionServiceForHandler) CreateTransfer(ctx context.Context, p services.CreateTransferParams) (services.TransferResult, error) {
	return m.createTransferFn(ctx, p)
}
func (m *mockTransactionServiceForHandler) GetByID(ctx context.Context, id uuid.UUID) (services.TransactionView, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockTransactionServiceForHandler) List(ctx context.Context, p services.ListTransactionsParams) (services.TransactionListResult, error) {
	return m.listFn(ctx, p)
}
func (m *mockTransactionServiceForHandler) DailySummary(ctx context.Context, p services.DailySummaryParams) (services.DailySummaryResult, error) {
	return m.dailySummaryFn(ctx, p)
}
func (m *mockTransactionServiceForHandler) MonthSummary(ctx context.Context, p repositories.MonthSummaryParams) (repositories.MonthSummaryResult, error) {
	return m.monthSummaryFn(ctx, p)
}
func (m *mockTransactionServiceForHandler) ListByDate(ctx context.Context, p services.ListByDateParams) (services.CursorListResult, error) {
	return m.listByDateFn(ctx, p)
}
func (m *mockTransactionServiceForHandler) Update(ctx context.Context, p services.UpdateTransactionParams) (services.TransactionView, error) {
	return m.updateFn(ctx, p)
}
func (m *mockTransactionServiceForHandler) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return m.deleteFn(ctx, id, workspaceID)
}

func transactionTestRouter(t *testing.T, svc transactionService) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	jwt := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, 7*24*time.Hour)
	tokens, err := jwt.GenerateTokenPair(userID)
	assert.NoError(t, err)

	h := NewTransactionHandler(svc)
	r := gin.New()
	ws := r.Group("/workspaces/:workspace_id", middleware.AuthMiddleware(jwt), middleware.WorkspaceMiddleware(alwaysMember{}))
	ws.POST("/transactions", h.Create)
	ws.POST("/transactions/transfer", h.CreateTransfer)
	ws.GET("/transactions/:transaction_id", h.Get)
	ws.GET("/transactions", h.List)
	ws.GET("/transactions/summary", h.DailySummary)
	ws.GET("/transactions/by-date/:date", h.ListByDate)
	ws.PUT("/transactions/:transaction_id", h.Update)
	ws.DELETE("/transactions/:transaction_id", h.Delete)
	ws.GET("/summary", h.WorkspaceSummary)

	return r, tokens.AccessToken
}

func TestTransactionHandler_Create(t *testing.T) {
	wsID := uuid.New()

	tests := []struct {
		name           string
		body           map[string]any
		mockFn         func(ctx context.Context, p services.CreateTransactionParams) (services.TransactionView, error)
		expectedStatus int
	}{
		{
			name: "success",
			body: map[string]any{"type": "expense", "amount": 50000, "date": "2026-06-01"},
			mockFn: func(_ context.Context, p services.CreateTransactionParams) (services.TransactionView, error) {
				return services.TransactionView{ID: uuid.New(), Type: p.Type, Amount: p.Amount}, nil
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "invalid type returns 400",
			body:           map[string]any{"type": "bogus", "amount": 50000, "date": "2026-06-01"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "missing amount returns 400",
			body:           map[string]any{"type": "expense", "date": "2026-06-01"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed date returns 400",
			body:           map[string]any{"type": "expense", "amount": 100, "date": "not-a-date"},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTransactionServiceForHandler{createFn: tt.mockFn}
			r, token := transactionTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodPost, "/workspaces/"+wsID.String()+"/transactions", token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

// CreateTransfer moves money between two workspaces — highest-value coverage in this file.

func TestTransactionHandler_CreateTransfer(t *testing.T) {
	wsID := uuid.New()
	toWsID := uuid.New()

	tests := []struct {
		name           string
		body           map[string]any
		mockFn         func(ctx context.Context, p services.CreateTransferParams) (services.TransferResult, error)
		expectedStatus int
	}{
		{
			name: "success",
			body: map[string]any{"to_workspace_id": toWsID.String(), "amount": 100000, "date": "2026-06-01"},
			mockFn: func(_ context.Context, p services.CreateTransferParams) (services.TransferResult, error) {
				return services.TransferResult{
					Out: services.TransactionView{ID: uuid.New(), TransferDirection: "out", Amount: p.Amount},
					In:  services.TransactionView{ID: uuid.New(), TransferDirection: "in", Amount: p.Amount},
				}, nil
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing to_workspace_id returns 400",
			body:           map[string]any{"amount": 100000, "date": "2026-06-01"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "zero amount returns 400 at bind time",
			body:           map[string]any{"to_workspace_id": toWsID.String(), "amount": 0, "date": "2026-06-01"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed date returns 400",
			body:           map[string]any{"to_workspace_id": toWsID.String(), "amount": 100, "date": "06/01/2026"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service validation error maps to 400",
			body: map[string]any{"to_workspace_id": toWsID.String(), "amount": 100, "date": "2026-06-01"},
			mockFn: func(_ context.Context, _ services.CreateTransferParams) (services.TransferResult, error) {
				return services.TransferResult{}, apperror.ErrInvalidInput
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTransactionServiceForHandler{createTransferFn: tt.mockFn}
			r, token := transactionTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodPost, "/workspaces/"+wsID.String()+"/transactions/transfer", token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestTransactionHandler_Get(t *testing.T) {
	wsID := uuid.New()
	txID := uuid.New()

	tests := []struct {
		name           string
		txIDInPath     string
		mockFn         func(ctx context.Context, id uuid.UUID) (services.TransactionView, error)
		expectedStatus int
	}{
		{
			name:       "success",
			txIDInPath: txID.String(),
			mockFn: func(_ context.Context, id uuid.UUID) (services.TransactionView, error) {
				return services.TransactionView{ID: id}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid transaction_id returns 400",
			txIDInPath:     "not-a-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:       "not found maps to 404",
			txIDInPath: txID.String(),
			mockFn: func(_ context.Context, _ uuid.UUID) (services.TransactionView, error) {
				return services.TransactionView{}, apperror.ErrNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockTransactionServiceForHandler{getByIDFn: tt.mockFn}
			r, token := transactionTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/transactions/"+tt.txIDInPath, token, nil)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestTransactionHandler_List(t *testing.T) {
	wsID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{
			listFn: func(_ context.Context, _ services.ListTransactionsParams) (services.TransactionListResult, error) {
				return services.TransactionListResult{Total: 1}, nil
			},
		}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/transactions", token, nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid category_id query param returns 400", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/transactions?category_id=not-a-uuid", token, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid date_from query param returns 400", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/transactions?date_from=bad", token, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTransactionHandler_DailySummary(t *testing.T) {
	wsID := uuid.New()
	svc := &mockTransactionServiceForHandler{
		dailySummaryFn: func(_ context.Context, _ services.DailySummaryParams) (services.DailySummaryResult, error) {
			return services.DailySummaryResult{}, nil
		},
	}
	r, token := transactionTestRouter(t, svc)
	w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/transactions/summary", token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTransactionHandler_ListByDate(t *testing.T) {
	wsID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{
			listByDateFn: func(_ context.Context, _ services.ListByDateParams) (services.CursorListResult, error) {
				return services.CursorListResult{}, nil
			},
		}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/transactions/by-date/2026-06-01", token, nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid date path param returns 400", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/transactions/by-date/not-a-date", token, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTransactionHandler_Update(t *testing.T) {
	wsID := uuid.New()
	txID := uuid.New()
	body := map[string]any{"amount": 200, "date": "2026-06-01"}

	t.Run("success", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{
			updateFn: func(_ context.Context, p services.UpdateTransactionParams) (services.TransactionView, error) {
				return services.TransactionView{ID: p.ID, Amount: p.Amount}, nil
			},
		}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodPut, "/workspaces/"+wsID.String()+"/transactions/"+txID.String(), token, body)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("transfer blocked maps to 403", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{
			updateFn: func(_ context.Context, _ services.UpdateTransactionParams) (services.TransactionView, error) {
				return services.TransactionView{}, apperror.ErrForbidden
			},
		}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodPut, "/workspaces/"+wsID.String()+"/transactions/"+txID.String(), token, body)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid transaction_id returns 400", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodPut, "/workspaces/"+wsID.String()+"/transactions/not-a-uuid", token, body)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestTransactionHandler_Delete(t *testing.T) {
	wsID := uuid.New()
	txID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return nil }}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/transactions/"+txID.String(), token, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("wrong workspace maps to 403", func(t *testing.T) {
		svc := &mockTransactionServiceForHandler{deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return apperror.ErrForbidden }}
		r, token := transactionTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/transactions/"+txID.String(), token, nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestTransactionHandler_WorkspaceSummary(t *testing.T) {
	wsID := uuid.New()
	svc := &mockTransactionServiceForHandler{
		monthSummaryFn: func(_ context.Context, _ repositories.MonthSummaryParams) (repositories.MonthSummaryResult, error) {
			return repositories.MonthSummaryResult{IncomeTotal: 100, ExpenseTotal: 50}, nil
		},
	}
	r, token := transactionTestRouter(t, svc)
	w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/summary", token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}
