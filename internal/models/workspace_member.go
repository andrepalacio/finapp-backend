package models

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

type WorkspaceMember struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Role        string
	JoinedAt    time.Time
}

type WorkspaceMemberWithUser struct {
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Role        string
	JoinedAt    time.Time
	Name        string
	Email       string
}
