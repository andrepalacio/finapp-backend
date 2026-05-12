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

type mockDebtRepo struct {
	createFn        func(ctx context.Context, p repositories.CreateDebtParams) (models.Debt, error)
	getByIDFn       func(ctx context.Context, id uuid.UUID) (models.Debt, error)
	listFn          func(ctx context.Context, workspaceID uuid.UUID) ([]models.Debt, error)
	updateFn        func(ctx context.Context, p repositories.UpdateDebtParams) (models.Debt, error)
	deleteFn        func(ctx context.Context, id, workspaceID uuid.UUID) error
	createPaymentFn func(ctx context.Context, p repositories.CreateDebtPaymentParams) (models.DebtPayment, error)
	getPaymentFn    func(ctx context.Context, id uuid.UUID) (models.DebtPayment, error)
	listPaymentsFn  func(ctx context.Context, debtID uuid.UUID) ([]models.DebtPayment, error)
	updatePaymentFn func(ctx context.Context, p repositories.UpdateDebtPaymentParams) (models.DebtPayment, error)
	deletePaymentFn func(ctx context.Context, id, debtID uuid.UUID) error
}

func (m *mockDebtRepo) Create(ctx context.Context, p repositories.CreateDebtParams) (models.Debt, error) {
	return m.createFn(ctx, p)
}
func (m *mockDebtRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Debt, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockDebtRepo) List(ctx context.Context, workspaceID uuid.UUID) ([]models.Debt, error) {
	return m.listFn(ctx, workspaceID)
}
func (m *mockDebtRepo) Update(ctx context.Context, p repositories.UpdateDebtParams) (models.Debt, error) {
	return m.updateFn(ctx, p)
}
func (m *mockDebtRepo) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return m.deleteFn(ctx, id, workspaceID)
}
func (m *mockDebtRepo) CreatePayment(ctx context.Context, p repositories.CreateDebtPaymentParams) (models.DebtPayment, error) {
	return m.createPaymentFn(ctx, p)
}
func (m *mockDebtRepo) GetPayment(ctx context.Context, id uuid.UUID) (models.DebtPayment, error) {
	return m.getPaymentFn(ctx, id)
}
func (m *mockDebtRepo) ListPayments(ctx context.Context, debtID uuid.UUID) ([]models.DebtPayment, error) {
	return m.listPaymentsFn(ctx, debtID)
}
func (m *mockDebtRepo) UpdatePayment(ctx context.Context, p repositories.UpdateDebtPaymentParams) (models.DebtPayment, error) {
	return m.updatePaymentFn(ctx, p)
}
func (m *mockDebtRepo) DeletePayment(ctx context.Context, id, debtID uuid.UUID) error {
	return m.deletePaymentFn(ctx, id, debtID)
}

