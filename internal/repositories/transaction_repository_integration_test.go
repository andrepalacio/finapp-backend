//go:build integration

package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionRepository_CreateAndGetByID(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	txRepo := NewTransactionRepository(pool)
	userID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, userID)

	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	tx, err := txRepo.Create(context.Background(), CreateTransactionParams{
		WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 50000, Description: "Groceries", Date: date,
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, tx.ID)
	assert.Equal(t, "expense", tx.Type)
	assert.Equal(t, "Groceries", tx.Description)
	assert.True(t, tx.Date.Equal(date))

	got, err := txRepo.GetByID(context.Background(), tx.ID)
	require.NoError(t, err)
	assert.Equal(t, tx.ID, got.ID)
}

func TestTransactionRepository_GetByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	txRepo := NewTransactionRepository(pool)

	_, err := txRepo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestTransactionRepository_CreateTransfer(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	txRepo := NewTransactionRepository(pool)
	userID := createTestUser(t, userRepo)
	fromWS := createTestWorkspace(t, wsRepo, userID)
	toWS := createTestWorkspace(t, wsRepo, userID)

	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	out, in, err := txRepo.CreateTransfer(context.Background(), CreateTransferParams{
		FromWorkspaceID: fromWS, ToWorkspaceID: toWS, UserID: userID, Amount: 100000,
		Description: "Rent split", Note: "monthly", Date: date,
	})
	require.NoError(t, err)

	assert.Equal(t, "transfer", out.Type)
	assert.Equal(t, "out", out.TransferDirection)
	assert.Equal(t, fromWS, out.WorkspaceID)
	assert.Equal(t, float64(100000), out.Amount)

	assert.Equal(t, "transfer", in.Type)
	assert.Equal(t, "in", in.TransferDirection)
	assert.Equal(t, toWS, in.WorkspaceID)
	assert.Equal(t, float64(100000), in.Amount)

	// Both legs must be linked to the same transfer record.
	require.NotNil(t, out.TransferID)
	require.NotNil(t, in.TransferID)
	assert.Equal(t, *out.TransferID, *in.TransferID)

	// Each leg is independently retrievable from its own workspace.
	outList, err := txRepo.List(context.Background(), ListTransactionsParams{WorkspaceID: fromWS, Limit: 10})
	require.NoError(t, err)
	require.Len(t, outList, 1)
	assert.Equal(t, "out", outList[0].TransferDirection)

	inList, err := txRepo.List(context.Background(), ListTransactionsParams{WorkspaceID: toWS, Limit: 10})
	require.NoError(t, err)
	require.Len(t, inList, 1)
	assert.Equal(t, "in", inList[0].TransferDirection)
}

func TestTransactionRepository_ListAndCount_Filters(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	txRepo := NewTransactionRepository(pool)
	userID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, userID)

	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err := txRepo.Create(context.Background(), CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 100, Date: date})
	require.NoError(t, err)
	_, err = txRepo.Create(context.Background(), CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "income", Amount: 500, Date: date})
	require.NoError(t, err)

	all, err := txRepo.List(context.Background(), ListTransactionsParams{WorkspaceID: wsID, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, all, 2)

	count, err := txRepo.Count(context.Background(), ListTransactionsParams{WorkspaceID: wsID})
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)

	expensesOnly, err := txRepo.List(context.Background(), ListTransactionsParams{WorkspaceID: wsID, Type: "expense", Limit: 10})
	require.NoError(t, err)
	require.Len(t, expensesOnly, 1)
	assert.Equal(t, "expense", expensesOnly[0].Type)

	expenseCount, err := txRepo.Count(context.Background(), ListTransactionsParams{WorkspaceID: wsID, Type: "expense"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), expenseCount)
}

