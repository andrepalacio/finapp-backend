// Package contracts holds generic interfaces describing method shapes
// already shared by multiple repository interfaces under internal/services.
// They are satisfied automatically by existing repositories (Go structural
// typing) — nothing under internal/ needs to change to use them.
package contracts

import (
	"context"

	"github.com/google/uuid"
)

// Reader is the GetByID(ctx, id) shape shared by every repository interface
// in internal/services (Budget, Debt, Transaction, Category, Savings, User).
type Reader[T any] interface {
	GetByID(ctx context.Context, id uuid.UUID) (T, error)
}

// WorkspaceLister is the List(ctx, workspaceID) shape shared by
// BudgetRepository, DebtRepository and SavingsRepository.
type WorkspaceLister[T any] interface {
	List(ctx context.Context, workspaceID uuid.UUID) ([]T, error)
}
