package services

import (
	"context"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
)

type InvitationRepository interface {
	Create(ctx context.Context, workspaceID uuid.UUID, email, role string, invitedBy uuid.UUID, expiresAt time.Time) (models.WorkspaceInvitation, error)
	GetByToken(ctx context.Context, token uuid.UUID) (models.WorkspaceInvitation, error)
	GetByID(ctx context.Context, id uuid.UUID) (models.WorkspaceInvitation, error)
	ListPending(ctx context.Context, workspaceID uuid.UUID) ([]models.WorkspaceInvitation, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status string) (models.WorkspaceInvitation, error)
}

type InvitationWorkspaceRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.Workspace, error)
	GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (models.WorkspaceMember, error)
	AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role string) error
}

type InvitationUserRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (models.User, error)
}

type InvitationService struct {
	invRepo  InvitationRepository
	wsRepo   InvitationWorkspaceRepository
	userRepo InvitationUserRepository
}

func NewInvitationService(invRepo InvitationRepository, wsRepo InvitationWorkspaceRepository, userRepo InvitationUserRepository) *InvitationService {
	return &InvitationService{invRepo: invRepo, wsRepo: wsRepo, userRepo: userRepo}
}

type InvitationView struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	Token       uuid.UUID `json:"token"`
	Status      string    `json:"status"`
	InvitedBy   uuid.UUID `json:"invited_by"`
	ExpiresAt   string    `json:"expires_at"`
	CreatedAt   string    `json:"created_at"`
}

func toInvitationView(inv models.WorkspaceInvitation) InvitationView {
	return InvitationView{
		ID:          inv.ID,
		WorkspaceID: inv.WorkspaceID,
		Email:       inv.Email,
		Role:        inv.Role,
		Token:       inv.Token,
		Status:      inv.Status,
		InvitedBy:   inv.InvitedBy,
		ExpiresAt:   inv.ExpiresAt.UTC().Format(time.RFC3339),
		CreatedAt:   inv.CreatedAt.UTC().Format(time.RFC3339),
	}
}

type SendInvitationParams struct {
	WorkspaceID uuid.UUID
	Email       string
	Role        string
	InviterID   uuid.UUID
}

func (s *InvitationService) Send(ctx context.Context, p SendInvitationParams) (InvitationView, error) {
	if p.Email == "" {
		return InvitationView{}, apperror.ErrInvalidInput
	}
	if p.Role != models.RoleAdmin && p.Role != models.RoleMember {
		p.Role = models.RoleMember
	}

	ws, err := s.wsRepo.GetByID(ctx, p.WorkspaceID)
	if err != nil {
		return InvitationView{}, err
	}
	if ws.OwnerID != p.InviterID {
		// only owner can invite
		member, err := s.wsRepo.GetMember(ctx, p.WorkspaceID, p.InviterID)
		if err != nil || member.Role != models.RoleAdmin {
			return InvitationView{}, apperror.ErrForbidden
		}
	}

	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	inv, err := s.invRepo.Create(ctx, p.WorkspaceID, p.Email, p.Role, p.InviterID, expiresAt)
	if err != nil {
		return InvitationView{}, err
	}
	return toInvitationView(inv), nil
}

func (s *InvitationService) Accept(ctx context.Context, token uuid.UUID, userID uuid.UUID) (InvitationView, error) {
	inv, err := s.invRepo.GetByToken(ctx, token)
	if err != nil {
		return InvitationView{}, err
	}
	if inv.Status != models.InvitationStatusPending {
		return InvitationView{}, apperror.WithMessage(apperror.ErrInvalidInput, "invitation already used or cancelled")
	}
	if time.Now().After(inv.ExpiresAt) {
		return InvitationView{}, apperror.WithMessage(apperror.ErrInvalidInput, "invitation expired")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return InvitationView{}, err
	}
	if inv.Email != user.Email {
		return InvitationView{}, apperror.ErrForbidden
	}

	// idempotent: if already member, just accept the invitation
	_, memberErr := s.wsRepo.GetMember(ctx, inv.WorkspaceID, userID)
	if memberErr != nil {
		if err := s.wsRepo.AddMember(ctx, inv.WorkspaceID, userID, inv.Role); err != nil {
			return InvitationView{}, err
		}
	}

	updated, err := s.invRepo.UpdateStatus(ctx, inv.ID, models.InvitationStatusAccepted)
	if err != nil {
		return InvitationView{}, err
	}
	return toInvitationView(updated), nil
}

func (s *InvitationService) Cancel(ctx context.Context, invID uuid.UUID, workspaceID uuid.UUID, requesterID uuid.UUID) error {
	inv, err := s.invRepo.GetByID(ctx, invID)
	if err != nil {
		return err
	}
	if inv.WorkspaceID != workspaceID {
		return apperror.ErrNotFound
	}

	ws, err := s.wsRepo.GetByID(ctx, workspaceID)
	if err != nil {
		return err
	}
	if ws.OwnerID != requesterID {
		member, err := s.wsRepo.GetMember(ctx, workspaceID, requesterID)
		if err != nil || member.Role != models.RoleAdmin {
			return apperror.ErrForbidden
		}
	}

	if inv.Status != models.InvitationStatusPending {
		return apperror.WithMessage(apperror.ErrInvalidInput, "invitation is not pending")
	}
	_, err = s.invRepo.UpdateStatus(ctx, invID, models.InvitationStatusCancelled)
	return err
}

func (s *InvitationService) ListPending(ctx context.Context, workspaceID uuid.UUID) ([]InvitationView, error) {
	invitations, err := s.invRepo.ListPending(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	out := make([]InvitationView, len(invitations))
	for i, inv := range invitations {
		out[i] = toInvitationView(inv)
	}
	return out, nil
}
