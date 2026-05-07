package models

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID *uuid.UUID `json:"workspace_id,omitempty"`
	Name        string     `json:"name"`
	Icon        string     `json:"icon,omitempty"`
	Color       string     `json:"color,omitempty"`
	Type        string     `json:"type"`
	IsSystem    bool       `json:"is_system"`
	CreatedAt   time.Time  `json:"created_at"`
}
