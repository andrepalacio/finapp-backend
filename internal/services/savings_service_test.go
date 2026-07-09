package services

import (
	"context"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockSavingsRepo struct {
	createFn             func(ctx context.Context, p repositories.CreateSavingsGoalParams) (models.SavingsGoal, error)
	getByIDFn            func(ctx context.Context, id uuid.UUID) (models.SavingsGoal, error)
	listFn               func(ctx context.Context, workspaceID uuid.UUID) ([]models.SavingsGoal, error)
	listWithProgressFn   func(ctx context.Context, workspaceID uuid.UUID) ([]repositories.SavingsGoalWithProgress, error)
	updateFn             func(ctx context.Context, p repositories.UpdateSavingsGoalParams) (models.SavingsGoal, error)
	deleteFn             func(ctx context.Context, id, workspaceID uuid.UUID) error
	createContributionFn func(ctx context.Context, p repositories.CreateContributionParams) (models.SavingsContribution, error)
	getContributionFn    func(ctx context.Context, id uuid.UUID) (models.SavingsContribution, error)
	listContributionsFn  func(ctx context.Context, goalID uuid.UUID) ([]models.SavingsContribution, error)
	deleteContributionFn func(ctx context.Context, id, goalID uuid.UUID) error
	totalContributedFn   func(ctx context.Context, goalID uuid.UUID) (float64, error)
}

func (m *mockSavingsRepo) Create(ctx context.Context, p repositories.CreateSavingsGoalParams) (models.SavingsGoal, error) {
	return m.createFn(ctx, p)
}
func (m *mockSavingsRepo) GetByID(ctx context.Context, id uuid.UUID) (models.SavingsGoal, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockSavingsRepo) List(ctx context.Context, workspaceID uuid.UUID) ([]models.SavingsGoal, error) {
	return m.listFn(ctx, workspaceID)
}
func (m *mockSavingsRepo) ListWithProgress(ctx context.Context, workspaceID uuid.UUID) ([]repositories.SavingsGoalWithProgress, error) {
	return m.listWithProgressFn(ctx, workspaceID)
}
func (m *mockSavingsRepo) Update(ctx context.Context, p repositories.UpdateSavingsGoalParams) (models.SavingsGoal, error) {
	return m.updateFn(ctx, p)
}
func (m *mockSavingsRepo) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return m.deleteFn(ctx, id, workspaceID)
}
func (m *mockSavingsRepo) CreateContribution(ctx context.Context, p repositories.CreateContributionParams) (models.SavingsContribution, error) {
	return m.createContributionFn(ctx, p)
}
func (m *mockSavingsRepo) GetContribution(ctx context.Context, id uuid.UUID) (models.SavingsContribution, error) {
	return m.getContributionFn(ctx, id)
}
func (m *mockSavingsRepo) ListContributions(ctx context.Context, goalID uuid.UUID) ([]models.SavingsContribution, error) {
	return m.listContributionsFn(ctx, goalID)
}
func (m *mockSavingsRepo) DeleteContribution(ctx context.Context, id, goalID uuid.UUID) error {
	return m.deleteContributionFn(ctx, id, goalID)
}
func (m *mockSavingsRepo) TotalContributed(ctx context.Context, goalID uuid.UUID) (float64, error) {
	return m.totalContributedFn(ctx, goalID)
}

