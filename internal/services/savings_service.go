package services

import (
	"context"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
)

type SavingsRepository interface {
	Create(ctx context.Context, p repositories.CreateSavingsGoalParams) (models.SavingsGoal, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.SavingsGoal, error)
	List(ctx context.Context, workspaceID uuid.UUID) ([]models.SavingsGoal, error)
	ListWithProgress(ctx context.Context, workspaceID uuid.UUID) ([]repositories.SavingsGoalWithProgress, error)
	Update(ctx context.Context, p repositories.UpdateSavingsGoalParams) (models.SavingsGoal, error)
	Delete(ctx context.Context, id, workspaceID uuid.UUID) error
	CreateContribution(ctx context.Context, p repositories.CreateContributionParams) (models.SavingsContribution, error)
	GetContribution(ctx context.Context, id uuid.UUID) (models.SavingsContribution, error)
	ListContributions(ctx context.Context, goalID uuid.UUID) ([]models.SavingsContribution, error)
	DeleteContribution(ctx context.Context, id, goalID uuid.UUID) error
	TotalContributed(ctx context.Context, goalID uuid.UUID) (float64, error)
}

type SavingsService struct {
	repo SavingsRepository
}

func NewSavingsService(repo SavingsRepository) *SavingsService {
	return &SavingsService{repo: repo}
}

type SavingsGoalProgress struct {
	models.SavingsGoal
	TotalContributed float64 `json:"total_contributed"`
	Remaining        float64 `json:"remaining"`
	ProgressPct      float64 `json:"progress_pct"`
}

type CreateSavingsGoalParams struct {
	WorkspaceID  uuid.UUID
	Name         string
	TargetAmount float64
	Deadline     *time.Time
	Notes        string
}

func (s *SavingsService) Create(ctx context.Context, p CreateSavingsGoalParams) (models.SavingsGoal, error) {
	if p.Name == "" {
		return models.SavingsGoal{}, apperror.WithMessage(apperror.ErrInvalidInput, "name is required")
	}
	if p.TargetAmount <= 0 {
		return models.SavingsGoal{}, apperror.WithMessage(apperror.ErrInvalidInput, "target_amount must be positive")
	}
	return s.repo.Create(ctx, repositories.CreateSavingsGoalParams{
		WorkspaceID:  p.WorkspaceID,
		Name:         p.Name,
		TargetAmount: p.TargetAmount,
		Deadline:     p.Deadline,
		Notes:        p.Notes,
	})
}

func (s *SavingsService) GetByID(ctx context.Context, id, workspaceID uuid.UUID) (models.SavingsGoal, error) {
	goal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return models.SavingsGoal{}, err
	}
	if goal.WorkspaceID != workspaceID {
		return models.SavingsGoal{}, apperror.ErrNotFound
	}
	return goal, nil
}

func (s *SavingsService) ListGoals(ctx context.Context, workspaceID uuid.UUID) ([]SavingsGoalProgress, error) {
	rows, err := s.repo.ListWithProgress(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]SavingsGoalProgress, len(rows))
	for i, row := range rows {
		var pct float64
		if row.TargetAmount > 0 {
			pct = round2(row.TotalContributed / row.TargetAmount * 100)
		}
		out[i] = SavingsGoalProgress{
			SavingsGoal:      row.SavingsGoal,
			TotalContributed: round2(row.TotalContributed),
			Remaining:        round2(row.TargetAmount - row.TotalContributed),
			ProgressPct:      pct,
		}
	}
	return out, nil
}

type UpdateSavingsGoalParams struct {
	ID           uuid.UUID
	WorkspaceID  uuid.UUID
	Name         string
	TargetAmount float64
	Deadline     *time.Time
	Notes        string
}

func (s *SavingsService) Update(ctx context.Context, p UpdateSavingsGoalParams) (models.SavingsGoal, error) {
	if p.Name == "" {
		return models.SavingsGoal{}, apperror.WithMessage(apperror.ErrInvalidInput, "name is required")
	}
	if p.TargetAmount <= 0 {
		return models.SavingsGoal{}, apperror.WithMessage(apperror.ErrInvalidInput, "target_amount must be positive")
	}
	return s.repo.Update(ctx, repositories.UpdateSavingsGoalParams{
		ID:           p.ID,
		WorkspaceID:  p.WorkspaceID,
		Name:         p.Name,
		TargetAmount: p.TargetAmount,
		Deadline:     p.Deadline,
		Notes:        p.Notes,
	})
}

func (s *SavingsService) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	goal, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if goal.WorkspaceID != workspaceID {
		return apperror.ErrNotFound
	}
	return s.repo.Delete(ctx, id, workspaceID)
}

func (s *SavingsService) GetWithProgress(ctx context.Context, id, workspaceID uuid.UUID) (SavingsGoalProgress, error) {
	goal, err := s.GetByID(ctx, id, workspaceID)
	if err != nil {
		return SavingsGoalProgress{}, err
	}
	total, err := s.repo.TotalContributed(ctx, id)
	if err != nil {
		return SavingsGoalProgress{}, err
	}
	remaining := goal.TargetAmount - total
	var pct float64
	if goal.TargetAmount > 0 {
		pct = round2(total / goal.TargetAmount * 100)
	}
	return SavingsGoalProgress{
		SavingsGoal:      goal,
		TotalContributed: round2(total),
		Remaining:        round2(remaining),
		ProgressPct:      pct,
	}, nil
}

type AddContributionParams struct {
	GoalID        uuid.UUID
	Amount        float64
	ContributedAt time.Time
	Notes         string
}

func (s *SavingsService) AddContribution(ctx context.Context, workspaceID uuid.UUID, p AddContributionParams) (models.SavingsContribution, error) {
	if _, err := s.GetByID(ctx, p.GoalID, workspaceID); err != nil {
		return models.SavingsContribution{}, err
	}
	if p.Amount <= 0 {
		return models.SavingsContribution{}, apperror.WithMessage(apperror.ErrInvalidInput, "amount must be positive")
	}
	if p.ContributedAt.IsZero() {
		return models.SavingsContribution{}, apperror.WithMessage(apperror.ErrInvalidInput, "contributed_at is required")
	}
	return s.repo.CreateContribution(ctx, repositories.CreateContributionParams{
		GoalID:        p.GoalID,
		Amount:        p.Amount,
		ContributedAt: p.ContributedAt,
		Notes:         p.Notes,
	})
}

func (s *SavingsService) ListContributions(ctx context.Context, goalID, workspaceID uuid.UUID) ([]models.SavingsContribution, error) {
	if _, err := s.GetByID(ctx, goalID, workspaceID); err != nil {
		return nil, err
	}
	return s.repo.ListContributions(ctx, goalID)
}

func (s *SavingsService) DeleteContribution(ctx context.Context, contribID, goalID, workspaceID uuid.UUID) error {
	if _, err := s.GetByID(ctx, goalID, workspaceID); err != nil {
		return err
	}
	contrib, err := s.repo.GetContribution(ctx, contribID)
	if err != nil {
		return err
	}
	if contrib.GoalID != goalID {
		return apperror.ErrNotFound
	}
	return s.repo.DeleteContribution(ctx, contribID, goalID)
}
