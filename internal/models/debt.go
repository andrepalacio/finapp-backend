package models

import (
	"time"

	"github.com/google/uuid"
)

type Debt struct {
	ID               uuid.UUID `json:"id"`
	WorkspaceID      uuid.UUID `json:"workspace_id"`
	Name             string    `json:"name"`
	Lender           string    `json:"lender,omitempty"`
	Principal        float64   `json:"principal"`
	Rate             float64   `json:"rate"`
	RateType         string    `json:"rate_type"`
	Installments     int32     `json:"installments"`
	FirstPaymentDate time.Time `json:"first_payment_date"`
	Notes            string    `json:"notes,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type DebtPayment struct {
	ID        uuid.UUID `json:"id"`
	DebtID    uuid.UUID `json:"debt_id"`
	Period    int32     `json:"period"`
	Amount    float64   `json:"amount"`
	PaidAt    time.Time `json:"paid_at"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type DebtScheduleInstallment struct {
	Period      int32      `json:"period"`
	DueDate     time.Time  `json:"due_date"`
	Payment     float64    `json:"payment"`
	Principal   float64    `json:"principal"`
	Interest    float64    `json:"interest"`
	Balance     float64    `json:"balance"`
	Status      string     `json:"status"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	PaidAmount  *float64   `json:"paid_amount,omitempty"`
}
