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

type TransactionRepository struct {
	q    *sqlc.Queries
	pool *pgxpool.Pool
}

func NewTransactionRepository(pool *pgxpool.Pool) *TransactionRepository {
	return &TransactionRepository{q: sqlc.New(pool), pool: pool}
}

type CreateTransactionParams struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	CategoryID  *uuid.UUID
	Type        string
	Amount      float64
	Description string
	Date        time.Time
}

func (r *TransactionRepository) Create(ctx context.Context, p CreateTransactionParams) (models.Transaction, error) {
	row, err := r.q.CreateTransaction(ctx, sqlc.CreateTransactionParams{
		WorkspaceID: p.WorkspaceID,
		UserID:      p.UserID,
		CategoryID:  p.CategoryID,
		Type:        p.Type,
		Amount:      p.Amount,
		Description: toPgText(p.Description),
		Date:        toPgDate(p.Date),
	})
	if err != nil {
		return models.Transaction{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toTransactionModel(row), nil
}

type CreateTransferParams struct {
	FromWorkspaceID uuid.UUID
	ToWorkspaceID   uuid.UUID
	UserID          uuid.UUID
	Amount          float64
	Description     string
	Note            string
	Date            time.Time
}

func (r *TransactionRepository) CreateTransfer(ctx context.Context, p CreateTransferParams) (models.Transaction, models.Transaction, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return models.Transaction{}, models.Transaction{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	qtx := r.q.WithTx(tx)

	transfer, err := qtx.CreateTransferRecord(ctx, sqlc.CreateTransferRecordParams{
		FromWorkspaceID: p.FromWorkspaceID,
		ToWorkspaceID:   p.ToWorkspaceID,
		Note:            toPgText(p.Note),
	})
	if err != nil {
		return models.Transaction{}, models.Transaction{}, apperror.Wrap(apperror.ErrInternal, err)
	}

	transferID := transfer.ID
	outTx, err := qtx.CreateTransferTransaction(ctx, sqlc.CreateTransferTransactionParams{
		WorkspaceID:       p.FromWorkspaceID,
		UserID:            p.UserID,
		TransferID:        &transferID,
		TransferDirection: toPgText("out"),
		Amount:            p.Amount,
		Description:       toPgText(p.Description),
		Date:              toPgDate(p.Date),
	})
	if err != nil {
		return models.Transaction{}, models.Transaction{}, apperror.Wrap(apperror.ErrInternal, err)
	}

	inTx, err := qtx.CreateTransferTransaction(ctx, sqlc.CreateTransferTransactionParams{
		WorkspaceID:       p.ToWorkspaceID,
		UserID:            p.UserID,
		TransferID:        &transferID,
		TransferDirection: toPgText("in"),
		Amount:            p.Amount,
		Description:       toPgText(p.Description),
		Date:              toPgDate(p.Date),
	})
	if err != nil {
		return models.Transaction{}, models.Transaction{}, apperror.Wrap(apperror.ErrInternal, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Transaction{}, models.Transaction{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toTransactionModel(outTx), toTransactionModel(inTx), nil
}

func (r *TransactionRepository) GetByID(ctx context.Context, id uuid.UUID) (models.Transaction, error) {
	row, err := r.q.GetTransactionByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Transaction{}, apperror.ErrNotFound
		}
		return models.Transaction{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toTransactionModel(row), nil
}

type ListTransactionsParams struct {
	WorkspaceID uuid.UUID
	DateFrom    *time.Time
	DateTo      *time.Time
	Type        string
	CategoryID  *uuid.UUID
	Limit       int32
	Offset      int32
}

func (r *TransactionRepository) List(ctx context.Context, p ListTransactionsParams) ([]models.Transaction, error) {
	arg := sqlc.ListTransactionsParams{
		WorkspaceID: p.WorkspaceID,
		CategoryID:  p.CategoryID,
		Limit:       p.Limit,
		Offset:      p.Offset,
	}
	if p.DateFrom != nil {
		arg.DateFrom = toPgDate(*p.DateFrom)
	}
	if p.DateTo != nil {
		arg.DateTo = toPgDate(*p.DateTo)
	}
	if p.Type != "" {
		arg.TxType = toPgText(p.Type)
	}

	rows, err := r.q.ListTransactions(ctx, arg)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.Transaction, len(rows))
	for i, row := range rows {
		out[i] = toTransactionModel(row)
	}
	return out, nil
}

func (r *TransactionRepository) Count(ctx context.Context, p ListTransactionsParams) (int64, error) {
	arg := sqlc.CountTransactionsParams{
		WorkspaceID: p.WorkspaceID,
		CategoryID:  p.CategoryID,
	}
	if p.DateFrom != nil {
		arg.DateFrom = toPgDate(*p.DateFrom)
	}
	if p.DateTo != nil {
		arg.DateTo = toPgDate(*p.DateTo)
	}
	if p.Type != "" {
		arg.TxType = toPgText(p.Type)
	}
	return r.q.CountTransactions(ctx, arg)
}

type DailySummaryParams struct {
	WorkspaceID uuid.UUID
	DateFrom    *time.Time
	DateTo      *time.Time
	Limit       int32
	Offset      int32
}

func (r *TransactionRepository) DailySummary(ctx context.Context, p DailySummaryParams) ([]models.DailySummary, error) {
	arg := sqlc.GetDailySummaryParams{
		WorkspaceID: p.WorkspaceID,
		Limit:       p.Limit,
		Offset:      p.Offset,
	}
	if p.DateFrom != nil {
		arg.DateFrom = toPgDate(*p.DateFrom)
	}
	if p.DateTo != nil {
		arg.DateTo = toPgDate(*p.DateTo)
	}

	rows, err := r.q.GetDailySummary(ctx, arg)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.DailySummary, len(rows))
	for i, row := range rows {
		out[i] = models.DailySummary{
			Date:             fromPgDate(row.Date),
			TotalExpense:     row.TotalExpense,
			TotalIncome:      row.TotalIncome,
			TotalTransferOut: row.TotalTransferOut,
			TotalTransferIn:  row.TotalTransferIn,
			TransactionCount: row.TransactionCount,
		}
	}
	return out, nil
}

type MonthSummaryParams struct {
	WorkspaceID uuid.UUID
	DateFrom    *time.Time
	DateTo      *time.Time
}

type MonthSummaryResult struct {
	IncomeTotal  float64
	IncomeCount  int32
	ExpenseTotal float64
	ExpenseCount int32
}

func (r *TransactionRepository) MonthSummary(ctx context.Context, p MonthSummaryParams) (MonthSummaryResult, error) {
	arg := sqlc.GetMonthSummaryParams{WorkspaceID: p.WorkspaceID}
	if p.DateFrom != nil {
		arg.DateFrom = toPgDate(*p.DateFrom)
	}
	if p.DateTo != nil {
		arg.DateTo = toPgDate(*p.DateTo)
	}
	row, err := r.q.GetMonthSummary(ctx, arg)
	if err != nil {
		return MonthSummaryResult{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return MonthSummaryResult{
		IncomeTotal:  row.IncomeTotal,
		IncomeCount:  row.IncomeCount,
		ExpenseTotal: row.ExpenseTotal,
		ExpenseCount: row.ExpenseCount,
	}, nil
}

type ListByDateCursorParams struct {
	WorkspaceID uuid.UUID
	Date        time.Time
	Cursor      *time.Time
	Limit       int32
}

func (r *TransactionRepository) ListByDateCursor(ctx context.Context, p ListByDateCursorParams) ([]models.Transaction, error) {
	arg := sqlc.ListTransactionsByDateCursorParams{
		WorkspaceID: p.WorkspaceID,
		Date:        toPgDate(p.Date),
		Limit:       p.Limit,
	}
	if p.Cursor != nil {
		arg.Cursor = pgtype.Timestamptz{Time: *p.Cursor, Valid: true}
	}

	rows, err := r.q.ListTransactionsByDateCursor(ctx, arg)
	if err != nil {
		return nil, apperror.Wrap(apperror.ErrInternal, err)
	}
	out := make([]models.Transaction, len(rows))
	for i, row := range rows {
		out[i] = toTransactionModel(row)
	}
	return out, nil
}

type UpdateTransactionParams struct {
	ID          uuid.UUID
	CategoryID  *uuid.UUID
	Amount      float64
	Description string
	Date        time.Time
}

func (r *TransactionRepository) Update(ctx context.Context, p UpdateTransactionParams) (models.Transaction, error) {
	row, err := r.q.UpdateTransaction(ctx, sqlc.UpdateTransactionParams{
		ID:          p.ID,
		CategoryID:  p.CategoryID,
		Amount:      p.Amount,
		Description: toPgText(p.Description),
		Date:        toPgDate(p.Date),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Transaction{}, apperror.ErrNotFound
		}
		return models.Transaction{}, apperror.Wrap(apperror.ErrInternal, err)
	}
	return toTransactionModel(row), nil
}

func (r *TransactionRepository) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return r.q.DeleteTransaction(ctx, sqlc.DeleteTransactionParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
}

func toTransactionModel(row sqlc.Transaction) models.Transaction {
	return models.Transaction{
		ID:                row.ID,
		WorkspaceID:       row.WorkspaceID,
		UserID:            row.UserID,
		CategoryID:        row.CategoryID,
		TransferID:        row.TransferID,
		Type:              row.Type,
		TransferDirection: fromPgText(row.TransferDirection),
		Amount:            row.Amount,
		Description:       fromPgText(row.Description),
		Date:              fromPgDate(row.Date),
		CreatedAt:         row.CreatedAt.Time,
		UpdatedAt:         row.UpdatedAt.Time,
	}
}
