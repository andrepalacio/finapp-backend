package services

import (
	"context"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
)

type WorkspaceRepository interface {
	Create(ctx context.Context, p repositories.CreateWorkspaceParams) (models.Workspace, error)
	AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role string) error
	GetByID(ctx context.Context, id uuid.UUID) (models.Workspace, error)
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (models.WorkspaceMember, error)
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Workspace, error)
	Update(ctx context.Context, id uuid.UUID, name, currency string) (models.Workspace, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]models.WorkspaceMemberWithUser, error)
	RemoveMember(ctx context.Context, workspaceID, userID uuid.UUID) error
	UpdateMemberRole(ctx context.Context, workspaceID, userID uuid.UUID, role string) error
}

type WorkspaceService struct {
	repo WorkspaceRepository
}

func NewWorkspaceService(repo WorkspaceRepository) *WorkspaceService {
	return &WorkspaceService{repo: repo}
}

type WorkspaceView struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Currency  string    `json:"currency"`
	CreatedAt string    `json:"created_at"`
	UpdatedAt string    `json:"updated_at"`
}

func toWorkspaceView(w models.Workspace) WorkspaceView {
	return WorkspaceView{
		ID:        w.ID,
		Name:      w.Name,
		OwnerID:   w.OwnerID,
		Currency:  w.Currency,
		CreatedAt: w.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: w.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type CreateWorkspaceParams struct {
	Name     string
	OwnerID  uuid.UUID
	Currency string
}

func (s *WorkspaceService) Create(ctx context.Context, p CreateWorkspaceParams) (WorkspaceView, error) {
	if p.Name == "" {
		return WorkspaceView{}, apperror.ErrInvalidInput
	}
	if p.Currency == "" {
		p.Currency = "COP"
	}

	ws, err := s.repo.Create(ctx, repositories.CreateWorkspaceParams{
		Name:     p.Name,
		OwnerID:  p.OwnerID,
		Currency: p.Currency,
	})
	if err != nil {
		return WorkspaceView{}, err
	}

	if err := s.repo.AddMember(ctx, ws.ID, p.OwnerID, models.RoleOwner); err != nil {
		return WorkspaceView{}, err
	}

	return toWorkspaceView(ws), nil
}

func (s *WorkspaceService) GetByID(ctx context.Context, id uuid.UUID) (WorkspaceView, error) {
	ws, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return WorkspaceView{}, err
	}
	return toWorkspaceView(ws), nil
}

func (s *WorkspaceService) ListByUser(ctx context.Context, userID uuid.UUID) ([]WorkspaceView, error) {
	workspaces, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]WorkspaceView, len(workspaces))
	for i, ws := range workspaces {
		out[i] = toWorkspaceView(ws)
	}
	return out, nil
}

type UpdateWorkspaceParams struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Name     string
	Currency string
}

func (s *WorkspaceService) Update(ctx context.Context, p UpdateWorkspaceParams) (WorkspaceView, error) {
	ws, err := s.repo.GetByID(ctx, p.ID)
	if err != nil {
		return WorkspaceView{}, err
	}
	if ws.OwnerID != p.UserID {
		return WorkspaceView{}, apperror.ErrForbidden
	}
	if p.Name == "" {
		p.Name = ws.Name
	}
	if p.Currency == "" {
		p.Currency = ws.Currency
	}

	updated, err := s.repo.Update(ctx, p.ID, p.Name, p.Currency)
	if err != nil {
		return WorkspaceView{}, err
	}
	return toWorkspaceView(updated), nil
}

func (s *WorkspaceService) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ws, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if ws.OwnerID != userID {
		return apperror.ErrForbidden
	}
	return s.repo.Delete(ctx, id)
}

type MemberView struct {
	UserID   uuid.UUID `json:"user_id"`
	Role     string    `json:"role"`
	JoinedAt string    `json:"joined_at"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
}

func (s *WorkspaceService) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]MemberView, error) {
	members, err := s.repo.ListMembers(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]MemberView, len(members))
	for i, m := range members {
		out[i] = MemberView{
			UserID:   m.UserID,
			Role:     m.Role,
			JoinedAt: m.JoinedAt.UTC().Format(time.RFC3339),
			Name:     m.Name,
			Email:    m.Email,
		}
	}
	return out, nil
}

type UpdateMemberRoleParams struct {
	WorkspaceID uuid.UUID
	TargetID    uuid.UUID
	RequesterID uuid.UUID
	Role        string
}

func (s *WorkspaceService) UpdateMemberRole(ctx context.Context, p UpdateMemberRoleParams) error {
	if p.Role != models.RoleAdmin && p.Role != models.RoleMember {
		return apperror.ErrInvalidInput
	}
	ws, err := s.repo.GetByID(ctx, p.WorkspaceID)
	if err != nil {
		return err
	}
	if ws.OwnerID != p.RequesterID {
		return apperror.ErrForbidden
	}
	if ws.OwnerID == p.TargetID {
		return apperror.WithMessage(apperror.ErrForbidden, "cannot change owner role")
	}
	return s.repo.UpdateMemberRole(ctx, p.WorkspaceID, p.TargetID, p.Role)
}

func (s *WorkspaceService) RemoveMember(ctx context.Context, workspaceID, targetID, requesterID uuid.UUID) error {
	ws, err := s.repo.GetByID(ctx, workspaceID)
	if err != nil {
		return err
	}
	if ws.OwnerID == targetID {
		return apperror.WithMessage(apperror.ErrForbidden, "cannot remove owner")
	}
	if ws.OwnerID != requesterID {
		// admins can remove members
		member, err := s.repo.GetMember(ctx, workspaceID, requesterID)
		if err != nil || member.Role != models.RoleAdmin {
			return apperror.ErrForbidden
		}
	}
	return s.repo.RemoveMember(ctx, workspaceID, targetID)
}
