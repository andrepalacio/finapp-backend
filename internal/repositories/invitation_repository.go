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
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type InvitationRepository struct {
	q *sqlc.Queries
}

func NewInvitationRepository(pool *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{q: sqlc.New(pool)}
}

func (r *InvitationRepository) Create(ctx context.Context, workspaceID uuid.UUID, email, role string, invitedBy uuid.UUID, expiresAt time.Time) (models.WorkspaceInvitation, error) {
	row, err := r.q.CreateInvitation(ctx, sqlc.CreateInvitationParams{
		WorkspaceID: workspaceID,
		Email:       email,
		Role:        role,
		InvitedBy:   invitedBy,
		ExpiresAt:   pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return models.WorkspaceInvitation{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toInvitationModel(row), nil
}

func (r *InvitationRepository) GetByToken(ctx context.Context, token uuid.UUID) (models.WorkspaceInvitation, error) {
	row, err := r.q.GetInvitationByToken(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.WorkspaceInvitation{}, apperror.ErrNotFound
		}
		return models.WorkspaceInvitation{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toInvitationModel(row), nil
}

func (r *InvitationRepository) GetByID(ctx context.Context, id uuid.UUID) (models.WorkspaceInvitation, error) {
	row, err := r.q.GetInvitationByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.WorkspaceInvitation{}, apperror.ErrNotFound
		}
		return models.WorkspaceInvitation{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toInvitationModel(row), nil
}

func (r *InvitationRepository) ListPending(ctx context.Context, workspaceID uuid.UUID) ([]models.WorkspaceInvitation, error) {
	rows, err := r.q.ListPendingInvitations(ctx, workspaceID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.WorkspaceInvitation, len(rows))
	for i, row := range rows {
		out[i] = toInvitationModel(row)
	}
	return out, nil
}

func (r *InvitationRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (models.WorkspaceInvitation, error) {
	row, err := r.q.UpdateInvitationStatus(ctx, sqlc.UpdateInvitationStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.WorkspaceInvitation{}, apperror.ErrNotFound
		}
		return models.WorkspaceInvitation{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toInvitationModel(row), nil
}

func toInvitationModel(row sqlc.WorkspaceInvitation) models.WorkspaceInvitation {
	return models.WorkspaceInvitation{
		ID:          row.ID,
		WorkspaceID: row.WorkspaceID,
		Email:       row.Email,
		Role:        row.Role,
		Token:       row.Token,
		Status:      row.Status,
		InvitedBy:   row.InvitedBy,
		ExpiresAt:   row.ExpiresAt.Time,
		CreatedAt:   row.CreatedAt.Time,
	}
}
