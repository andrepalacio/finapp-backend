package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAlertBudgetService struct {
	getWithProgressFn func(ctx context.Context, workspaceID uuid.UUID, year, month int16) (services.BudgetView, error)
}

func (m *mockAlertBudgetService) GetWithProgress(ctx context.Context, workspaceID uuid.UUID, year, month int16) (services.BudgetView, error) {
	return m.getWithProgressFn(ctx, workspaceID, year, month)
}

func newAlertTestRouter(t *testing.T, svc alertBudgetService) (*gin.Engine, string, uuid.UUID) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	wsID := uuid.New()
	jwt := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, time.Hour)
	pair, err := jwt.GenerateTokenPair(userID)
	require.NoError(t, err)

	h := NewAlertHandler(svc)
	r := gin.New()
	grp := r.Group("/workspaces/:workspace_id", middleware.AuthMiddleware(jwt), middleware.WorkspaceMiddleware(mockMemberChecker{ok: true}))
	grp.GET("/alerts", h.GetAlerts)

	return r, pair.AccessToken, wsID
}

func TestAlertHandler_GetAlerts(t *testing.T) {
	catID := uuid.New()

	tests := []struct {
		name        string
		budget      services.BudgetView
		budgetErr   error
		wantTypes   []string
	}{
		{
			name:      "no budget set returns empty alerts",
			budgetErr: apperror.ErrNotFound,
			wantTypes: []string{},
		},
		{
			name:      "under budget returns no alerts",
			budget:    services.BudgetView{TotalLimit: 1000, TotalSpent: 100},
			wantTypes: []string{},
		},
		{
			name:      "budget exceeded",
			budget:    services.BudgetView{TotalLimit: 1000, TotalSpent: 1500},
			wantTypes: []string{"budget_exceeded"},
		},
		{
			name:      "budget warning at 90%",
			budget:    services.BudgetView{TotalLimit: 1000, TotalSpent: 900},
			wantTypes: []string{"budget_warning"},
		},
		{
			name: "category exceeded",
			budget: services.BudgetView{
				TotalLimit: 1000, TotalSpent: 200,
				Categories: []services.BudgetCategoryProgressView{
					{CategoryID: catID, CategoryName: "Food", LimitAmount: 100, Spent: 150},
				},
			},
			wantTypes: []string{"category_exceeded"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockAlertBudgetService{
				getWithProgressFn: func(_ context.Context, _ uuid.UUID, _, _ int16) (services.BudgetView, error) {
					return tt.budget, tt.budgetErr
				},
			}
			r, token, wsID := newAlertTestRouter(t, svc)
			w := doReq(r, http.MethodGet, "/workspaces/"+wsID.String()+"/alerts", token, nil)
			assert.Equal(t, http.StatusOK, w.Code)

			var resp AlertsResponse
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
			gotTypes := make([]string, len(resp.Alerts))
			for i, a := range resp.Alerts {
				gotTypes[i] = a.Type
			}
			assert.Equal(t, tt.wantTypes, gotTypes)
		})
	}
}