func makeSavingsGoal(wsID uuid.UUID) models.SavingsGoal {
	return models.SavingsGoal{
		ID:           uuid.New(),
		WorkspaceID:  wsID,
		Name:         "Emergency fund",
		TargetAmount: 5000000,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
}

func TestSavingsService_Create_Validation(t *testing.T) {
	wsID := uuid.New()

	tests := []struct {
		name    string
		params  CreateSavingsGoalParams
		wantErr bool
		errType error
	}{
		{
			name:   "success",
			params: CreateSavingsGoalParams{WorkspaceID: wsID, Name: "Emergency fund", TargetAmount: 5000000},
		},
		{
			name:    "empty name",
			params:  CreateSavingsGoalParams{WorkspaceID: wsID, TargetAmount: 1000},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
		{
			name:    "zero target",
			params:  CreateSavingsGoalParams{WorkspaceID: wsID, Name: "X", TargetAmount: 0},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
		{
			name:    "negative target",
			params:  CreateSavingsGoalParams{WorkspaceID: wsID, Name: "X", TargetAmount: -100},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockSavingsRepo{
				createFn: func(_ context.Context, p repositories.CreateSavingsGoalParams) (models.SavingsGoal, error) {
					g := makeSavingsGoal(p.WorkspaceID)
					g.Name = p.Name
					return g, nil
				},
			}
			svc := NewSavingsService(repo)
			_, err := svc.Create(context.Background(), tt.params)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestSavingsService_ListGoals(t *testing.T) {
	wsID := uuid.New()
	rows := []repositories.SavingsGoalWithProgress{
		{SavingsGoal: models.SavingsGoal{ID: uuid.New(), WorkspaceID: wsID, Name: "A", TargetAmount: 1000}, TotalContributed: 250},
		{SavingsGoal: models.SavingsGoal{ID: uuid.New(), WorkspaceID: wsID, Name: "B", TargetAmount: 0}, TotalContributed: 0},
	}

	repo := &mockSavingsRepo{
		listWithProgressFn: func(_ context.Context, _ uuid.UUID) ([]repositories.SavingsGoalWithProgress, error) { return rows, nil },
	}
	svc := NewSavingsService(repo)

	got, err := svc.ListGoals(context.Background(), wsID)
	assert.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, float64(25), got[0].ProgressPct)
	assert.Equal(t, float64(750), got[0].Remaining)
	assert.Equal(t, float64(0), got[1].ProgressPct)
}

func TestSavingsService_ListGoals_RepoError(t *testing.T) {
	repo := &mockSavingsRepo{
		listWithProgressFn: func(_ context.Context, _ uuid.UUID) ([]repositories.SavingsGoalWithProgress, error) {
			return nil, apperror.ErrInternal
		},
	}
	svc := NewSavingsService(repo)

	_, err := svc.ListGoals(context.Background(), uuid.New())
	assert.ErrorIs(t, err, apperror.ErrInternal)
}

func TestSavingsService_Update(t *testing.T) {
	wsID := uuid.New()

	tests := []struct {
		name    string
		params  UpdateSavingsGoalParams
		wantErr bool
		errType error
	}{
		{
			name:   "success",
			params: UpdateSavingsGoalParams{ID: uuid.New(), WorkspaceID: wsID, Name: "Renamed", TargetAmount: 200000},
		},
		{
			name:    "empty name",
			params:  UpdateSavingsGoalParams{ID: uuid.New(), WorkspaceID: wsID, TargetAmount: 200000},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
		{
			name:    "zero target",
			params:  UpdateSavingsGoalParams{ID: uuid.New(), WorkspaceID: wsID, Name: "X", TargetAmount: 0},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockSavingsRepo{
				updateFn: func(_ context.Context, p repositories.UpdateSavingsGoalParams) (models.SavingsGoal, error) {
					g := makeSavingsGoal(p.WorkspaceID)
					g.ID = p.ID
					g.Name = p.Name
					g.TargetAmount = p.TargetAmount
					return g, nil
				},
			}
			svc := NewSavingsService(repo)
			got, err := svc.Update(context.Background(), tt.params)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.params.Name, got.Name)
		})
	}
}

func TestSavingsService_Delete(t *testing.T) {
	wsID := uuid.New()
	otherWS := uuid.New()
	goal := makeSavingsGoal(wsID)

	t.Run("success", func(t *testing.T) {
		deleteCalled := false
		repo := &mockSavingsRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (models.SavingsGoal, error) { return goal, nil },
			deleteFn:  func(_ context.Context, _, _ uuid.UUID) error { deleteCalled = true; return nil },
		}
		svc := NewSavingsService(repo)
		err := svc.Delete(context.Background(), goal.ID, wsID)
		assert.NoError(t, err)
		assert.True(t, deleteCalled)
	})

	t.Run("wrong workspace", func(t *testing.T) {
		repo := &mockSavingsRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (models.SavingsGoal, error) { return goal, nil },
		}
		svc := NewSavingsService(repo)
		err := svc.Delete(context.Background(), goal.ID, otherWS)
		assert.ErrorIs(t, err, apperror.ErrNotFound)
	})
}

func TestSavingsService_ListContributions(t *testing.T) {
	wsID := uuid.New()
	goal := makeSavingsGoal(wsID)
	contribs := []models.SavingsContribution{{ID: uuid.New(), GoalID: goal.ID, Amount: 500}}

	t.Run("success", func(t *testing.T) {
		repo := &mockSavingsRepo{
			getByIDFn:           func(_ context.Context, _ uuid.UUID) (models.SavingsGoal, error) { return goal, nil },
			listContributionsFn: func(_ context.Context, _ uuid.UUID) ([]models.SavingsContribution, error) { return contribs, nil },
		}
		svc := NewSavingsService(repo)
		got, err := svc.ListContributions(context.Background(), goal.ID, wsID)
		assert.NoError(t, err)
		assert.Len(t, got, 1)
	})

	t.Run("wrong workspace", func(t *testing.T) {
		repo := &mockSavingsRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (models.SavingsGoal, error) { return goal, nil },
		}
		svc := NewSavingsService(repo)
		_, err := svc.ListContributions(context.Background(), goal.ID, uuid.New())
		assert.ErrorIs(t, err, apperror.ErrNotFound)
	})
}

func TestSavingsService_GetWithProgress(t *testing.T) {
	wsID := uuid.New()
	goal := makeSavingsGoal(wsID)

	repo := &mockSavingsRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.SavingsGoal, error) { return goal, nil },
		totalContributedFn: func(_ context.Context, _ uuid.UUID) (float64, error) { return 2000000, nil },
	}
	svc := NewSavingsService(repo)

	prog, err := svc.GetWithProgress(context.Background(), goal.ID, wsID)
	assert.NoError(t, err)
	assert.Equal(t, float64(5000000), prog.TargetAmount)
	assert.Equal(t, float64(2000000), prog.TotalContributed)
	assert.Equal(t, float64(3000000), prog.Remaining)
	assert.Equal(t, float64(40), prog.ProgressPct)
}

