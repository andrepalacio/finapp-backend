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

type UserRepository struct {
	q *sqlc.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{q: sqlc.New(pool)}
}

type CreateUserParams struct {
	Email        string
	PasswordHash string
	Name         string
}

func (r *UserRepository) Create(ctx context.Context, params CreateUserParams) (models.User, error) {
	row, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		Email:        params.Email,
		PasswordHash: params.PasswordHash,
		Name:         params.Name,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.User{}, apperror.ErrConflict
		}
		return models.User{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toUserModel(row), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, apperror.ErrNotFound
		}
		return models.User{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toUserModel(row), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, apperror.ErrNotFound
		}
		return models.User{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toUserModel(row), nil
}

func (r *UserRepository) Update(ctx context.Context, userID uuid.UUID, name, email string) (models.User, error) {
	row, err := r.q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:    userID,
		Name:  name,
		Email: email,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.User{}, apperror.ErrConflict
		}
		return models.User{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toUserModel(row), nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) (models.User, error) {
	row, err := r.q.UpdateUserPassword(ctx, sqlc.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return models.User{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toUserModel(row), nil
}

func (r *UserRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	err := r.q.DeleteUser(ctx, userID)
	if err != nil {
		return apperror.Wrap(apperror.ErrInternal, err)
	}
	return nil
}

func toUserModel(row sqlc.User) models.User {
	return models.User{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Name:         row.Name,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}
}
