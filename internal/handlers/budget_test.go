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
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMemberChecker struct{ ok bool }

func (m mockMemberChecker) IsMember(_ context.Context, _, _ uuid.UUID) bool { return m.ok }

type mockBudgetService struct {
	upsertFn         func(ctx context.Context, p services.UpsertBudgetParams) (services.BudgetView, error)
	listFn           func(ctx context.Context, workspaceID uuid.UUID) ([]services.BudgetView, error)
	getWithProgressFn func(ctx context.Context, workspaceID uuid.UUID, year, month int16) (services.BudgetView, error)
	deleteFn         func(ctx context.Context, workspaceID uuid.UUID, year, month int16) error
	upsertCategoryFn func(ctx context.Context, workspaceID uuid.UUID, year, month int16, cat services.BudgetCategoryInput) error
	deleteCategoryFn func(ctx context.Context, workspaceID uuid.UUID, year, month int16, categoryID uuid.UUID) error
}

func (m *mockBudgetService) Upsert(ctx context.Context, p services.UpsertBudgetParams) (services.BudgetView, error) {
	return m.upsertFn(ctx, p)
}
func (m *mockBudgetService) List(ctx context.Context, workspaceID uuid.UUID) ([]services.BudgetView, error) {
	return m.listFn(ctx, workspaceID)
}
func (m *mockBudgetService) GetWithProgress(ctx context.Context, workspaceID uuid.UUID, year, month int16) (services.BudgetView, error) {
	return m.getWithProgressFn(ctx, workspaceID, year, month)
}
func (m *mockBudgetService) Delete(ctx context.Context, workspaceID uuid.UUID, year, month int16) error {
	return m.deleteFn(ctx, workspaceID, year, month)
}
func (m *mockBudgetService) UpsertCategory(ctx context.Context, workspaceID uuid.UUID, year, month int16, cat services.BudgetCategoryInput) error {
	return m.upsertCategoryFn(ctx, workspaceID, year, month, cat)
}
func (m *mockBudgetService) DeleteCategory(ctx context.Context, workspaceID uuid.UUID, year, month int16, categoryID uuid.UUID) error {
	return m.deleteCategoryFn(ctx, workspaceID, year, month, categoryID)
}

func newBudgetTestRouter(t *testing.T, svc budgetService, allowed bool) (*gin.Engine, string, uuid.UUID) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	wsID := uuid.New()
	jwt := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, time.Hour)
	pair, err := jwt.GenerateTokenPair(userID)
	require.NoError(t, err)

	h := NewBudgetHandler(svc)
	r := gin.New()
	grp := r.Group("/workspaces/:workspace_id", middleware.AuthMiddleware(jwt), middleware.WorkspaceMiddleware(mockMemberChecker{ok: allowed}))
	grp.PUT("/budgets/:year/:month", h.Upsert)
	grp.GET("/budgets", h.List)
	grp.GET("/budgets/:year/:month", h.Get)
	grp.DELETE("/budgets/:year/:month", h.Delete)
	grp.PUT("/budgets/:year/:month/categories/:category_id", h.UpsertCategory)
	grp.DELETE("/budgets/:year/:month/categories/:category_id", h.DeleteCategory)

	return r, pair.AccessToken, wsID
}

