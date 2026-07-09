package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/services"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockWorkspaceService struct {
	createFn           func(ctx context.Context, p services.CreateWorkspaceParams) (services.WorkspaceView, error)
	getByIDFn          func(ctx context.Context, id uuid.UUID) (services.WorkspaceView, error)
	listByUserFn       func(ctx context.Context, userID uuid.UUID) ([]services.WorkspaceView, error)
	updateFn           func(ctx context.Context, p services.UpdateWorkspaceParams) (services.WorkspaceView, error)
	deleteFn           func(ctx context.Context, id, userID uuid.UUID) error
	listMembersFn      func(ctx context.Context, workspaceID uuid.UUID) ([]services.MemberView, error)
	updateMemberRoleFn func(ctx context.Context, p services.UpdateMemberRoleParams) error
	removeMemberFn     func(ctx context.Context, workspaceID, targetID, requesterID uuid.UUID) error
}

func (m *mockWorkspaceService) Create(ctx context.Context, p services.CreateWorkspaceParams) (services.WorkspaceView, error) {
	return m.createFn(ctx, p)
}
func (m *mockWorkspaceService) GetByID(ctx context.Context, id uuid.UUID) (services.WorkspaceView, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockWorkspaceService) ListByUser(ctx context.Context, userID uuid.UUID) ([]services.WorkspaceView, error) {
	return m.listByUserFn(ctx, userID)
}
func (m *mockWorkspaceService) Update(ctx context.Context, p services.UpdateWorkspaceParams) (services.WorkspaceView, error) {
	return m.updateFn(ctx, p)
}
func (m *mockWorkspaceService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return m.deleteFn(ctx, id, userID)
}
func (m *mockWorkspaceService) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]services.MemberView, error) {
	return m.listMembersFn(ctx, workspaceID)
}
func (m *mockWorkspaceService) UpdateMemberRole(ctx context.Context, p services.UpdateMemberRoleParams) error {
	return m.updateMemberRoleFn(ctx, p)
}
func (m *mockWorkspaceService) RemoveMember(ctx context.Context, workspaceID, targetID, requesterID uuid.UUID) error {
	return m.removeMemberFn(ctx, workspaceID, targetID, requesterID)
}

type stubMemberChecker struct {
	isMember bool
}

func (s stubMemberChecker) IsMember(_ context.Context, _, _ uuid.UUID) bool {
	return s.isMember
}

// withUserID injects userID into gin context the same way AuthMiddleware does.
func withUserID(userID uuid.UUID) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.UserIDContextKey, userID)
		c.Next()
	}
}

func newWorkspaceRouter(svc *mockWorkspaceService, userID uuid.UUID, isMember bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewWorkspaceHandler(svc)
	r := gin.New()

	r.POST("/workspaces", withUserID(userID), h.Create)
	r.GET("/workspaces", withUserID(userID), h.List)

	wsGroup := r.Group("/workspaces/:workspace_id", withUserID(userID), middleware.WorkspaceMiddleware(stubMemberChecker{isMember: isMember}))
	{
		wsGroup.GET("", h.Get)
		wsGroup.PUT("", h.Update)
		wsGroup.DELETE("", h.Delete)
		wsGroup.GET("/members", h.ListMembers)
		wsGroup.PUT("/members/:user_id/role", h.UpdateMemberRole)
		wsGroup.DELETE("/members/:user_id", h.RemoveMember)
	}
	return r
}

