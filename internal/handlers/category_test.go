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

type mockCategoryService struct {
	createFn           func(ctx context.Context, p services.CreateCategoryParams) (services.CategoryView, error)
	listForWorkspaceFn func(ctx context.Context, workspaceID uuid.UUID) ([]services.CategoryView, error)
	updateFn           func(ctx context.Context, p services.UpdateCategoryParams) (services.CategoryView, error)
	deleteFn           func(ctx context.Context, id, workspaceID uuid.UUID) error
}

func (m *mockCategoryService) Create(ctx context.Context, p services.CreateCategoryParams) (services.CategoryView, error) {
	return m.createFn(ctx, p)
}
func (m *mockCategoryService) ListForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]services.CategoryView, error) {
	return m.listForWorkspaceFn(ctx, workspaceID)
}
func (m *mockCategoryService) Update(ctx context.Context, p services.UpdateCategoryParams) (services.CategoryView, error) {
	return m.updateFn(ctx, p)
}
func (m *mockCategoryService) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return m.deleteFn(ctx, id, workspaceID)
}

func newCategoryTestRouter(t *testing.T, svc categoryService) (*gin.Engine, string, uuid.UUID) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	wsID := uuid.New()
	jwt := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, time.Hour)
	pair, err := jwt.GenerateTokenPair(userID)
	require.NoError(t, err)

	h := NewCategoryHandler(svc)
	r := gin.New()
	grp := r.Group("/workspaces/:workspace_id", middleware.AuthMiddleware(jwt), middleware.WorkspaceMiddleware(mockMemberChecker{ok: true}))
	grp.POST("/categories", h.Create)
	grp.GET("/categories", h.List)
	grp.PUT("/categories/:category_id", h.Update)
	grp.DELETE("/categories/:category_id", h.Delete)

	return r, pair.AccessToken, wsID
}

func TestCategoryHandler_Create(t *testing.T) {
	tests := []struct {
		name           string
		body           any
		svc            *mockCategoryService
		expectedStatus int
	}{
		{
			name: "success",
			body: map[string]any{"name": "Gym", "type": "expense"},
			svc: &mockCategoryService{
				createFn: func(_ context.Context, p services.CreateCategoryParams) (services.CategoryView, error) {
					return services.CategoryView{ID: uuid.New(), Name: p.Name, Type: p.Type}, nil
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing name",
			body:           map[string]any{"type": "expense"},
			svc:            &mockCategoryService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid type",
			body:           map[string]any{"name": "Gym", "type": "bogus"},
			svc:            &mockCategoryService{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, token, wsID := newCategoryTestRouter(t, tt.svc)
			w := doReq(r, http.MethodPost, "/workspaces/"+wsID.String()+"/categories", token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCategoryHandler_List(t *testing.T) {
	svc := &mockCategoryService{
		listForWorkspaceFn: func(_ context.Context, workspaceID uuid.UUID) ([]services.CategoryView, error) {
			return []services.CategoryView{{ID: uuid.New(), Name: "Food", Type: "expense"}}, nil
		},
	}
	r, token, wsID := newCategoryTestRouter(t, svc)
	w := doReq(r, http.MethodGet, "/workspaces/"+wsID.String()+"/categories", token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var got []services.CategoryView
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Len(t, got, 1)
}

func TestCategoryHandler_Update(t *testing.T) {
	catID := uuid.New()

	tests := []struct {
		name           string
		catIDInPath    string
		body           any
		svc            *mockCategoryService
		expectedStatus int
	}{
		{
			name:        "success",
			catIDInPath: catID.String(),
			body:        map[string]any{"name": "Renamed", "type": "expense"},
			svc: &mockCategoryService{
				updateFn: func(_ context.Context, p services.UpdateCategoryParams) (services.CategoryView, error) {
					return services.CategoryView{ID: p.ID, Name: p.Name, Type: p.Type}, nil
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid category_id",
			catIDInPath:    "not-a-uuid",
			body:           map[string]any{"name": "X"},
			svc:            &mockCategoryService{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:        "system category forbidden",
			catIDInPath: catID.String(),
			body:        map[string]any{"name": "X", "type": "expense"},
			svc: &mockCategoryService{
				updateFn: func(_ context.Context, _ services.UpdateCategoryParams) (services.CategoryView, error) {
					return services.CategoryView{}, apperror.ErrForbidden
				},
			},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, token, wsID := newCategoryTestRouter(t, tt.svc)
			w := doReq(r, http.MethodPut, "/workspaces/"+wsID.String()+"/categories/"+tt.catIDInPath, token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCategoryHandler_Delete(t *testing.T) {
	catID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockCategoryService{deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return nil }}
		r, token, wsID := newCategoryTestRouter(t, svc)
		w := doReq(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/categories/"+catID.String(), token, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("system category forbidden", func(t *testing.T) {
		svc := &mockCategoryService{deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return apperror.ErrForbidden }}
		r, token, wsID := newCategoryTestRouter(t, svc)
		w := doReq(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/categories/"+catID.String(), token, nil)
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("invalid category_id", func(t *testing.T) {
		r, token, wsID := newCategoryTestRouter(t, &mockCategoryService{})
		w := doReq(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/categories/not-a-uuid", token, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
