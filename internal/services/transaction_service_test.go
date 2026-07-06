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

type mockTransactionRepo struct {
	createFn         func(ctx context.Context, p repositories.CreateTransactionParams) (models.Transaction, error)
	createTransferFn func(ctx context.Context, p repositories.CreateTransferParams) (models.Transaction, models.Transaction, error)
	getByIDFn        func(ctx context.Context, id uuid.UUID) (models.Transaction, error)
	listFn           func(ctx context.Context, p repositories.ListTransactionsParams) ([]models.Transaction, error)
	countFn          func(ctx context.Context, p repositories.ListTransactionsParams) (int64, error)
	dailySummaryFn   func(ctx context.Context, p repositories.DailySummaryParams) ([]models.DailySummary, error)
	monthSummaryFn   func(ctx context.Context, p repositories.MonthSummaryParams) (repositories.MonthSummaryResult, error)
	listByDateCursorFn func(ctx context.Context, p repositories.ListByDateCursorParams) ([]models.Transaction, error)
	updateFn         func(ctx context.Context, p repositories.UpdateTransactionParams) (models.Transaction, error)
	deleteFn         func(ctx context.Context, id, workspaceID uuid.UUID) error
}

func (m *mockTransactionRepo) Create(ctx context.Context, p repositories.CreateTransactionParams) (models.Transaction, error) {
	return m.createFn(ctx, p)
}
func (m *mockTransactionRepo) CreateTransfer(ctx context.Context, p repositories.CreateTransferParams) (models.Transaction, models.Transaction, error) {
	return m.createTransferFn(ctx, p)
}
func (m *mockTransactionRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Transaction, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockTransactionRepo) List(ctx context.Context, p repositories.ListTransactionsParams) ([]models.Transaction, error) {
	return m.listFn(ctx, p)
}
func (m *mockTransactionRepo) Count(ctx context.Context, p repositories.ListTransactionsParams) (int64, error) {
	return m.countFn(ctx, p)
}
func (m *mockTransactionRepo) DailySummary(ctx context.Context, p repositories.DailySummaryParams) ([]models.DailySummary, error) {
	return m.dailySummaryFn(ctx, p)
}
func (m *mockTransactionRepo) MonthSummary(ctx context.Context, p repositories.MonthSummaryParams) (repositories.MonthSummaryResult, error) {
	return m.monthSummaryFn(ctx, p)
}
func (m *mockTransactionRepo) ListByDateCursor(ctx context.Context, p repositories.ListByDateCursorParams) ([]models.Transaction, error) {
	return m.listByDateCursorFn(ctx, p)
}
func (m *mockTransactionRepo) Update(ctx context.Context, p repositories.UpdateTransactionParams) (models.Transaction, error) {
	return m.updateFn(ctx, p)
}
func (m *mockTransactionRepo) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return m.deleteFn(ctx, id, workspaceID)
}

func TestTransactionService_Create(t *testing.T) {
	wsID := uuid.New()
	userID := uuid.New()
	now := time.Now().UTC()
	date := now.Truncate(24 * time.Hour)

	tests := []struct {
		name    string
		params  CreateTransactionParams
		wantErr bool
		errType error
	}{
		{
			name:   "expense success",
			params: CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 50000, Date: date},
		},
		{
			name:   "income success",
			params: CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "income", Amount: 1000000, Date: date},
		},
		{
			name:    "transfer type rejected via Create",
			params:  CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "transfer", Amount: 100, Date: date},
			wantErr: true,
			errType: apperror.ErrInvalidInput,
		},
		{
			name:    "zero amount",
			params:  CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 0, Date: date},
			wantErr: true,
			errType: apperror.ErrInvalidInput,
		},
		{
			name:    "zero date",
			params:  CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 100},
			wantErr: true,
			errType: apperror.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockTransactionRepo{
				createFn: func(_ context.Context, p repositories.CreateTransactionParams) (models.Transaction, error) {
					return models.Transaction{
						ID: uuid.New(), WorkspaceID: p.WorkspaceID, UserID: p.UserID,
						Type: p.Type, Amount: p.Amount, Date: p.Date, CreatedAt: now, UpdatedAt: now,
					}, nil
				},
			}
			svc := NewTransactionService(repo)
			tx, err := svc.Create(context.Background(), tt.params)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.params.Amount, tx.Amount)
			assert.Equal(t, tt.params.Type, tx.Type)
		})
	}
}

func TestTransactionService_Update_TransferBlocked(t *testing.T) {
	wsID := uuid.New()
	txID := uuid.New()
	now := time.Now().UTC()
	transferTx := models.Transaction{ID: txID, WorkspaceID: wsID, Type: "transfer", Amount: 100, Date: now, CreatedAt: now, UpdatedAt: now}

	repo := &mockTransactionRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Transaction, error) { return transferTx, nil },
	}
	svc := NewTransactionService(repo)

	_, err := svc.Update(context.Background(), UpdateTransactionParams{
		ID: txID, WorkspaceID: wsID, Amount: 200, Date: now,
	})
	assert.ErrorIs(t, err, apperror.ErrForbidden)
}

func TestTransactionService_Delete_WrongWorkspace(t *testing.T) {
	wsID := uuid.New()
	otherWS := uuid.New()
	txID := uuid.New()
	now := time.Now().UTC()
	tx := models.Transaction{ID: txID, WorkspaceID: wsID, Type: "expense", Amount: 100, Date: now}

	repo := &mockTransactionRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Transaction, error) { return tx, nil },
	}
	svc := NewTransactionService(repo)

	err := svc.Delete(context.Background(), txID, otherWS)
	assert.ErrorIs(t, err, apperror.ErrForbidden)
}

func TestTransactionService_ListByDate_CursorSet(t *testing.T) {
	wsID := uuid.New()
	date := time.Now().Truncate(24 * time.Hour)
	now := time.Now().UTC()

	txs := []models.Transaction{
		{ID: uuid.New(), WorkspaceID: wsID, Type: "expense", Amount: 10, Date: date, CreatedAt: now.Add(-1 * time.Minute)},
		{ID: uuid.New(), WorkspaceID: wsID, Type: "income", Amount: 20, Date: date, CreatedAt: now.Add(-2 * time.Minute)},
	}

	repo := &mockTransactionRepo{
		listByDateCursorFn: func(_ context.Context, _ repositories.ListByDateCursorParams) ([]models.Transaction, error) {
			return txs, nil
		},
	}
	svc := NewTransactionService(repo)

	// limit == len(txs) → next_cursor should be set
	result, err := svc.ListByDate(context.Background(), ListByDateParams{
		WorkspaceID: wsID, Date: date, Limit: 2,
	})
	assert.NoError(t, err)
	assert.Len(t, result.Items, 2)
	assert.NotNil(t, result.NextCursor)
}
