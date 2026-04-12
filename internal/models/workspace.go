package models

import (
	"time"

	"github.com/google/uuid"
)

type Workspace struct {
	ID        uuid.UUID
	Name      string
	OwnerID   uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}
