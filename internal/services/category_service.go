package services

import (
	"context"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
)

type CategoryRepository interface {
	Create(ctx context.Context, p repositories.CreateCategoryParams) (models.Category, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.Category, error)
	ListForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Category, error)
	Update(ctx context.Context, p repositories.UpdateCategoryParams) (models.Category, error)
	Delete(ctx context.Context, id, workspaceID uuid.UUID) error
}

type CategoryService struct {
	repo CategoryRepository
}

func NewCategoryService(repo CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

type CategoryView struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID *uuid.UUID `json:"workspace_id,omitempty"`
	Name        string     `json:"name"`
	Icon        string     `json:"icon,omitempty"`
	Color       string     `json:"color,omitempty"`
	Type        string     `json:"type"`
	IsSystem    bool       `json:"is_system"`
	CreatedAt   string     `json:"created_at"`
}

func toCategoryView(c models.Category) CategoryView {
	return CategoryView{
		ID:          c.ID,
		WorkspaceID: c.WorkspaceID,
		Name:        c.Name,
		Icon:        c.Icon,
		Color:       c.Color,
		Type:        c.Type,
		IsSystem:    c.IsSystem,
		CreatedAt:   c.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type CreateCategoryParams struct {
	WorkspaceID uuid.UUID
	Name        string
	Icon        string
	Color       string
	Type        string
}

var validCategoryTypes = map[string]bool{"expense": true, "income": true, "both": true}

func (s *CategoryService) Create(ctx context.Context, p CreateCategoryParams) (CategoryView, error) {
	if p.Name == "" || !validCategoryTypes[p.Type] {
		return CategoryView{}, apperror.ErrInvalidInput
	}
	cat, err := s.repo.Create(ctx, repositories.CreateCategoryParams{
		WorkspaceID: p.WorkspaceID,
		Name:        p.Name,
		Icon:        p.Icon,
		Color:       p.Color,
		Type:        p.Type,
	})
	if err != nil {
		return CategoryView{}, err
	}
	return toCategoryView(cat), nil
}

func (s *CategoryService) ListForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]CategoryView, error) {
	cats, err := s.repo.ListForWorkspace(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]CategoryView, len(cats))
	for i, c := range cats {
		out[i] = toCategoryView(c)
	}
	return out, nil
}

type UpdateCategoryParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Icon        string
	Color       string
	Type        string
}

func (s *CategoryService) Update(ctx context.Context, p UpdateCategoryParams) (CategoryView, error) {
	existing, err := s.repo.GetByID(ctx, p.ID)
	if err != nil {
		return CategoryView{}, err
	}
	if existing.IsSystem {
		return CategoryView{}, apperror.ErrForbidden
	}
	if existing.WorkspaceID == nil || *existing.WorkspaceID != p.WorkspaceID {
		return CategoryView{}, apperror.ErrForbidden
	}
	if p.Name == "" {
		p.Name = existing.Name
	}
	if p.Type == "" {
		p.Type = existing.Type
	}
	if !validCategoryTypes[p.Type] {
		return CategoryView{}, apperror.ErrInvalidInput
	}

	cat, err := s.repo.Update(ctx, repositories.UpdateCategoryParams{
		ID:    p.ID,
		Name:  p.Name,
		Icon:  p.Icon,
		Color: p.Color,
		Type:  p.Type,
	})
	if err != nil {
		return CategoryView{}, err
	}
	return toCategoryView(cat), nil
}

func (s *CategoryService) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.IsSystem {
		return apperror.ErrForbidden
	}
	if existing.WorkspaceID == nil || *existing.WorkspaceID != workspaceID {
		return apperror.ErrForbidden
	}
	return s.repo.Delete(ctx, id, workspaceID)
}
