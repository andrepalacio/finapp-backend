package repositories

import (
	"context"
	"errors"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories/sqlc"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BudgetRepository struct {
	q *sqlc.Queries
}

func NewBudgetRepository(pool *pgxpool.Pool) *BudgetRepository {
	return &BudgetRepository{q: sqlc.New(pool)}
}

type UpsertBudgetParams struct {
	WorkspaceID uuid.UUID
	Year        int16
	Month       int16
	TotalLimit  float64
}

func (r *BudgetRepository) Upsert(ctx context.Context, p UpsertBudgetParams) (models.Budget, error) {
	row, err := r.q.UpsertBudget(ctx, sqlc.UpsertBudgetParams{
		WorkspaceID: p.WorkspaceID,
		Year:        p.Year,
		Month:       p.Month,
		TotalLimit:  p.TotalLimit,
	})
	if err != nil {
		return models.Budget{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toBudgetModel(row), nil
}

func (r *BudgetRepository) GetByYearMonth(ctx context.Context, workspaceID uuid.UUID, year, month int16) (models.Budget, error) {
	row, err := r.q.GetBudgetByYearMonth(ctx, sqlc.GetBudgetByYearMonthParams{
		WorkspaceID: workspaceID,
		Year:        year,
		Month:       month,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Budget{}, apperror.ErrNotFound
		}
		return models.Budget{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toBudgetModel(row), nil
}

func (r *BudgetRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Budget, error) {
	row, err := r.q.GetBudgetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Budget{}, apperror.ErrNotFound
		}
		return models.Budget{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toBudgetModel(row), nil
}

func (r *BudgetRepository) List(ctx context.Context, workspaceID uuid.UUID) ([]models.Budget, error) {
	rows, err := r.q.ListBudgets(ctx, workspaceID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.Budget, len(rows))
	for i, row := range rows {
		out[i] = toBudgetModel(row)
	}
	return out, nil
}

func (r *BudgetRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteBudget(ctx, id)
}

func (r *BudgetRepository) UpsertCategory(ctx context.Context, budgetID, categoryID uuid.UUID, limit float64) error {
	return r.q.UpsertBudgetCategory(ctx, sqlc.UpsertBudgetCategoryParams{
		BudgetID:    budgetID,
		CategoryID:  categoryID,
		LimitAmount: limit,
	})
}

func (r *BudgetRepository) DeleteCategory(ctx context.Context, budgetID, categoryID uuid.UUID) error {
	return r.q.DeleteBudgetCategory(ctx, sqlc.DeleteBudgetCategoryParams{
		BudgetID:   budgetID,
		CategoryID: categoryID,
	})
}

func (r *BudgetRepository) ListCategories(ctx context.Context, budgetID uuid.UUID) ([]models.BudgetCategory, error) {
	rows, err := r.q.ListBudgetCategories(ctx, budgetID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.BudgetCategory, len(rows))
	for i, row := range rows {
		out[i] = models.BudgetCategory{
			BudgetID:     row.BudgetID,
			CategoryID:   row.CategoryID,
			CategoryName: row.CategoryName,
			CategoryIcon: fromPgText(row.CategoryIcon),
			LimitAmount:  row.LimitAmount,
		}
	}
	return out, nil
}

func (r *BudgetRepository) CategorySpending(ctx context.Context, budgetID, workspaceID uuid.UUID, year, month int32) ([]models.BudgetCategorySpending, error) {
	rows, err := r.q.GetBudgetCategorySpending(ctx, sqlc.GetBudgetCategorySpendingParams{
		BudgetID:    budgetID,
		WorkspaceID: workspaceID,
		Column3:     year,
		Column4:     month,
	})
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.BudgetCategorySpending, len(rows))
	for i, row := range rows {
		out[i] = models.BudgetCategorySpending{
			CategoryID:  row.CategoryID,
			LimitAmount: row.LimitAmount,
			Spent:       row.Spent,
		}
	}
	return out, nil
}

func toBudgetModel(row sqlc.Budget) models.Budget {
	return models.Budget{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		Year:        row.Year,
		Month:       row.Month,
		TotalLimit:  row.TotalLimit,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}
}
