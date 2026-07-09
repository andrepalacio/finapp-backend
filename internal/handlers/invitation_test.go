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

type mockInvitationService struct {
	sendFn        func(ctx context.Context, p services.SendInvitationParams) (services.InvitationView, error)
	acceptFn      func(ctx context.Context, token, userID uuid.UUID) (services.InvitationView, error)
	cancelFn      func(ctx context.Context, invID, workspaceID, requesterID uuid.UUID) error
	listPendingFn func(ctx context.Context, workspaceID uuid.UUID) ([]services.InvitationView, error)
}

func (m *mockInvitationService) Send(ctx context.Context, p services.SendInvitationParams) (services.InvitationView, error) {
	return m.sendFn(ctx, p)
}
func (m *mockInvitationService) Accept(ctx context.Context, token, userID uuid.UUID) (services.InvitationView, error) {
	return m.acceptFn(ctx, token, userID)
}
func (m *mockInvitationService) Cancel(ctx context.Context, invID, workspaceID, requesterID uuid.UUID) error {
	return m.cancelFn(ctx, invID, workspaceID, requesterID)
}
func (m *mockInvitationService) ListPending(ctx context.Context, workspaceID uuid.UUID) ([]services.InvitationView, error) {
	return m.listPendingFn(ctx, workspaceID)
}

func newInvitationRouter(svc *mockInvitationService, userID uuid.UUID, isMember bool) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewInvitationHandler(svc)
	r := gin.New()

	r.GET("/invitations/accept", withUserID(userID), h.Accept)

	wsGroup := r.Group("/workspaces/:workspace_id", withUserID(userID), middleware.WorkspaceMiddleware(stubMemberChecker{isMember: isMember}))
	{
		wsGroup.POST("/invitations", h.Send)
		wsGroup.GET("/invitations", h.ListPending)
		wsGroup.DELETE("/invitations/:invitation_id", h.Cancel)
	}
	return r
}

func TestInvitationHandler_Send(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockInvitationService{
			sendFn: func(_ context.Context, p services.SendInvitationParams) (services.InvitationView, error) {
				return services.InvitationView{ID: uuid.New(), WorkspaceID: p.WorkspaceID, Email: p.Email}, nil
			},
		}
		r := newInvitationRouter(svc, userID, true)

		body, _ := json.Marshal(map[string]string{"email": "a@b.com", "role": "member"})
		req := httptest.NewRequest(http.MethodPost, "/workspaces/"+wsID.String()+"/invitations", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("malformed email rejected by binding", func(t *testing.T) {
		svc := &mockInvitationService{}
		r := newInvitationRouter(svc, userID, true)

		body, _ := json.Marshal(map[string]string{"email": "not-an-email"})
		req := httptest.NewRequest(http.MethodPost, "/workspaces/"+wsID.String()+"/invitations", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("service forbidden propagates", func(t *testing.T) {
		svc := &mockInvitationService{
			sendFn: func(_ context.Context, _ services.SendInvitationParams) (services.InvitationView, error) {
				return services.InvitationView{}, apperror.ErrForbidden
			},
		}
		r := newInvitationRouter(svc, userID, true)

		body, _ := json.Marshal(map[string]string{"email": "a@b.com"})
		req := httptest.NewRequest(http.MethodPost, "/workspaces/"+wsID.String()+"/invitations", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestInvitationHandler_Accept(t *testing.T) {
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		token := uuid.New()
		svc := &mockInvitationService{
			acceptFn: func(_ context.Context, tok, uid uuid.UUID) (services.InvitationView, error) {
				return services.InvitationView{ID: uuid.New(), Token: tok, Status: "accepted"}, nil
			},
		}
		r := newInvitationRouter(svc, userID, true)

		req := httptest.NewRequest(http.MethodGet, "/invitations/accept?token="+token.String(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid token format", func(t *testing.T) {
		svc := &mockInvitationService{}
		r := newInvitationRouter(svc, userID, true)

		req := httptest.NewRequest(http.MethodGet, "/invitations/accept?token=not-a-uuid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("expired or invalid invitation", func(t *testing.T) {
		svc := &mockInvitationService{
			acceptFn: func(_ context.Context, _, _ uuid.UUID) (services.InvitationView, error) {
				return services.InvitationView{}, apperror.WithMessage(apperror.ErrInvalidInput, "invitation expired")
			},
		}
		r := newInvitationRouter(svc, userID, true)

		req := httptest.NewRequest(http.MethodGet, "/invitations/accept?token="+uuid.New().String(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestInvitationHandler_Cancel(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	invID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockInvitationService{
			cancelFn: func(_ context.Context, _, _, _ uuid.UUID) error { return nil },
		}
		r := newInvitationRouter(svc, userID, true)

		req := httptest.NewRequest(http.MethodDelete, "/workspaces/"+wsID.String()+"/invitations/"+invID.String(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("invalid invitation_id", func(t *testing.T) {
		svc := &mockInvitationService{}
		r := newInvitationRouter(svc, userID, true)

		req := httptest.NewRequest(http.MethodDelete, "/workspaces/"+wsID.String()+"/invitations/not-a-uuid", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("not found", func(t *testing.T) {
		svc := &mockInvitationService{
			cancelFn: func(_ context.Context, _, _, _ uuid.UUID) error { return apperror.ErrNotFound },
		}
		r := newInvitationRouter(svc, userID, true)

		req := httptest.NewRequest(http.MethodDelete, "/workspaces/"+wsID.String()+"/invitations/"+invID.String(), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestInvitationHandler_ListPending(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()
	svc := &mockInvitationService{
		listPendingFn: func(_ context.Context, _ uuid.UUID) ([]services.InvitationView, error) {
			return []services.InvitationView{{ID: uuid.New(), Email: "a@b.com"}}, nil
		},
	}
	r := newInvitationRouter(svc, userID, true)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+wsID.String()+"/invitations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