func TestWorkspaceHandler_Create(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockWorkspaceService{
			createFn: func(_ context.Context, p services.CreateWorkspaceParams) (services.WorkspaceView, error) {
				return services.WorkspaceView{ID: uuid.New(), Name: p.Name, OwnerID: p.OwnerID, Currency: "COP"}, nil
			},
		}
		r := newWorkspaceRouter(svc, userID, true)

		body, _ := json.Marshal(map[string]string{"name": "Home"})
		req := httptest.NewRequest(http.MethodPost, "/workspaces", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("malformed body", func(t *testing.T) {
		svc := &mockWorkspaceService{}
		r := newWorkspaceRouter(svc, userID, true)

		req := httptest.NewRequest(http.MethodPost, "/workspaces", bytes.NewReader([]byte("{not json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("missing required name", func(t *testing.T) {
		svc := &mockWorkspaceService{}
		r := newWorkspaceRouter(svc, userID, true)

		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/workspaces", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestWorkspaceHandler_List(t *testing.T) {
	userID := uuid.New()
	svc := &mockWorkspaceService{
		listByUserFn: func(_ context.Context, _ uuid.UUID) ([]services.WorkspaceView, error) {
			return []services.WorkspaceView{{ID: uuid.New(), Name: "A"}}, nil
		},
	}
	r := newWorkspaceRouter(svc, userID, true)

	req := httptest.NewRequest(http.MethodGet, "/workspaces", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_Get_NotMember(t *testing.T) {
	userID := uuid.New()
	svc := &mockWorkspaceService{}
	r := newWorkspaceRouter(svc, userID, false)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestWorkspaceHandler_Get_InvalidWorkspaceID(t *testing.T) {
	userID := uuid.New()
	svc := &mockWorkspaceService{}
	r := newWorkspaceRouter(svc, userID, true)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/not-a-uuid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWorkspaceHandler_Get_Success(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	svc := &mockWorkspaceService{
		getByIDFn: func(_ context.Context, id uuid.UUID) (services.WorkspaceView, error) {
			return services.WorkspaceView{ID: id, Name: "Home"}, nil
		},
	}
	r := newWorkspaceRouter(svc, userID, true)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+wsID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_Update_Forbidden(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	svc := &mockWorkspaceService{
		updateFn: func(_ context.Context, _ services.UpdateWorkspaceParams) (services.WorkspaceView, error) {
			return services.WorkspaceView{}, apperror.ErrForbidden
		},
	}
	r := newWorkspaceRouter(svc, userID, true)

	body, _ := json.Marshal(map[string]string{"name": "New name"})
	req := httptest.NewRequest(http.MethodPut, "/workspaces/"+wsID.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestWorkspaceHandler_Delete_Success(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	deleteCalled := false
	svc := &mockWorkspaceService{
		deleteFn: func(_ context.Context, _, _ uuid.UUID) error { deleteCalled = true; return nil },
	}
	r := newWorkspaceRouter(svc, userID, true)

	req := httptest.NewRequest(http.MethodDelete, "/workspaces/"+wsID.String(), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.True(t, deleteCalled)
}

func TestWorkspaceHandler_ListMembers(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	svc := &mockWorkspaceService{
		listMembersFn: func(_ context.Context, _ uuid.UUID) ([]services.MemberView, error) {
			return []services.MemberView{{UserID: uuid.New(), Role: "member"}}, nil
		},
	}
	r := newWorkspaceRouter(svc, userID, true)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+wsID.String()+"/members", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestWorkspaceHandler_UpdateMemberRole(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	targetID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockWorkspaceService{
			updateMemberRoleFn: func(_ context.Context, _ services.UpdateMemberRoleParams) error { return nil },
		}
		r := newWorkspaceRouter(svc, userID, true)

		body, _ := json.Marshal(map[string]string{"role": "admin"})
		req := httptest.NewRequest(http.MethodPut, "/workspaces/"+wsID.String()+"/members/"+targetID.String()+"/role", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("invalid role rejected by service", func(t *testing.T) {
		svc := &mockWorkspaceService{
			updateMemberRoleFn: func(_ context.Context, _ services.UpdateMemberRoleParams) error { return apperror.ErrInvalidInput },
		}
		r := newWorkspaceRouter(svc, userID, true)

		body, _ := json.Marshal(map[string]string{"role": "bogus"})
		req := httptest.NewRequest(http.MethodPut, "/workspaces/"+wsID.String()+"/members/"+targetID.String()+"/role", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid user_id path param", func(t *testing.T) {
		svc := &mockWorkspaceService{}
		r := newWorkspaceRouter(svc, userID, true)

		body, _ := json.Marshal(map[string]string{"role": "admin"})
		req := httptest.NewRequest(http.MethodPut, "/workspaces/"+wsID.String()+"/members/not-a-uuid/role", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestWorkspaceHandler_RemoveMember(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	targetID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockWorkspaceService{
			removeMemberFn: func(_ context.Context, _, _, _ uuid.UUID) error { return nil },
		}
		r := newWorkspaceRouter(svc, userID, true)

		req := httptest.NewRequest(http.MethodDelete, "/workspaces/"+wsID.String()+"/members/"+targetID.String(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc := &mockWorkspaceService{
			removeMemberFn: func(_ context.Context, _, _, _ uuid.UUID) error { return apperror.ErrNotFound },
		}
		r := newWorkspaceRouter(svc, userID, true)

		req := httptest.NewRequest(http.MethodDelete, "/workspaces/"+wsID.String()+"/members/"+targetID.String(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
