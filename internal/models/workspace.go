package models

import (
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID        uuid.UUID
	Name      string
	OwnerID   uuid.UUID
	Currency  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WorkspaceWithRole pairs a workspace with the requesting user's role in it.
type WorkspaceWithRole struct {
	Workspace
	Role string
}