func TestSavingsService_GetWithProgress_WrongWorkspace(t *testing.T) {
	wsID := uuid.New()
	otherWS := uuid.New()
	goal := makeSavingsGoal(wsID)

	repo := &mockSavingsRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.SavingsGoal, error) { return goal, nil },
	}
	svc := NewSavingsService(repo)

	_, err := svc.GetWithProgress(context.Background(), goal.ID, otherWS)
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestSavingsService_AddContribution_ZeroAmount(t *testing.T) {
	wsID := uuid.New()
	goal := makeSavingsGoal(wsID)

	repo := &mockSavingsRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.SavingsGoal, error) { return goal, nil },
	}
	svc := NewSavingsService(repo)

	_, err := svc.AddContribution(context.Background(), wsID, AddContributionParams{
		GoalID: goal.ID, Amount: 0, ContributedAt: time.Now(),
	})
	assert.ErrorIs(t, err, apperror.ErrInvalidInput)
}

func TestSavingsService_DeleteContribution_WrongGoal(t *testing.T) {
	wsID := uuid.New()
	goal := makeSavingsGoal(wsID)
	otherGoalID := uuid.New()
	contribID := uuid.New()
	contrib := models.SavingsContribution{ID: contribID, GoalID: otherGoalID, Amount: 100}

	repo := &mockSavingsRepo{
		getByIDFn:         func(_ context.Context, _ uuid.UUID) (models.SavingsGoal, error) { return goal, nil },
		getContributionFn: func(_ context.Context, _ uuid.UUID) (models.SavingsContribution, error) { return contrib, nil },
	}
	svc := NewSavingsService(repo)

	err := svc.DeleteContribution(context.Background(), contribID, goal.ID, wsID)
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}
