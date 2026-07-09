package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockDebtServiceForHandler struct {
	createFn        func(ctx context.Context, p services.CreateDebtParams) (models.Debt, error)
	getByIDFn       func(ctx context.Context, id, workspaceID uuid.UUID) (models.Debt, error)
	listFn          func(ctx context.Context, workspaceID uuid.UUID) ([]models.Debt, error)
	updateFn        func(ctx context.Context, p services.UpdateDebtParams) (models.Debt, error)
	deleteFn        func(ctx context.Context, id, workspaceID uuid.UUID) error
	getScheduleFn   func(ctx context.Context, id, workspaceID uuid.UUID) ([]models.DebtScheduleInstallment, error)
	recordPaymentFn func(ctx context.Context, workspaceID uuid.UUID, p services.RecordPaymentParams) (models.DebtPayment, error)
	listPaymentsFn  func(ctx context.Context, debtID, workspaceID uuid.UUID) ([]models.DebtPayment, error)
	updatePaymentFn func(ctx context.Context, workspaceID uuid.UUID, p services.UpdatePaymentParams) (models.DebtPayment, error)
	deletePaymentFn func(ctx context.Context, paymentID, debtID, workspaceID uuid.UUID) error
}

func (m *mockDebtServiceForHandler) Create(ctx context.Context, p services.CreateDebtParams) (models.Debt, error) {
	return m.createFn(ctx, p)
}
func (m *mockDebtServiceForHandler) GetByID(ctx context.Context, id, workspaceID uuid.UUID) (models.Debt, error) {
	return m.getByIDFn(ctx, id, workspaceID)
}
func (m *mockDebtServiceForHandler) List(ctx context.Context, workspaceID uuid.UUID) ([]models.Debt, error) {
	return m.listFn(ctx, workspaceID)
}
func (m *mockDebtServiceForHandler) Update(ctx context.Context, p services.UpdateDebtParams) (models.Debt, error) {
	return m.updateFn(ctx, p)
}
func (m *mockDebtServiceForHandler) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return m.deleteFn(ctx, id, workspaceID)
}
func (m *mockDebtServiceForHandler) GetSchedule(ctx context.Context, id, workspaceID uuid.UUID) ([]models.DebtScheduleInstallment, error) {
	return m.getScheduleFn(ctx, id, workspaceID)
}
func (m *mockDebtServiceForHandler) RecordPayment(ctx context.Context, workspaceID uuid.UUID, p services.RecordPaymentParams) (models.DebtPayment, error) {
	return m.recordPaymentFn(ctx, workspaceID, p)
}
func (m *mockDebtServiceForHandler) ListPayments(ctx context.Context, debtID, workspaceID uuid.UUID) ([]models.DebtPayment, error) {
	return m.listPaymentsFn(ctx, debtID, workspaceID)
}
func (m *mockDebtServiceForHandler) UpdatePayment(ctx context.Context, workspaceID uuid.UUID, p services.UpdatePaymentParams) (models.DebtPayment, error) {
	return m.updatePaymentFn(ctx, workspaceID, p)
}
func (m *mockDebtServiceForHandler) DeletePayment(ctx context.Context, paymentID, debtID, workspaceID uuid.UUID) error {
	return m.deletePaymentFn(ctx, paymentID, debtID, workspaceID)
}

type alwaysMember struct{}

func (alwaysMember) IsMember(_ context.Context, _, _ uuid.UUID) bool { return true }

func debtTestRouter(t *testing.T, svc debtService) (*gin.Engine, uuid.UUID, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	jwt := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, 7*24*time.Hour)
	tokens, err := jwt.GenerateTokenPair(userID)
	assert.NoError(t, err)

	h := NewDebtHandler(svc)
	r := gin.New()
	ws := r.Group("/workspaces/:workspace_id", middleware.AuthMiddleware(jwt), middleware.WorkspaceMiddleware(alwaysMember{}))
	ws.POST("/debts", h.Create)
	ws.GET("/debts", h.List)
	ws.GET("/debts/:debt_id", h.Get)
	ws.PUT("/debts/:debt_id", h.Update)
	ws.DELETE("/debts/:debt_id", h.Delete)
	ws.GET("/debts/:debt_id/schedule", h.GetSchedule)
	ws.POST("/debts/:debt_id/payments", h.RecordPayment)
	ws.GET("/debts/:debt_id/payments", h.ListPayments)
	ws.PUT("/debts/:debt_id/payments/:payment_id", h.UpdatePayment)
	ws.DELETE("/debts/:debt_id/payments/:payment_id", h.DeletePayment)

	return r, userID, tokens.AccessToken
}

