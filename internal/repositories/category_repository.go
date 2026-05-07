package repositories

import (
	"context"
	"errors"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories/sqlc"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository struct {
	q *sqlc.Queries
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{q: sqlc.New(pool)}
}

type CreateCategoryParams struct {
	WorkspaceID uuid.UUID
	Name        string
	Icon        string
	Color       string
	Type        string
}

func (r *CategoryRepository) Create(ctx context.Context, p CreateCategoryParams) (models.Category, error) {
	wid := &p.WorkspaceID
	row, err := r.q.CreateCategory(ctx, sqlc.CreateCategoryParams{
		WorkspaceID: wid,
		Name:        p.Name,
		Icon:        toPgText(p.Icon),
		Color:       toPgText(p.Color),
		Type:        p.Type,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Category{}, apperror.ErrConflict
		}
		return models.Category{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toCategoryModel(row), nil
}

func (r *CategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Category, error) {
	row, err := r.q.GetCategoryByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Category{}, apperror.ErrNotFound
		}
		return models.Category{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toCategoryModel(row), nil
}

func (r *CategoryRepository) ListForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Category, error) {
	rows, err := r.q.ListCategoriesForWorkspace(ctx, &workspaceID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.Category, len(rows))
	for i, row := range rows {
		out[i] = toCategoryModel(row)
	}
	return out, nil
}

type UpdateCategoryParams struct {
	ID    uuid.UUID
	Name  string
	Icon  string
	Color string
	Type  string
}

func (r *CategoryRepository) Update(ctx context.Context, p UpdateCategoryParams) (models.Category, error) {
	row, err := r.q.UpdateCategory(ctx, sqlc.UpdateCategoryParams{
		ID:    p.ID,
		Name:  p.Name,
		Icon:  toPgText(p.Icon),
		Color: toPgText(p.Color),
		Type:  p.Type,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Category{}, apperror.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.Category{}, apperror.ErrConflict
		}
		return models.Category{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toCategoryModel(row), nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	err := r.q.DeleteCategory(ctx, sqlc.DeleteCategoryParams{
		ID:          id,
		WorkspaceID: &workspaceID,
	})
	if err != nil {
		return apperror.Wrap(apperror.ErrInternal, err)
	}
	return nil
}

func toCategoryModel(row sqlc.Category) models.Category {
	return models.Category{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		Name:        row.Name,
		Icon:        fromPgText(row.Icon),
		Color:       fromPgText(row.Color),
		Type:        row.Type,
		IsSystem:    row.IsSystem,
		CreatedAt:   row.CreatedAt.Time,
	}
}
