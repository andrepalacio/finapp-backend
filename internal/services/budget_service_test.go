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

type mockBudgetRepo struct {
	upsertFn          func(ctx context.Context, p repositories.UpsertBudgetParams) (models.Budget, error)
	getByYearMonthFn  func(ctx context.Context, workspaceID uuid.UUID, year, month int16) (models.Budget, error)
	getByIDFn         func(ctx context.Context, id uuid.UUID) (models.Budget, error)
	listFn            func(ctx context.Context, workspaceID uuid.UUID) ([]models.Budget, error)
	deleteFn          func(ctx context.Context, id uuid.UUID) error
	upsertCategoryFn  func(ctx context.Context, budgetID, categoryID uuid.UUID, limit float64) error
	deleteCategoryFn  func(ctx context.Context, budgetID, categoryID uuid.UUID) error
	listCategoriesFn  func(ctx context.Context, budgetID uuid.UUID) ([]models.BudgetCategory, error)
	categorySpendingFn func(ctx context.Context, budgetID, workspaceID uuid.UUID, year, month int32) ([]models.BudgetCategorySpending, error)
}

func (m *mockBudgetRepo) Upsert(ctx context.Context, p repositories.UpsertBudgetParams) (models.Budget, error) {
	return m.upsertFn(ctx, p)
}
func (m *mockBudgetRepo) GetByYearMonth(ctx context.Context, workspaceID uuid.UUID, year, month int16) (models.Budget, error) {
	return m.getByYearMonthFn(ctx, workspaceID, year, month)
}
func (m *mockBudgetRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Budget, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockBudgetRepo) List(ctx context.Context, workspaceID uuid.UUID) ([]models.Budget, error) {
	return m.listFn(ctx, workspaceID)
}
func (m *mockBudgetRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}
func (m *mockBudgetRepo) UpsertCategory(ctx context.Context, budgetID, categoryID uuid.UUID, limit float64) error {
	return m.upsertCategoryFn(ctx, budgetID, categoryID, limit)
}
func (m *mockBudgetRepo) DeleteCategory(ctx context.Context, budgetID, categoryID uuid.UUID) error {
	return m.deleteCategoryFn(ctx, budgetID, categoryID)
}
func (m *mockBudgetRepo) ListCategories(ctx context.Context, budgetID uuid.UUID) ([]models.BudgetCategory, error) {
	return m.listCategoriesFn(ctx, budgetID)
}
func (m *mockBudgetRepo) CategorySpending(ctx context.Context, budgetID, workspaceID uuid.UUID, year, month int32) ([]models.BudgetCategorySpending, error) {
	return m.categorySpendingFn(ctx, budgetID, workspaceID, year, month)
}

func TestBudgetService_Upsert_Validation(t *testing.T) {
	wsID := uuid.New()
	now := time.Now().UTC()
	budgetID := uuid.New()

	tests := []struct {
		name    string
		params  UpsertBudgetParams
		wantErr bool
		errType error
	}{
		{name: "success", params: UpsertBudgetParams{WorkspaceID: wsID, Year: 2026, Month: 5, TotalLimit: 3000000}},
		{name: "zero limit", params: UpsertBudgetParams{WorkspaceID: wsID, Year: 2026, Month: 5, TotalLimit: 0}, wantErr: true, errType: apperror.ErrInvalidInput},
		{name: "invalid month 0", params: UpsertBudgetParams{WorkspaceID: wsID, Year: 2026, Month: 0, TotalLimit: 100}, wantErr: true, errType: apperror.ErrInvalidInput},
		{name: "invalid month 13", params: UpsertBudgetParams{WorkspaceID: wsID, Year: 2026, Month: 13, TotalLimit: 100}, wantErr: true, errType: apperror.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockBudgetRepo{
				upsertFn: func(_ context.Context, p repositories.UpsertBudgetParams) (models.Budget, error) {
					return models.Budget{ID: budgetID, WorkspaceID: p.WorkspaceID, Year: p.Year, Month: p.Month, TotalLimit: p.TotalLimit, CreatedAt: now, UpdatedAt: now}, nil
				},
				upsertCategoryFn: func(_ context.Context, _, _ uuid.UUID, _ float64) error { return nil },
			}
			svc := NewBudgetService(repo)
			_, err := svc.Upsert(context.Background(), tt.params)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestBudgetService_GetWithProgress(t *testing.T) {
	wsID := uuid.New()
	budgetID := uuid.New()
	catID := uuid.New()
	now := time.Now().UTC()

	budget := models.Budget{ID: budgetID, WorkspaceID: wsID, Year: 2026, Month: 5, TotalLimit: 3000000, CreatedAt: now, UpdatedAt: now}
	cats := []models.BudgetCategory{
		{BudgetID: budgetID, CategoryID: catID, CategoryName: "Alimentacion", LimitAmount: 500000},
	}
	spending := []models.BudgetCategorySpending{
		{CategoryID: catID, LimitAmount: 500000, Spent: 320000},
	}

	repo := &mockBudgetRepo{
		getByYearMonthFn: func(_ context.Context, _ uuid.UUID, _, _ int16) (models.Budget, error) { return budget, nil },
		listCategoriesFn: func(_ context.Context, _ uuid.UUID) ([]models.BudgetCategory, error) { return cats, nil },
		categorySpendingFn: func(_ context.Context, _, _ uuid.UUID, _, _ int32) ([]models.BudgetCategorySpending, error) {
			return spending, nil
		},
	}
	svc := NewBudgetService(repo)

	view, err := svc.GetWithProgress(context.Background(), wsID, 2026, 5)
	assert.NoError(t, err)
	assert.Equal(t, float64(3000000), view.TotalLimit)
	assert.Equal(t, float64(320000), view.TotalSpent)
	assert.Equal(t, float64(2680000), view.Remaining)
	assert.Len(t, view.Categories, 1)
	assert.Equal(t, float64(180000), view.Categories[0].Remaining)
}
