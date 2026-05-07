package models

import (
	"time"

	"github.com/google/uuid"
)

type Transaction struct {
	ID                uuid.UUID  `json:"id"`
	WorkspaceID       uuid.UUID  `json:"workspace_id"`
	UserID            uuid.UUID  `json:"user_id"`
	CategoryID        *uuid.UUID `json:"category_id,omitempty"`
	TransferID        *uuid.UUID `json:"transfer_id,omitempty"`
	Type              string     `json:"type"`
	TransferDirection string     `json:"transfer_direction,omitempty"`
	Amount            float64    `json:"amount"`
	Description       string     `json:"description,omitempty"`
	Date              time.Time  `json:"date"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type Transfer struct {
	ID              uuid.UUID `json:"id"`
	FromWorkspaceID uuid.UUID `json:"from_workspace_id"`
	ToWorkspaceID   uuid.UUID `json:"to_workspace_id"`
	Note            string    `json:"note,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type DailySummary struct {
	Date             time.Time `json:"date"`
	TotalExpense     float64   `json:"total_expense"`
	TotalIncome      float64   `json:"total_income"`
	TotalTransferOut float64   `json:"total_transfer_out"`
	TotalTransferIn  float64   `json:"total_transfer_in"`
	TransactionCount int32     `json:"transaction_count"`
}
