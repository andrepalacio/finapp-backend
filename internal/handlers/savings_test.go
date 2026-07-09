package handlers

import (
	"context"
	"net/http"
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

type mockSavingsServiceForHandler struct {
	createFn             func(ctx context.Context, p services.CreateSavingsGoalParams) (models.SavingsGoal, error)
	getByIDFn            func(ctx context.Context, id, workspaceID uuid.UUID) (models.SavingsGoal, error)
	listGoalsFn          func(ctx context.Context, workspaceID uuid.UUID) ([]services.SavingsGoalProgress, error)
	updateFn             func(ctx context.Context, p services.UpdateSavingsGoalParams) (models.SavingsGoal, error)
	deleteFn             func(ctx context.Context, id, workspaceID uuid.UUID) error
	getWithProgressFn    func(ctx context.Context, id, workspaceID uuid.UUID) (services.SavingsGoalProgress, error)
	addContributionFn    func(ctx context.Context, workspaceID uuid.UUID, p services.AddContributionParams) (models.SavingsContribution, error)
	listContributionsFn  func(ctx context.Context, goalID, workspaceID uuid.UUID) ([]models.SavingsContribution, error)
	deleteContributionFn func(ctx context.Context, contribID, goalID, workspaceID uuid.UUID) error
}

func (m *mockSavingsServiceForHandler) Create(ctx context.Context, p services.CreateSavingsGoalParams) (models.SavingsGoal, error) {
	return m.createFn(ctx, p)
}
func (m *mockSavingsServiceForHandler) GetByID(ctx context.Context, id, workspaceID uuid.UUID) (models.SavingsGoal, error) {
	return m.getByIDFn(ctx, id, workspaceID)
}
func (m *mockSavingsServiceForHandler) ListGoals(ctx context.Context, workspaceID uuid.UUID) ([]services.SavingsGoalProgress, error) {
	return m.listGoalsFn(ctx, workspaceID)
}
func (m *mockSavingsServiceForHandler) Update(ctx context.Context, p services.UpdateSavingsGoalParams) (models.SavingsGoal, error) {
	return m.updateFn(ctx, p)
}
func (m *mockSavingsServiceForHandler) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return m.deleteFn(ctx, id, workspaceID)
}
func (m *mockSavingsServiceForHandler) GetWithProgress(ctx context.Context, id, workspaceID uuid.UUID) (services.SavingsGoalProgress, error) {
	return m.getWithProgressFn(ctx, id, workspaceID)
}
func (m *mockSavingsServiceForHandler) AddContribution(ctx context.Context, workspaceID uuid.UUID, p services.AddContributionParams) (models.SavingsContribution, error) {
	return m.addContributionFn(ctx, workspaceID, p)
}
func (m *mockSavingsServiceForHandler) ListContributions(ctx context.Context, goalID, workspaceID uuid.UUID) ([]models.SavingsContribution, error) {
	return m.listContributionsFn(ctx, goalID, workspaceID)
}
func (m *mockSavingsServiceForHandler) DeleteContribution(ctx context.Context, contribID, goalID, workspaceID uuid.UUID) error {
	return m.deleteContributionFn(ctx, contribID, goalID, workspaceID)
}

func savingsTestRouter(t *testing.T, svc savingsService) (*gin.Engine, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	userID := uuid.New()
	jwt := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, 7*24*time.Hour)
	tokens, err := jwt.GenerateTokenPair(userID)
	assert.NoError(t, err)

	h := NewSavingsHandler(svc)
	r := gin.New()
	ws := r.Group("/workspaces/:workspace_id", middleware.AuthMiddleware(jwt), middleware.WorkspaceMiddleware(alwaysMember{}))
	ws.POST("/savings-goals", h.Create)
	ws.GET("/savings-goals", h.List)
	ws.GET("/savings-goals/:goal_id", h.Get)
	ws.PUT("/savings-goals/:goal_id", h.Update)
	ws.DELETE("/savings-goals/:goal_id", h.Delete)
	ws.POST("/savings-goals/:goal_id/contributions", h.AddContribution)
	ws.GET("/savings-goals/:goal_id/contributions", h.ListContributions)
	ws.DELETE("/savings-goals/:goal_id/contributions/:contribution_id", h.DeleteContribution)

	return r, tokens.AccessToken
}