func doAuthedRequest(r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	var reqBody *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reqBody = bytes.NewReader(b)
	} else {
		reqBody = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reqBody)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDebtHandler_Create(t *testing.T) {
	wsID := uuid.New()

	tests := []struct {
		name           string
		body           map[string]any
		mockFn         func(ctx context.Context, p services.CreateDebtParams) (models.Debt, error)
		expectedStatus int
	}{
		{
			name: "success",
			body: map[string]any{
				"name": "Car loan", "principal": 12000000, "rate_type": "effective_annual",
				"installments": 24, "first_payment_date": "2026-06-01",
			},
			mockFn: func(_ context.Context, p services.CreateDebtParams) (models.Debt, error) {
				return models.Debt{ID: uuid.New(), WorkspaceID: p.WorkspaceID, Name: p.Name}, nil
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing required field returns 400",
			body:           map[string]any{"lender": "Bank"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid date format returns 400",
			body: map[string]any{
				"name": "X", "principal": 100, "rate_type": "monthly",
				"installments": 12, "first_payment_date": "06/01/2026",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service validation error maps to 400",
			body: map[string]any{
				"name": "X", "principal": 100, "rate_type": "monthly",
				"installments": 12, "first_payment_date": "2026-06-01",
			},
			mockFn: func(_ context.Context, _ services.CreateDebtParams) (models.Debt, error) {
				return models.Debt{}, apperror.ErrInvalidInput
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockDebtServiceForHandler{createFn: tt.mockFn}
			r, _, token := debtTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodPost, "/workspaces/"+wsID.String()+"/debts", token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDebtHandler_List(t *testing.T) {
	wsID := uuid.New()
	svc := &mockDebtServiceForHandler{
		listFn: func(_ context.Context, _ uuid.UUID) ([]models.Debt, error) {
			return []models.Debt{{ID: uuid.New(), Name: "A"}}, nil
		},
	}
	r, _, token := debtTestRouter(t, svc)
	w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/debts", token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDebtHandler_Get(t *testing.T) {
	wsID := uuid.New()
	debtID := uuid.New()

	tests := []struct {
		name           string
		debtIDInPath   string
		mockFn         func(ctx context.Context, id, workspaceID uuid.UUID) (models.Debt, error)
		expectedStatus int
	}{
		{
			name:         "success",
			debtIDInPath: debtID.String(),
			mockFn: func(_ context.Context, id, _ uuid.UUID) (models.Debt, error) {
				return models.Debt{ID: id, Name: "Car loan"}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid debt_id returns 400",
			debtIDInPath:   "not-a-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "not found maps to 404",
			debtIDInPath: debtID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) (models.Debt, error) {
				return models.Debt{}, apperror.ErrNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockDebtServiceForHandler{getByIDFn: tt.mockFn}
			r, _, token := debtTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/debts/"+tt.debtIDInPath, token, nil)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDebtHandler_Update(t *testing.T) {
	wsID := uuid.New()
	debtID := uuid.New()
	body := map[string]any{
		"name": "Renamed", "principal": 500000, "rate_type": "monthly",
		"installments": 6, "first_payment_date": "2026-06-01",
	}

	svc := &mockDebtServiceForHandler{
		updateFn: func(_ context.Context, p services.UpdateDebtParams) (models.Debt, error) {
			return models.Debt{ID: p.ID, Name: p.Name}, nil
		},
	}
	r, _, token := debtTestRouter(t, svc)
	w := doAuthedRequest(r, http.MethodPut, "/workspaces/"+wsID.String()+"/debts/"+debtID.String(), token, body)
	assert.Equal(t, http.StatusOK, w.Code)

	w = doAuthedRequest(r, http.MethodPut, "/workspaces/"+wsID.String()+"/debts/not-a-uuid", token, body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDebtHandler_Delete(t *testing.T) {
	wsID := uuid.New()
	debtID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockDebtServiceForHandler{deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return nil }}
		r, _, token := debtTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/debts/"+debtID.String(), token, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("wrong workspace maps to 404", func(t *testing.T) {
		svc := &mockDebtServiceForHandler{deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return apperror.ErrNotFound }}
		r, _, token := debtTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/debts/"+debtID.String(), token, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDebtHandler_GetSchedule(t *testing.T) {
	wsID := uuid.New()
	debtID := uuid.New()
	svc := &mockDebtServiceForHandler{
		getScheduleFn: func(_ context.Context, _, _ uuid.UUID) ([]models.DebtScheduleInstallment, error) {
			return []models.DebtScheduleInstallment{{Period: 1}}, nil
		},
	}
	r, _, token := debtTestRouter(t, svc)
	w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/debts/"+debtID.String()+"/schedule", token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

// RecordPayment, UpdatePayment, DeletePayment move real money — highest-value coverage.

func TestDebtHandler_RecordPayment(t *testing.T) {
	wsID := uuid.New()
	debtID := uuid.New()

	tests := []struct {
		name           string
		body           map[string]any
		mockFn         func(ctx context.Context, workspaceID uuid.UUID, p services.RecordPaymentParams) (models.DebtPayment, error)
		expectedStatus int
	}{
		{
			name: "success",
			body: map[string]any{"period": 1, "amount": 500000, "paid_at": "2026-06-05"},
			mockFn: func(_ context.Context, _ uuid.UUID, p services.RecordPaymentParams) (models.DebtPayment, error) {
				return models.DebtPayment{ID: uuid.New(), DebtID: p.DebtID, Period: p.Period, Amount: p.Amount}, nil
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing amount returns 400",
			body:           map[string]any{"period": 1, "paid_at": "2026-06-05"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed paid_at returns 400",
			body:           map[string]any{"period": 1, "amount": 100, "paid_at": "not-a-date"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "period out of range maps to 400",
			body: map[string]any{"period": 99, "amount": 100, "paid_at": "2026-06-05"},
			mockFn: func(_ context.Context, _ uuid.UUID, _ services.RecordPaymentParams) (models.DebtPayment, error) {
				return models.DebtPayment{}, apperror.WithMessage(apperror.ErrInvalidInput, "period out of range")
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "debt not in workspace maps to 404",
			body: map[string]any{"period": 1, "amount": 100, "paid_at": "2026-06-05"},
			mockFn: func(_ context.Context, _ uuid.UUID, _ services.RecordPaymentParams) (models.DebtPayment, error) {
				return models.DebtPayment{}, apperror.ErrNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockDebtServiceForHandler{recordPaymentFn: tt.mockFn}
			r, _, token := debtTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodPost, "/workspaces/"+wsID.String()+"/debts/"+debtID.String()+"/payments", token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDebtHandler_ListPayments(t *testing.T) {
	wsID := uuid.New()
	debtID := uuid.New()
	svc := &mockDebtServiceForHandler{
		listPaymentsFn: func(_ context.Context, _, _ uuid.UUID) ([]models.DebtPayment, error) {
			return []models.DebtPayment{{ID: uuid.New()}}, nil
		},
	}
	r, _, token := debtTestRouter(t, svc)
	w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/debts/"+debtID.String()+"/payments", token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDebtHandler_UpdatePayment(t *testing.T) {
	wsID := uuid.New()
	debtID := uuid.New()
	paymentID := uuid.New()
	validBody := map[string]any{"amount": 200000, "paid_at": "2026-06-10"}

	tests := []struct {
		name           string
		paymentIDPath  string
		body           map[string]any
		mockFn         func(ctx context.Context, workspaceID uuid.UUID, p services.UpdatePaymentParams) (models.DebtPayment, error)
		expectedStatus int
	}{
		{
			name:          "success",
			paymentIDPath: paymentID.String(),
			body:          validBody,
			mockFn: func(_ context.Context, _ uuid.UUID, p services.UpdatePaymentParams) (models.DebtPayment, error) {
				return models.DebtPayment{ID: p.PaymentID, Amount: p.Amount}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid payment_id returns 400",
			paymentIDPath:  "not-a-uuid",
			body:           validBody,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:          "payment belongs to different debt maps to 404",
			paymentIDPath: paymentID.String(),
			body:          validBody,
			mockFn: func(_ context.Context, _ uuid.UUID, _ services.UpdatePaymentParams) (models.DebtPayment, error) {
				return models.DebtPayment{}, apperror.ErrNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockDebtServiceForHandler{updatePaymentFn: tt.mockFn}
			r, _, token := debtTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodPut, "/workspaces/"+wsID.String()+"/debts/"+debtID.String()+"/payments/"+tt.paymentIDPath, token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestDebtHandler_DeletePayment(t *testing.T) {
	wsID := uuid.New()
	debtID := uuid.New()
	paymentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockDebtServiceForHandler{deletePaymentFn: func(_ context.Context, _, _, _ uuid.UUID) error { return nil }}
		r, _, token := debtTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/debts/"+debtID.String()+"/payments/"+paymentID.String(), token, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("payment belongs to different debt maps to 404", func(t *testing.T) {
		svc := &mockDebtServiceForHandler{deletePaymentFn: func(_ context.Context, _, _, _ uuid.UUID) error { return apperror.ErrNotFound }}
		r, _, token := debtTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/debts/"+debtID.String()+"/payments/"+paymentID.String(), token, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
