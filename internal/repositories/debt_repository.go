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
	"time"
)

type DebtRepository struct {
	q *sqlc.Queries
}

func NewDebtRepository(pool *pgxpool.Pool) *DebtRepository {
	return &DebtRepository{q: sqlc.New(pool)}
}

type CreateDebtParams struct {
	WorkspaceID      uuid.UUID
	Name             string
	Lender           string
	Principal        float64
	Rate             float64
	RateType         string
	Installments     int32
	FirstPaymentDate time.Time
	Notes            string
}

func (r *DebtRepository) Create(ctx context.Context, p CreateDebtParams) (models.Debt, error) {
	row, err := r.q.CreateDebt(ctx, sqlc.CreateDebtParams{
		WorkspaceID:      p.WorkspaceID,
		Name:             p.Name,
		Lender:           toPgText(p.Lender),
		Principal:        p.Principal,
		Rate:             p.Rate,
		RateType:         p.RateType,
		Installments:     p.Installments,
		FirstPaymentDate: toPgDate(p.FirstPaymentDate),
		Notes:            toPgText(p.Notes),
	})
	if err != nil {
		return models.Debt{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toDebtModel(row), nil
}

func (r *DebtRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Debt, error) {
	row, err := r.q.GetDebtByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Debt{}, apperror.ErrNotFound
		}
		return models.Debt{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toDebtModel(row), nil
}

func (r *DebtRepository) List(ctx context.Context, workspaceID uuid.UUID) ([]models.Debt, error) {
	rows, err := r.q.ListDebts(ctx, workspaceID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.Debt, len(rows))
	for i, row := range rows {
		out[i] = toDebtModel(row)
	}
	return out, nil
}

type UpdateDebtParams struct {
	ID               uuid.UUID
	WorkspaceID      uuid.UUID
	Name             string
	Lender           string
	Principal        float64
	Rate             float64
	RateType         string
	Installments     int32
	FirstPaymentDate time.Time
	Notes            string
}

func (r *DebtRepository) Update(ctx context.Context, p UpdateDebtParams) (models.Debt, error) {
	row, err := r.q.UpdateDebt(ctx, sqlc.UpdateDebtParams{
		ID:               p.ID,
		WorkspaceID:      p.WorkspaceID,
		Name:             p.Name,
		Lender:           toPgText(p.Lender),
		Principal:        p.Principal,
		Rate:             p.Rate,
		RateType:         p.RateType,
		Installments:     p.Installments,
		FirstPaymentDate: toPgDate(p.FirstPaymentDate),
		Notes:            toPgText(p.Notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Debt{}, apperror.ErrNotFound
		}
		return models.Debt{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toDebtModel(row), nil
}

func (r *DebtRepository) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return r.q.DeleteDebt(ctx, sqlc.DeleteDebtParams{ID: id, WorkspaceID: workspaceID})
}

type CreateDebtPaymentParams struct {
	DebtID uuid.UUID
	Period int32
	Amount float64
	PaidAt time.Time
	Notes  string
}

func (r *DebtRepository) CreatePayment(ctx context.Context, p CreateDebtPaymentParams) (models.DebtPayment, error) {
	row, err := r.q.CreateDebtPayment(ctx, sqlc.CreateDebtPaymentParams{
		DebtID: p.DebtID,
		Period: p.Period,
		Amount: p.Amount,
		PaidAt: toPgDate(p.PaidAt),
		Notes:  toPgText(p.Notes),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return models.DebtPayment{}, apperror.ErrConflict
		}
		return models.DebtPayment{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toDebtPaymentModel(row), nil
}

func (r *DebtRepository) GetPayment(ctx context.Context, id uuid.UUID) (models.DebtPayment, error) {
	row, err := r.q.GetDebtPayment(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.DebtPayment{}, apperror.ErrNotFound
		}
		return models.DebtPayment{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toDebtPaymentModel(row), nil
}

func (r *DebtRepository) ListPayments(ctx context.Context, debtID uuid.UUID) ([]models.DebtPayment, error) {
	rows, err := r.q.ListDebtPayments(ctx, debtID)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.DebtPayment, len(rows))
	for i, row := range rows {
		out[i] = toDebtPaymentModel(row)
	}
	return out, nil
}

type UpdateDebtPaymentParams struct {
	ID     uuid.UUID
	Amount float64
	PaidAt time.Time
	Notes  string
}

func (r *DebtRepository) UpdatePayment(ctx context.Context, p UpdateDebtPaymentParams) (models.DebtPayment, error) {
	row, err := r.q.UpdateDebtPayment(ctx, sqlc.UpdateDebtPaymentParams{
		ID:     p.ID,
		Amount: p.Amount,
		PaidAt: toPgDate(p.PaidAt),
		Notes:  toPgText(p.Notes),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.DebtPayment{}, apperror.ErrNotFound
		}
		return models.DebtPayment{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toDebtPaymentModel(row), nil
}

func (r *DebtRepository) DeletePayment(ctx context.Context, id, debtID uuid.UUID) error {
	return r.q.DeleteDebtPayment(ctx, sqlc.DeleteDebtPaymentParams{ID: id, DebtID: debtID})
}

func toDebtModel(row sqlc.Debt) models.Debt {
	return models.Debt{
		ID:               row.ID,
		WorkspaceID:      row.WorkspaceID,
		Name:             row.Name,
		Lender:           fromPgText(row.Lender),
		Principal:        row.Principal,
		Rate:             row.Rate,
		RateType:         row.RateType,
		Installments:     row.Installments,
		FirstPaymentDate: fromPgDate(row.FirstPaymentDate),
		Notes:            fromPgText(row.Notes),
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}

func toDebtPaymentModel(row sqlc.DebtPayment) models.DebtPayment {
	return models.DebtPayment{
		ID:        row.ID,
		DebtID:    row.DebtID,
		Period:    row.Period,
		Amount:    row.Amount,
		PaidAt:    fromPgDate(row.PaidAt),
		Notes:     fromPgText(row.Notes),
		CreatedAt: row.CreatedAt.Time,
	}
}
