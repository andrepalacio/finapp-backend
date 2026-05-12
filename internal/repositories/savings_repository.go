package repositories

import (
	"context"
	"errors"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories/sqlc"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SavingsRepository struct {
	q *sqlc.Queries
}

func NewSavingsRepository(pool *pgxpool.Pool) *SavingsRepository {
	return &SavingsRepository{q: sqlc.New(pool)}
}

type CreateSavingsGoalParams struct {
	WorkspaceID  uuid.UUID
	Name         string
	TargetAmount float64
	Deadline     *time.Time
	Notes        string
}

func (r *SavingsRepository) Create(ctx context.Context, p CreateSavingsGoalParams) (models.SavingsGoal, error) {
	row, err := r.q.CreateSavingsGoal(ctx, sqlc.CreateSavingsGoalParams{
		WorkspaceID:  p.WorkspaceID,
		Name:         p.Name,
		TargetAmount: p.TargetAmount,
		Deadline:     toPgDatePtr(p.Deadline),
		Notes:        toPgText(p.Notes),
	})
	if err != nil {
		return models.SavingsGoal{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toSavingsGoalModel(row), nil
}

func (r *SavingsRepository) GetByID(ctx context.Context, id uuid.UUID) (models.SavingsGoal, error) {
	row, err := r.q.GetSavingsGoalByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.SavingsGoal{}, apperror.ErrNotFound
		}
		return models.SavingsGoal{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toSavingsGoalModel(row), nil
}

func (r *SavingsRepository) List(ctx context.Context, workspaceID uuid.UUID) ([]models.SavingsGoal, error) {
	rows, err := r.q.ListSavingsGoals(ctx, workspaceID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.SavingsGoal, len(rows))
	for i, row := range rows {
		out[i] = toSavingsGoalModel(row)
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

func (r *SavingsRepository) Update(ctx context.Context, p UpdateSavingsGoalParams) (models.SavingsGoal, error) {
	row, err := r.q.UpdateSavingsGoal(ctx, sqlc.UpdateSavingsGoalParams{
		ID:           p.ID,
		WorkspaceID:  p.WorkspaceID,
		Name:         p.Name,
		TargetAmount: p.TargetAmount,
		Deadline:     toPgDatePtr(p.Deadline),
		Notes:        toPgText(p.Notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.SavingsGoal{}, apperror.ErrNotFound
		}
		return models.SavingsGoal{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toSavingsGoalModel(row), nil
}

func (r *SavingsRepository) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return r.q.DeleteSavingsGoal(ctx, sqlc.DeleteSavingsGoalParams{ID: id, WorkspaceID: workspaceID})
}

type CreateContributionParams struct {
	GoalID        uuid.UUID
	Amount        float64
	ContributedAt time.Time
	Notes         string
}

func (r *SavingsRepository) CreateContribution(ctx context.Context, p CreateContributionParams) (models.SavingsContribution, error) {
	row, err := r.q.CreateContribution(ctx, sqlc.CreateContributionParams{
		GoalID:        p.GoalID,
		Amount:        p.Amount,
		ContributedAt: toPgDate(p.ContributedAt),
		Notes:         toPgText(p.Notes),
	})
	if err != nil {
		return models.SavingsContribution{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toContributionModel(row), nil
}

func (r *SavingsRepository) GetContribution(ctx context.Context, id uuid.UUID) (models.SavingsContribution, error) {
	row, err := r.q.GetContribution(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.SavingsContribution{}, apperror.ErrNotFound
		}
		return models.SavingsContribution{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toContributionModel(row), nil
}

func (r *SavingsRepository) ListContributions(ctx context.Context, goalID uuid.UUID) ([]models.SavingsContribution, error) {
	rows, err := r.q.ListContributions(ctx, goalID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.SavingsContribution, len(rows))
	for i, row := range rows {
		out[i] = toContributionModel(row)
	}
	return out, nil
}

func (r *SavingsRepository) DeleteContribution(ctx context.Context, id, goalID uuid.UUID) error {
	return r.q.DeleteContribution(ctx, sqlc.DeleteContributionParams{ID: id, GoalID: goalID})
}

func (r *SavingsRepository) TotalContributed(ctx context.Context, goalID uuid.UUID) (float64, error) {
	total, err := r.q.GetTotalContributed(ctx, goalID)
	if err != nil {
		return 0, apperror.Wrap(apperror.ErrInternal, err)
	}
	return total, nil
}

func toSavingsGoalModel(row sqlc.SavingsGoal) models.SavingsGoal {
	return models.SavingsGoal{
		ID:           row.ID,
		WorkspaceID:  row.WorkspaceID,
		Name:         row.Name,
		TargetAmount: row.TargetAmount,
		Deadline:     fromPgDatePtr(row.Deadline),
		Notes:        fromPgText(row.Notes),
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}

func toContributionModel(row sqlc.SavingsContribution) models.SavingsContribution {
	return models.SavingsContribution{
		ID:            row.ID,
		GoalID:        row.GoalID,
		Amount:        row.Amount,
		ContributedAt: fromPgDate(row.ContributedAt),
		Notes:         fromPgText(row.Notes),
		CreatedAt:     row.CreatedAt.Time,
	}
}
