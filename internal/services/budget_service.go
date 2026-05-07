package services

import (
	"context"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
)

type BudgetRepository interface {
	Upsert(ctx context.Context, p repositories.UpsertBudgetParams) (models.Budget, error)
	GetByYearMonth(ctx context.Context, workspaceID uuid.UUID, year, month int16) (models.Budget, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.Budget, error)
	List(ctx context.Context, workspaceID uuid.UUID) ([]models.Budget, error)
	Delete(ctx context.Context, id uuid.UUID) error
	UpsertCategory(ctx context.Context, budgetID, categoryID uuid.UUID, limit float64) error
	DeleteCategory(ctx context.Context, budgetID, categoryID uuid.UUID) error
	ListCategories(ctx context.Context, budgetID uuid.UUID) ([]models.BudgetCategory, error)
	CategorySpending(ctx context.Context, budgetID, workspaceID uuid.UUID, year, month int32) ([]models.BudgetCategorySpending, error)
}

type BudgetService struct {
	repo BudgetRepository
}

func NewBudgetService(repo BudgetRepository) *BudgetService {
	return &BudgetService{repo: repo}
}

type BudgetCategoryInput struct {
	CategoryID  uuid.UUID `json:"category_id"`
	LimitAmount float64   `json:"limit_amount"`
}

type BudgetCategoryProgressView struct {
	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`
	CategoryIcon string    `json:"category_icon,omitempty"`
	LimitAmount  float64   `json:"limit_amount"`
	Spent        float64   `json:"spent"`
	Remaining    float64   `json:"remaining"`
}

type BudgetView struct {
	ID          uuid.UUID                    `json:"id"`
	WorkspaceID uuid.UUID                    `json:"workspace_id"`
	Year        int16                        `json:"year"`
	Month       int16                        `json:"month"`
	TotalLimit  float64                      `json:"total_limit"`
	TotalSpent  float64                      `json:"total_spent,omitempty"`
	Remaining   float64                      `json:"remaining,omitempty"`
	Categories  []BudgetCategoryProgressView `json:"categories,omitempty"`
	CreatedAt   string                       `json:"created_at"`
	UpdatedAt   string                       `json:"updated_at"`
}

func toBudgetView(b models.Budget) BudgetView {
	return BudgetView{
		ID:          b.ID,
		WorkspaceID: b.WorkspaceID,
		Year:        b.Year,
		Month:       b.Month,
		TotalLimit:  b.TotalLimit,
		CreatedAt:   b.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   b.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type UpsertBudgetParams struct {
	WorkspaceID uuid.UUID
	Year        int16
	Month       int16
	TotalLimit  float64
	Categories  []BudgetCategoryInput
}

func (s *BudgetService) Upsert(ctx context.Context, p UpsertBudgetParams) (BudgetView, error) {
	if p.TotalLimit <= 0 {
		return BudgetView{}, apperror.ErrInvalidInput
	}
	if p.Month < 1 || p.Month > 12 {
		return BudgetView{}, apperror.ErrInvalidInput
	}

	budget, err := s.repo.Upsert(ctx, repositories.UpsertBudgetParams{
		WorkspaceID: p.WorkspaceID,
		Year:        p.Year,
		Month:       p.Month,
		TotalLimit:  p.TotalLimit,
	})
	if err != nil {
		return BudgetView{}, err
	}

	for _, cat := range p.Categories {
		if cat.LimitAmount <= 0 {
			return BudgetView{}, apperror.ErrInvalidInput
		}
		if err := s.repo.UpsertCategory(ctx, budget.ID, cat.CategoryID, cat.LimitAmount); err != nil {
			return BudgetView{}, err
		}
	}

	return toBudgetView(budget), nil
}

func (s *BudgetService) List(ctx context.Context, workspaceID uuid.UUID) ([]BudgetView, error) {
	budgets, err := s.repo.List(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]BudgetView, len(budgets))
	for i, b := range budgets {
		out[i] = toBudgetView(b)
	}
	return out, nil
}

func (s *BudgetService) GetWithProgress(ctx context.Context, workspaceID uuid.UUID, year, month int16) (BudgetView, error) {
	budget, err := s.repo.GetByYearMonth(ctx, workspaceID, year, month)
	if err != nil {
		return BudgetView{}, err
	}

	cats, err := s.repo.ListCategories(ctx, budget.ID)
	if err != nil {
		return BudgetView{}, err
	}

	spending, err := s.repo.CategorySpending(ctx, budget.ID, workspaceID, int32(year), int32(month))
	if err != nil {
		return BudgetView{}, err
	}

	spendMap := make(map[uuid.UUID]float64, len(spending))
	for _, s := range spending {
		spendMap[s.CategoryID] = s.Spent
	}

	var totalSpent float64
	catViews := make([]BudgetCategoryProgressView, len(cats))
	for i, cat := range cats {
		spent := spendMap[cat.CategoryID]
		totalSpent += spent
		catViews[i] = BudgetCategoryProgressView{
			CategoryID:   cat.CategoryID,
			CategoryName: cat.CategoryName,
			CategoryIcon: cat.CategoryIcon,
			LimitAmount:  cat.LimitAmount,
			Spent:        spent,
			Remaining:    cat.LimitAmount - spent,
		}
	}

	view := toBudgetView(budget)
	view.TotalSpent = totalSpent
	view.Remaining = budget.TotalLimit - totalSpent
	view.Categories = catViews
	return view, nil
}

func (s *BudgetService) Delete(ctx context.Context, workspaceID uuid.UUID, year, month int16) error {
	budget, err := s.repo.GetByYearMonth(ctx, workspaceID, year, month)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, budget.ID)
}

func (s *BudgetService) UpsertCategory(ctx context.Context, workspaceID uuid.UUID, year, month int16, cat BudgetCategoryInput) error {
	if cat.LimitAmount <= 0 {
		return apperror.ErrInvalidInput
	}
	budget, err := s.repo.GetByYearMonth(ctx, workspaceID, year, month)
	if err != nil {
		return err
	}
	return s.repo.UpsertCategory(ctx, budget.ID, cat.CategoryID, cat.LimitAmount)
}

func (s *BudgetService) DeleteCategory(ctx context.Context, workspaceID uuid.UUID, year, month int16, categoryID uuid.UUID) error {
	budget, err := s.repo.GetByYearMonth(ctx, workspaceID, year, month)
	if err != nil {
		return err
	}
	return s.repo.DeleteCategory(ctx, budget.ID, categoryID)
}
