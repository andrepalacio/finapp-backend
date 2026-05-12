package models

import (
	"time"

	"github.com/google/uuid"
)

type SavingsGoal struct {
	ID           uuid.UUID  `json:"id"`
	WorkspaceID  uuid.UUID  `json:"workspace_id"`
	Name         string     `json:"name"`
	TargetAmount float64    `json:"target_amount"`
	Deadline     *time.Time `json:"deadline,omitempty"`
	Notes        string     `json:"notes,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type SavingsContribution struct {
	ID            uuid.UUID `json:"id"`
	GoalID        uuid.UUID `json:"goal_id"`
	Amount        float64   `json:"amount"`
	ContributedAt time.Time `json:"contributed_at"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