func makeDebt(wsID uuid.UUID) models.Debt {
	return models.Debt{
		ID:               uuid.New(),
		WorkspaceID:      wsID,
		Name:             "Car loan",
		Principal:        12000000,
		Rate:             0.12,
		RateType:         "effective_annual",
		Installments:     24,
		FirstPaymentDate: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
}

func TestDebtService_Create_Validation(t *testing.T) {
	wsID := uuid.New()
	firstPayment := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		params  CreateDebtParams
		wantErr bool
		errType error
	}{
		{
			name: "success",
			params: CreateDebtParams{
				WorkspaceID: wsID, Name: "Car loan", Principal: 12000000,
				Rate: 0.12, RateType: "effective_annual", Installments: 24,
				FirstPaymentDate: firstPayment,
			},
		},
		{
			name:    "empty name",
			params:  CreateDebtParams{WorkspaceID: wsID, Principal: 100, Rate: 0.1, RateType: "monthly", Installments: 12, FirstPaymentDate: firstPayment},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
		{
			name:    "zero principal",
			params:  CreateDebtParams{WorkspaceID: wsID, Name: "X", Rate: 0.1, RateType: "monthly", Installments: 12, FirstPaymentDate: firstPayment},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
		{
			name:    "invalid rate type",
			params:  CreateDebtParams{WorkspaceID: wsID, Name: "X", Principal: 100, Rate: 0.1, RateType: "bad", Installments: 12, FirstPaymentDate: firstPayment},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
		{
			name:    "zero installments",
			params:  CreateDebtParams{WorkspaceID: wsID, Name: "X", Principal: 100, Rate: 0.1, RateType: "monthly", Installments: 0, FirstPaymentDate: firstPayment},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
		{
			name:    "zero date",
			params:  CreateDebtParams{WorkspaceID: wsID, Name: "X", Principal: 100, Rate: 0.1, RateType: "monthly", Installments: 12},
			wantErr: true, errType: apperror.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockDebtRepo{
				createFn: func(_ context.Context, p repositories.CreateDebtParams) (models.Debt, error) {
					debt := makeDebt(p.WorkspaceID)
					debt.Name = p.Name
					return debt, nil
				},
			}
			svc := NewDebtService(repo)
			_, err := svc.Create(context.Background(), tt.params)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestDebtService_GetByID_WrongWorkspace(t *testing.T) {
	wsID := uuid.New()
	otherWS := uuid.New()
	debt := makeDebt(wsID)

	repo := &mockDebtRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Debt, error) { return debt, nil },
	}
	svc := NewDebtService(repo)

	_, err := svc.GetByID(context.Background(), debt.ID, otherWS)
	assert.ErrorIs(t, err, apperror.ErrNotFound)
}

func TestDebtService_RecordPayment_PeriodOutOfRange(t *testing.T) {
	wsID := uuid.New()
	debt := makeDebt(wsID)

	repo := &mockDebtRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Debt, error) { return debt, nil },
	}
	svc := NewDebtService(repo)

	_, err := svc.RecordPayment(context.Background(), wsID, RecordPaymentParams{
		DebtID: debt.ID, Period: 25, Amount: 100, PaidAt: time.Now(),
	})
	assert.ErrorIs(t, err, apperror.ErrInvalidInput)
}

func TestDebtService_RecordPayment_ZeroPeriod(t *testing.T) {
	wsID := uuid.New()
	debt := makeDebt(wsID)

	repo := &mockDebtRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Debt, error) { return debt, nil },
	}
	svc := NewDebtService(repo)

	_, err := svc.RecordPayment(context.Background(), wsID, RecordPaymentParams{
		DebtID: debt.ID, Period: 0, Amount: 100, PaidAt: time.Now(),
	})
	assert.ErrorIs(t, err, apperror.ErrInvalidInput)
}

func TestComputeSchedule_FrenchAmortization(t *testing.T) {
	debt := models.Debt{
		Principal:        1200000,
		Rate:             0.12,
		RateType:         "nominal_annual",
		Installments:     12,
		FirstPaymentDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	schedule := computeSchedule(debt, nil)

	assert.Len(t, schedule, 12)
	// monthly rate = 0.12/12 = 0.01
	// M = 1200000 * 0.01 * (1.01)^12 / ((1.01)^12 - 1)
	// Verify first installment
	assert.Equal(t, int32(1), schedule[0].Period)
	assert.Equal(t, "pending", schedule[0].Status)
	assert.True(t, schedule[0].Interest > 0)
	assert.True(t, schedule[0].Principal > 0)
	assert.InDelta(t, schedule[0].Payment, schedule[0].Principal+schedule[0].Interest, 0.01)
	// Last installment balance should be 0
	assert.InDelta(t, 0.0, schedule[11].Balance, 1.0)
}

func TestComputeSchedule_WithPayments_C3Status(t *testing.T) {
	debt := models.Debt{
		Principal:        600000,
		Rate:             0.01,
		RateType:         "monthly",
		Installments:     6,
		FirstPaymentDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	paidAt := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	payments := []models.DebtPayment{
		{Period: 1, Amount: 103000, PaidAt: paidAt},
		{Period: 3, Amount: 101000, PaidAt: paidAt},
	}

	schedule := computeSchedule(debt, payments)

	assert.Equal(t, "paid", schedule[0].Status)
	assert.NotNil(t, schedule[0].PaidAt)
	assert.Equal(t, "pending", schedule[1].Status)
	assert.Nil(t, schedule[1].PaidAt)
	assert.Equal(t, "paid", schedule[2].Status)
	assert.Equal(t, "pending", schedule[3].Status)
}

func TestComputeSchedule_ZeroRate(t *testing.T) {
	debt := models.Debt{
		Principal:        1200000,
		Rate:             0,
		RateType:         "monthly",
		Installments:     12,
		FirstPaymentDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	schedule := computeSchedule(debt, nil)

	assert.Len(t, schedule, 12)
	for _, inst := range schedule {
		assert.Equal(t, float64(0), inst.Interest)
	}
	assert.InDelta(t, 0.0, schedule[11].Balance, 0.01)
}

func TestComputeSchedule_EffectiveAnnualRate(t *testing.T) {
	debt := models.Debt{
		Principal:        1000000,
		Rate:             0.12,
		RateType:         "effective_annual",
		Installments:     12,
		FirstPaymentDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	schedule := computeSchedule(debt, nil)

	assert.Len(t, schedule, 12)
	// r_m = (1.12)^(1/12) - 1 ≈ 0.009489
	// All payments approx equal
	firstPayment := schedule[0].Payment
	for i := 0; i < 11; i++ {
		assert.InDelta(t, firstPayment, schedule[i].Payment, 1.0)
	}
	assert.InDelta(t, 0.0, schedule[11].Balance, 1.0)
}
