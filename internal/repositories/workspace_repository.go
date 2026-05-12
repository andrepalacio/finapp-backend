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

type WorkspaceRepository struct {
	q *sqlc.Queries
}

func NewWorkspaceRepository(pool *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{q: sqlc.New(pool)}
}

type CreateWorkspaceParams struct {
	Name     string
	OwnerID  uuid.UUID
	Currency string
}

func (r *WorkspaceRepository) Create(ctx context.Context, p CreateWorkspaceParams) (models.Workspace, error) {
	row, err := r.q.CreateWorkspace(ctx, sqlc.CreateWorkspaceParams{
		Name:     p.Name,
		OwnerID:  p.OwnerID,
		Currency: p.Currency,
	})
	if err != nil {
		return models.Workspace{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toWorkspaceModel(row), nil
}

func (r *WorkspaceRepository) AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role string) error {
	err := r.q.AddWorkspaceMember(ctx, sqlc.AddWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
	})
	if err != nil {
		return apperror.Wrap(apperror.ErrInternal, err)
	}
	return nil
}

func (r *WorkspaceRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Workspace, error) {
	row, err := r.q.GetWorkspaceByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Workspace{}, apperror.ErrNotFound
		}
		return models.Workspace{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toWorkspaceModel(row), nil
}

func (r *WorkspaceRepository) GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (models.WorkspaceMember, error) {
	row, err := r.q.GetWorkspaceMember(ctx, sqlc.GetWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.WorkspaceMember{}, apperror.ErrNotFound
		}
		return models.WorkspaceMember{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return models.WorkspaceMember{
		WorkspaceID: row.WorkspaceID,
		UserID:      row.UserID,
		Role:        row.Role,
		JoinedAt:    row.JoinedAt.Time,
	}, nil
}

func (r *WorkspaceRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Workspace, error) {
	rows, err := r.q.ListWorkspacesByUser(ctx, userID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.Workspace, len(rows))
	for i, row := range rows {
		out[i] = toWorkspaceModel(row)
	}
	return out, nil
}

func (r *WorkspaceRepository) Update(ctx context.Context, id uuid.UUID, name, currency string) (models.Workspace, error) {
	row, err := r.q.UpdateWorkspace(ctx, sqlc.UpdateWorkspaceParams{
		ID:       id,
		Name:     name,
		Currency: currency,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Workspace{}, apperror.ErrNotFound
		}
		return models.Workspace{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toWorkspaceModel(row), nil
}

func (r *WorkspaceRepository) IsMember(ctx context.Context, workspaceID, userID uuid.UUID) bool {
	_, err := r.GetMember(ctx, workspaceID, userID)
	return err == nil
}

func (r *WorkspaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteWorkspace(ctx, id)
}

func (r *WorkspaceRepository) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]models.WorkspaceMemberWithUser, error) {
	rows, err := r.q.ListWorkspaceMembers(ctx, workspaceID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.WorkspaceMemberWithUser, len(rows))
	for i, row := range rows {
		out[i] = models.WorkspaceMemberWithUser{
			WorkspaceID: row.WorkspaceID,
			UserID:      row.UserID,
			Role:        row.Role,
			JoinedAt:    row.JoinedAt.Time,
			Name:        row.Name,
			Email:       row.Email,
		}
	}
	return out, nil
}

func (r *WorkspaceRepository) RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error {
	err := r.q.RemoveWorkspaceMember(ctx, sqlc.RemoveWorkspaceMemberParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	if err != nil {
		return apperror.Wrap(apperror.ErrInternal, err)
	}
	return nil
}

func (r *WorkspaceRepository) UpdateMemberRole(ctx context.Context, workspaceID, userID uuid.UUID, role string) error {
	err := r.q.UpdateMemberRole(ctx, sqlc.UpdateMemberRoleParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
	})
	if err != nil {
		return apperror.Wrap(apperror.ErrInternal, err)
	}
	return nil
}

func toWorkspaceModel(row sqlc.Workspace) models.Workspace {
	return models.Workspace{
		ID:        row.ID,
		Name:      row.Name,
		OwnerID:   row.OwnerID,
		Currency:  row.Currency,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
