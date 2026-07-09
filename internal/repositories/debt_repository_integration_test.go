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

func createTestWorkspace(t *testing.T, wsRepo *WorkspaceRepository, ownerID uuid.UUID) uuid.UUID {
	t.Helper()
	ws, err := wsRepo.Create(context.Background(), CreateWorkspaceParams{Name: "Test WS", OwnerID: ownerID, Currency: "COP"})
	require.NoError(t, err)
	return ws.ID
}

func TestDebtRepository_CreateAndGetByID(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	debtRepo := NewDebtRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)

	firstPayment := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	debt, err := debtRepo.Create(context.Background(), CreateDebtParams{
		WorkspaceID: wsID, Name: "Car loan", Lender: "Bank", Principal: 12000000,
		Rate: 12, RateType: "effective_annual", Installments: 24,
		FirstPaymentDate: firstPayment, InsuranceType: "",
	})
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, debt.ID)
	assert.Equal(t, "Car loan", debt.Name)
	assert.Equal(t, "Bank", debt.Lender)
	assert.True(t, debt.FirstPaymentDate.Equal(firstPayment))

	got, err := debtRepo.GetByID(context.Background(), debt.ID)
	require.NoError(t, err)
	assert.Equal(t, debt.ID, got.ID)
	assert.Equal(t, debt.Principal, got.Principal)
}

func TestDebtRepository_GetByID_NotFound(t *testing.T) {
	pool := setupTestDB(t)
	debtRepo := NewDebtRepository(pool)

	_, err := debtRepo.GetByID(context.Background(), uuid.New())
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestDebtRepository_List(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	debtRepo := NewDebtRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)
	firstPayment := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	for _, name := range []string{"Loan A", "Loan B"} {
		_, err := debtRepo.Create(context.Background(), CreateDebtParams{
			WorkspaceID: wsID, Name: name, Principal: 1000, Rate: 1, RateType: "monthly",
			Installments: 12, FirstPaymentDate: firstPayment,
		})
		require.NoError(t, err)
	}

	list, err := debtRepo.List(context.Background(), wsID)
	require.NoError(t, err)
	assert.Len(t, list, 2)
}

func TestDebtRepository_Update(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	debtRepo := NewDebtRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)
	firstPayment := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	debt, err := debtRepo.Create(context.Background(), CreateDebtParams{
		WorkspaceID: wsID, Name: "Original", Principal: 1000, Rate: 1, RateType: "monthly",
		Installments: 12, FirstPaymentDate: firstPayment,
	})
	require.NoError(t, err)

	updated, err := debtRepo.Update(context.Background(), UpdateDebtParams{
		ID: debt.ID, WorkspaceID: wsID, Name: "Renamed", Principal: 2000, Rate: 2,
		RateType: "monthly", Installments: 24, FirstPaymentDate: firstPayment,
	})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", updated.Name)
	assert.Equal(t, float64(2000), updated.Principal)
}

func TestDebtRepository_Update_WrongWorkspace(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	debtRepo := NewDebtRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)
	otherWsID := createTestWorkspace(t, wsRepo, ownerID)
	firstPayment := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	debt, err := debtRepo.Create(context.Background(), CreateDebtParams{
		WorkspaceID: wsID, Name: "X", Principal: 1000, Rate: 1, RateType: "monthly",
		Installments: 12, FirstPaymentDate: firstPayment,
	})
	require.NoError(t, err)

	_, err = debtRepo.Update(context.Background(), UpdateDebtParams{
		ID: debt.ID, WorkspaceID: otherWsID, Name: "Hijacked", Principal: 1, Rate: 1,
		RateType: "monthly", Installments: 1, FirstPaymentDate: firstPayment,
	})
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestDebtRepository_Delete(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	debtRepo := NewDebtRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)
	firstPayment := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	debt, err := debtRepo.Create(context.Background(), CreateDebtParams{
		WorkspaceID: wsID, Name: "ToDelete", Principal: 1000, Rate: 1, RateType: "monthly",
		Installments: 12, FirstPaymentDate: firstPayment,
	})
	require.NoError(t, err)

	require.NoError(t, debtRepo.Delete(context.Background(), debt.ID, wsID))

	_, err = debtRepo.GetByID(context.Background(), debt.ID)
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestDebtRepository_Payments_CRUD(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	debtRepo := NewDebtRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)
	firstPayment := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	debt, err := debtRepo.Create(context.Background(), CreateDebtParams{
		WorkspaceID: wsID, Name: "Loan", Principal: 1000, Rate: 1, RateType: "monthly",
		Installments: 12, FirstPaymentDate: firstPayment,
	})
	require.NoError(t, err)

	paidAt := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	payment, err := debtRepo.CreatePayment(context.Background(), CreateDebtPaymentParams{
		DebtID: debt.ID, Period: 1, Amount: 100, PaidAt: paidAt, Notes: "first",
	})
	require.NoError(t, err)
	assert.Equal(t, int32(1), payment.Period)

	got, err := debtRepo.GetPayment(context.Background(), payment.ID)
	require.NoError(t, err)
	assert.Equal(t, payment.ID, got.ID)

	list, err := debtRepo.ListPayments(context.Background(), debt.ID)
	require.NoError(t, err)
	assert.Len(t, list, 1)

	updated, err := debtRepo.UpdatePayment(context.Background(), UpdateDebtPaymentParams{
		ID: payment.ID, Amount: 150, PaidAt: paidAt, Notes: "updated",
	})
	require.NoError(t, err)
	assert.Equal(t, float64(150), updated.Amount)

	require.NoError(t, debtRepo.DeletePayment(context.Background(), payment.ID, debt.ID))
	_, err = debtRepo.GetPayment(context.Background(), payment.ID)
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestDebtRepository_CreatePayment_DuplicatePeriodConflict(t *testing.T) {
	pool := setupTestDB(t)
	userRepo := NewUserRepository(pool)
	wsRepo := NewWorkspaceRepository(pool)
	debtRepo := NewDebtRepository(pool)
	ownerID := createTestUser(t, userRepo)
	wsID := createTestWorkspace(t, wsRepo, ownerID)
	firstPayment := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	debt, err := debtRepo.Create(context.Background(), CreateDebtParams{
		WorkspaceID: wsID, Name: "Loan", Principal: 1000, Rate: 1, RateType: "monthly",
		Installments: 12, FirstPaymentDate: firstPayment,
	})
	require.NoError(t, err)

	paidAt := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)
	_, err = debtRepo.CreatePayment(context.Background(), CreateDebtPaymentParams{
		DebtID: debt.ID, Period: 1, Amount: 100, PaidAt: paidAt,
	})
	require.NoError(t, err)

	_, err = debtRepo.CreatePayment(context.Background(), CreateDebtPaymentParams{
		DebtID: debt.ID, Period: 1, Amount: 999, PaidAt: paidAt,
	})
	assert.ErrorIs(t, err, apperror.ErrConflict)
}