func TestTransactionRepository_DailySummary(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	txRepo := NewTransactionRepository(pool)
	userID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, userID)

	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err := txRepo.Create(context.Background(), CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 100, Date: date})
	require.NoError(t, err)
	_, err = txRepo.Create(context.Background(), CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "income", Amount: 500, Date: date})
	require.NoError(t, err)

	rows, err := txRepo.DailySummary(context.Background(), DailySummaryParams{WorkspaceID: wsID, Limit: 30})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, float64(100), rows[0].TotalExpense)
	assert.Equal(t, float64(500), rows[0].TotalIncome)
	assert.Equal(t, int32(2), rows[0].TransactionCount)
	assert.True(t, rows[0].Date.Equal(date))
}

func TestTransactionRepository_MonthSummary(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	txRepo := NewTransactionRepository(pool)
	userID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, userID)

	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
	_, err := txRepo.Create(context.Background(), CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 100, Date: date})
	require.NoError(t, err)
	_, err = txRepo.Create(context.Background(), CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 50, Date: date})
	require.NoError(t, err)
	_, err = txRepo.Create(context.Background(), CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "income", Amount: 1000, Date: date})
	require.NoError(t, err)

	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	summary, err := txRepo.MonthSummary(context.Background(), MonthSummaryParams{WorkspaceID: wsID, DateFrom: &from, DateTo: &to})
	require.NoError(t, err)
	assert.Equal(t, float64(150), summary.ExpenseTotal)
	assert.Equal(t, int32(2), summary.ExpenseCount)
	assert.Equal(t, float64(1000), summary.IncomeTotal)
	assert.Equal(t, int32(1), summary.IncomeCount)
}

func TestTransactionRepository_ListByDateCursor_Pagination(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	txRepo := NewTransactionRepository(pool)
	userID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, userID)
	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	for i := 0; i < 3; i++ {
		_, err := txRepo.Create(context.Background(), CreateTransactionParams{
			WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: float64(i + 1), Date: date,
		})
		require.NoError(t, err)
	}

	firstPage, err := txRepo.ListByDateCursor(context.Background(), ListByDateCursorParams{WorkspaceID: wsID, Date: date, Limit: 2})
	require.NoError(t, err)
	require.Len(t, firstPage, 2)

	cursor := firstPage[len(firstPage)-1].CreatedAt
	secondPage, err := txRepo.ListByDateCursor(context.Background(), ListByDateCursorParams{WorkspaceID: wsID, Date: date, Cursor: &cursor, Limit: 2})
	require.NoError(t, err)
	require.Len(t, secondPage, 1)

	// No overlap between pages.
	assert.NotEqual(t, firstPage[0].ID, secondPage[0].ID)
	assert.NotEqual(t, firstPage[1].ID, secondPage[0].ID)
}

func TestTransactionRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	txRepo := NewTransactionRepository(pool)
	userID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, userID)
	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	tx, err := txRepo.Create(context.Background(), CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 100, Date: date})
	require.NoError(t, err)

	newDate := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
	updated, err := txRepo.Update(context.Background(), UpdateTransactionParams{
		ID: tx.ID, Amount: 200, Description: "Updated", Date: newDate,
	})
	require.NoError(t, err)
	assert.Equal(t, float64(200), updated.Amount)
	assert.Equal(t, "Updated", updated.Description)
	assert.True(t, updated.Date.Equal(newDate))
}

func TestTransactionRepository_Update_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	txRepo := NewTransactionRepository(pool)

	_, err := txRepo.Update(context.Background(), UpdateTransactionParams{
		ID: uuid.New(), Amount: 100, Date: time.Now(),
	})
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestTransactionRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	txRepo := NewTransactionRepository(pool)
	userID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, userID)
	date := time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)

	tx, err := txRepo.Create(context.Background(), CreateTransactionParams{WorkspaceID: wsID, UserID: userID, Type: "expense", Amount: 100, Date: date})
	require.NoError(t, err)

	require.NoError(t, txRepo.Delete(context.Background(), tx.ID, wsID))

	_, err = txRepo.GetByID(context.Background(), tx.ID)
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}
