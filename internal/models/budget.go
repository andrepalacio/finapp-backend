package models

import (
	"time"

	"github.com/google/uuid"
)

type Budget struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Year        int16     `json:"year"`
	Month       int16     `json:"month"`
	TotalLimit  float64   `json:"total_limit"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type BudgetCategory struct {
	BudgetID     uuid.UUID `json:"budget_id"`
	CategoryID   uuid.UUID `json:"category_id"`
	CategoryName string    `json:"category_name"`
	CategoryIcon string    `json:"category_icon,omitempty"`
	LimitAmount  float64   `json:"limit_amount"`
}

type BudgetCategorySpending struct {
	CategoryID  uuid.UUID `json:"category_id"`
	LimitAmount float64   `json:"limit_amount"`
	Spent       float64   `json:"spent"`
}
