package contracts_test

import (
	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/pkg/contracts"
)

// Compile-time proof that existing repositories already satisfy the
// contracts, with no changes to internal/.
var (
	_ contracts.Reader[models.Budget]      = (*repositories.BudgetRepository)(nil)
	_ contracts.Reader[models.Debt]        = (*repositories.DebtRepository)(nil)
	_ contracts.Reader[models.Transaction] = (*repositories.TransactionRepository)(nil)
	_ contracts.Reader[models.Category]    = (*repositories.CategoryRepository)(nil)
	_ contracts.Reader[models.SavingsGoal] = (*repositories.SavingsRepository)(nil)
	_ contracts.Reader[models.User]        = (*repositories.UserRepository)(nil)

	_ contracts.WorkspaceLister[models.Budget]      = (*repositories.BudgetRepository)(nil)
	_ contracts.WorkspaceLister[models.Debt]        = (*repositories.DebtRepository)(nil)
	_ contracts.WorkspaceLister[models.SavingsGoal] = (*repositories.SavingsRepository)(nil)
)