func doReq(r *gin.Engine, method, path, token string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestBudgetHandler_Upsert(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		body           any
		svc            *mockBudgetService
		expectedStatus int
	}{
		{
			name: "success",
			path: "/budgets/2026/6",
			body: map[string]any{"total_limit": 500000},
			svc: &mockBudgetService{
				upsertFn: func(_ context.Context, p services.UpsertBudgetParams) (services.BudgetView, error) {
					return services.BudgetView{WorkspaceID: p.WorkspaceID, Year: p.Year, Month: p.Month, TotalLimit: p.TotalLimit}, nil
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid body missing total_limit",
			path:           "/budgets/2026/6",
			body:           map[string]any{},
			svc:            &mockBudgetService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid month",
			path:           "/budgets/2026/13",
			body:           map[string]any{"total_limit": 100},
			svc:            &mockBudgetService{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, token, wsID := newBudgetTestRouter(t, tt.svc, true)
			w := doReq(r, http.MethodPut, "/workspaces/"+wsID.String()+tt.path, token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestBudgetHandler_List(t *testing.T) {
	svc := &mockBudgetService{
		listFn: func(_ context.Context, workspaceID uuid.UUID) ([]services.BudgetView, error) {
			return []services.BudgetView{{WorkspaceID: workspaceID, Year: 2026, Month: 6}}, nil
		},
	}
	r, token, wsID := newBudgetTestRouter(t, svc, true)
	w := doReq(r, http.MethodGet, "/workspaces/"+wsID.String()+"/budgets", token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var got []services.BudgetView
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Len(t, got, 1)
}

func TestBudgetHandler_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockBudgetService{
			getWithProgressFn: func(_ context.Context, workspaceID uuid.UUID, year, month int16) (services.BudgetView, error) {
				return services.BudgetView{WorkspaceID: workspaceID, Year: year, Month: month}, nil
			},
		}
		r, token, wsID := newBudgetTestRouter(t, svc, true)
		w := doReq(r, http.MethodGet, "/workspaces/"+wsID.String()+"/budgets/2026/6", token, nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc := &mockBudgetService{
			getWithProgressFn: func(_ context.Context, _ uuid.UUID, _, _ int16) (services.BudgetView, error) {
				return services.BudgetView{}, apperror.ErrNotFound
			},
		}
		r, token, wsID := newBudgetTestRouter(t, svc, true)
		w := doReq(r, http.MethodGet, "/workspaces/"+wsID.String()+"/budgets/2026/6", token, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestBudgetHandler_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &mockBudgetService{deleteFn: func(_ context.Context, _ uuid.UUID, _, _ int16) error { return nil }}
		r, token, wsID := newBudgetTestRouter(t, svc, true)
		w := doReq(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/budgets/2026/6", token, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc := &mockBudgetService{deleteFn: func(_ context.Context, _ uuid.UUID, _, _ int16) error { return apperror.ErrNotFound }}
		r, token, wsID := newBudgetTestRouter(t, svc, true)
		w := doReq(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/budgets/2026/6", token, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestBudgetHandler_UpsertCategory(t *testing.T) {
	catID := uuid.New()

	tests := []struct {
		name           string
		catIDInPath    string
		body           any
		svc            *mockBudgetService
		expectedStatus int
	}{
		{
			name:        "success",
			catIDInPath: catID.String(),
			body:        map[string]any{"limit_amount": 1000},
			svc: &mockBudgetService{
				upsertCategoryFn: func(_ context.Context, _ uuid.UUID, _, _ int16, _ services.BudgetCategoryInput) error { return nil },
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "invalid category_id",
			catIDInPath:    "not-a-uuid",
			body:           map[string]any{"limit_amount": 1000},
			svc:            &mockBudgetService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid body missing limit_amount",
			catIDInPath:    catID.String(),
			body:           map[string]any{},
			svc:            &mockBudgetService{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, token, wsID := newBudgetTestRouter(t, tt.svc, true)
			w := doReq(r, http.MethodPut, "/workspaces/"+wsID.String()+"/budgets/2026/6/categories/"+tt.catIDInPath, token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestBudgetHandler_DeleteCategory(t *testing.T) {
	catID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockBudgetService{
			deleteCategoryFn: func(_ context.Context, _ uuid.UUID, _, _ int16, _ uuid.UUID) error { return nil },
		}
		r, token, wsID := newBudgetTestRouter(t, svc, true)
		w := doReq(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/budgets/2026/6/categories/"+catID.String(), token, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("invalid category_id", func(t *testing.T) {
		r, token, wsID := newBudgetTestRouter(t, &mockBudgetService{}, true)
		w := doReq(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/budgets/2026/6/categories/not-a-uuid", token, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
