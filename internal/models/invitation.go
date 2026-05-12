package models

import (
	"time"

	"github.com/google/uuid"
)

type WorkspaceInvitation struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Email       string
	Role        string
	Token       uuid.UUID
	Status      string
	InvitedBy   uuid.UUID
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

const (
	InvitationStatusPending   = "pending"
	InvitationStatusAccepted  = "accepted"
	InvitationStatusCancelled = "cancelled"
)