func TestSavingsHandler_Create(t *testing.T) {
	wsID := uuid.New()

	tests := []struct {
		name           string
		body           map[string]any
		mockFn         func(ctx context.Context, p services.CreateSavingsGoalParams) (models.SavingsGoal, error)
		expectedStatus int
	}{
		{
			name: "success",
			body: map[string]any{"name": "Emergency fund", "target_amount": 5000000},
			mockFn: func(_ context.Context, p services.CreateSavingsGoalParams) (models.SavingsGoal, error) {
				return models.SavingsGoal{ID: uuid.New(), WorkspaceID: p.WorkspaceID, Name: p.Name, TargetAmount: p.TargetAmount}, nil
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing target_amount returns 400",
			body:           map[string]any{"name": "X"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid deadline format returns 400",
			body:           map[string]any{"name": "X", "target_amount": 100, "deadline": "12-31-2026"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "service validation error maps to 400",
			body: map[string]any{"name": "X", "target_amount": 100},
			mockFn: func(_ context.Context, _ services.CreateSavingsGoalParams) (models.SavingsGoal, error) {
				return models.SavingsGoal{}, apperror.ErrInvalidInput
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSavingsServiceForHandler{createFn: tt.mockFn}
			r, token := savingsTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodPost, "/workspaces/"+wsID.String()+"/savings-goals", token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestSavingsHandler_List(t *testing.T) {
	wsID := uuid.New()
	svc := &mockSavingsServiceForHandler{
		listGoalsFn: func(_ context.Context, _ uuid.UUID) ([]services.SavingsGoalProgress, error) {
			return []services.SavingsGoalProgress{{}}, nil
		},
	}
	r, token := savingsTestRouter(t, svc)
	w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/savings-goals", token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSavingsHandler_Get(t *testing.T) {
	wsID := uuid.New()
	goalID := uuid.New()

	tests := []struct {
		name           string
		goalIDInPath   string
		mockFn         func(ctx context.Context, id, workspaceID uuid.UUID) (services.SavingsGoalProgress, error)
		expectedStatus int
	}{
		{
			name:         "success",
			goalIDInPath: goalID.String(),
			mockFn: func(_ context.Context, id, _ uuid.UUID) (services.SavingsGoalProgress, error) {
				return services.SavingsGoalProgress{SavingsGoal: models.SavingsGoal{ID: id}}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid goal_id returns 400",
			goalIDInPath:   "not-a-uuid",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "not found maps to 404",
			goalIDInPath: goalID.String(),
			mockFn: func(_ context.Context, _, _ uuid.UUID) (services.SavingsGoalProgress, error) {
				return services.SavingsGoalProgress{}, apperror.ErrNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSavingsServiceForHandler{getWithProgressFn: tt.mockFn}
			r, token := savingsTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/savings-goals/"+tt.goalIDInPath, token, nil)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestSavingsHandler_Update(t *testing.T) {
	wsID := uuid.New()
	goalID := uuid.New()
	body := map[string]any{"name": "Renamed", "target_amount": 200000}

	svc := &mockSavingsServiceForHandler{
		updateFn: func(_ context.Context, p services.UpdateSavingsGoalParams) (models.SavingsGoal, error) {
			return models.SavingsGoal{ID: p.ID, Name: p.Name}, nil
		},
	}
	r, token := savingsTestRouter(t, svc)
	w := doAuthedRequest(r, http.MethodPut, "/workspaces/"+wsID.String()+"/savings-goals/"+goalID.String(), token, body)
	assert.Equal(t, http.StatusOK, w.Code)

	w = doAuthedRequest(r, http.MethodPut, "/workspaces/"+wsID.String()+"/savings-goals/not-a-uuid", token, body)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSavingsHandler_Delete(t *testing.T) {
	wsID := uuid.New()
	goalID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockSavingsServiceForHandler{deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return nil }}
		r, token := savingsTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/savings-goals/"+goalID.String(), token, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("wrong workspace maps to 404", func(t *testing.T) {
		svc := &mockSavingsServiceForHandler{deleteFn: func(_ context.Context, _, _ uuid.UUID) error { return apperror.ErrNotFound }}
		r, token := savingsTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/savings-goals/"+goalID.String(), token, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// AddContribution moves money into a goal — highest-value coverage for this handler.

func TestSavingsHandler_AddContribution(t *testing.T) {
	wsID := uuid.New()
	goalID := uuid.New()

	tests := []struct {
		name           string
		body           map[string]any
		mockFn         func(ctx context.Context, workspaceID uuid.UUID, p services.AddContributionParams) (models.SavingsContribution, error)
		expectedStatus int
	}{
		{
			name: "success",
			body: map[string]any{"amount": 100000, "contributed_at": "2026-06-01"},
			mockFn: func(_ context.Context, _ uuid.UUID, p services.AddContributionParams) (models.SavingsContribution, error) {
				return models.SavingsContribution{ID: uuid.New(), GoalID: p.GoalID, Amount: p.Amount}, nil
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "missing amount returns 400",
			body:           map[string]any{"contributed_at": "2026-06-01"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "malformed contributed_at returns 400",
			body:           map[string]any{"amount": 100, "contributed_at": "not-a-date"},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "goal not in workspace maps to 404",
			body: map[string]any{"amount": 100, "contributed_at": "2026-06-01"},
			mockFn: func(_ context.Context, _ uuid.UUID, _ services.AddContributionParams) (models.SavingsContribution, error) {
				return models.SavingsContribution{}, apperror.ErrNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &mockSavingsServiceForHandler{addContributionFn: tt.mockFn}
			r, token := savingsTestRouter(t, svc)
			w := doAuthedRequest(r, http.MethodPost, "/workspaces/"+wsID.String()+"/savings-goals/"+goalID.String()+"/contributions", token, tt.body)
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestSavingsHandler_ListContributions(t *testing.T) {
	wsID := uuid.New()
	goalID := uuid.New()
	svc := &mockSavingsServiceForHandler{
		listContributionsFn: func(_ context.Context, _, _ uuid.UUID) ([]models.SavingsContribution, error) {
			return []models.SavingsContribution{{ID: uuid.New()}}, nil
		},
	}
	r, token := savingsTestRouter(t, svc)
	w := doAuthedRequest(r, http.MethodGet, "/workspaces/"+wsID.String()+"/savings-goals/"+goalID.String()+"/contributions", token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestSavingsHandler_DeleteContribution(t *testing.T) {
	wsID := uuid.New()
	goalID := uuid.New()
	contribID := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &mockSavingsServiceForHandler{deleteContributionFn: func(_ context.Context, _, _, _ uuid.UUID) error { return nil }}
		r, token := savingsTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/savings-goals/"+goalID.String()+"/contributions/"+contribID.String(), token, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("invalid contribution_id returns 400", func(t *testing.T) {
		svc := &mockSavingsServiceForHandler{}
		r, token := savingsTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/savings-goals/"+goalID.String()+"/contributions/not-a-uuid", token, nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("contribution belongs to different goal maps to 404", func(t *testing.T) {
		svc := &mockSavingsServiceForHandler{deleteContributionFn: func(_ context.Context, _, _, _ uuid.UUID) error { return apperror.ErrNotFound }}
		r, token := savingsTestRouter(t, svc)
		w := doAuthedRequest(r, http.MethodDelete, "/workspaces/"+wsID.String()+"/savings-goals/"+goalID.String()+"/contributions/"+contribID.String(), token, nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
