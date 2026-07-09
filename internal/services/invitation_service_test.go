package services

import (
	"context"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockInvitationRepo struct {
	createFn       func(ctx context.Context, workspaceID uuid.UUID, email, role string, invitedBy uuid.UUID, expiresAt time.Time) (models.WorkspaceInvitation, error)
	getByTokenFn   func(ctx context.Context, token uuid.UUID) (models.WorkspaceInvitation, error)
	getByIDFn      func(ctx context.Context, id uuid.UUID) (models.WorkspaceInvitation, error)
	listPendingFn  func(ctx context.Context, workspaceID uuid.UUID) ([]models.WorkspaceInvitation, error)
	updateStatusFn func(ctx context.Context, id uuid.UUID, status string) (models.WorkspaceInvitation, error)
}

func (m *mockInvitationRepo) Create(ctx context.Context, workspaceID uuid.UUID, email, role string, invitedBy uuid.UUID, expiresAt time.Time) (models.WorkspaceInvitation, error) {
	return m.createFn(ctx, workspaceID, email, role, invitedBy, expiresAt)
}
func (m *mockInvitationRepo) GetByToken(ctx context.Context, token uuid.UUID) (models.WorkspaceInvitation, error) {
	return m.getByTokenFn(ctx, token)
}
func (m *mockInvitationRepo) GetByID(ctx context.Context, id uuid.UUID) (models.WorkspaceInvitation, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockInvitationRepo) ListPending(ctx context.Context, workspaceID uuid.UUID) ([]models.WorkspaceInvitation, error) {
	return m.listPendingFn(ctx, workspaceID)
}
func (m *mockInvitationRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (models.WorkspaceInvitation, error) {
	return m.updateStatusFn(ctx, id, status)
}

type mockInvitationWorkspaceRepo struct {
	getByIDFn   func(ctx context.Context, id uuid.UUID) (models.Workspace, error)
	getMemberFn func(ctx context.Context, workspaceID, userID uuid.UUID) (models.WorkspaceMember, error)
	addMemberFn func(ctx context.Context, workspaceID, userID uuid.UUID, role string) error
}

func (m *mockInvitationWorkspaceRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Workspace, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockInvitationWorkspaceRepo) GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (models.WorkspaceMember, error) {
	return m.getMemberFn(ctx, workspaceID, userID)
}
func (m *mockInvitationWorkspaceRepo) AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role string) error {
	return m.addMemberFn(ctx, workspaceID, userID, role)
}

type mockInvitationUserRepo struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (models.User, error)
}

func (m *mockInvitationUserRepo) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return m.getByIDFn(ctx, id)
}

func TestInvitationService_Send(t *testing.T) {
	wsID := uuid.New()
	ownerID := uuid.New()
	adminID := uuid.New()
	otherID := uuid.New()
	ws := models.Workspace{ID: wsID, OwnerID: ownerID}

	tests := []struct {
		name     string
		params   SendInvitationParams
		wantErr  bool
		errType  error
		wantRole string
	}{
		{
			name:     "owner invites",
			params:   SendInvitationParams{WorkspaceID: wsID, Email: "a@b.com", Role: models.RoleMember, InviterID: ownerID},
			wantRole: models.RoleMember,
		},
		{
			name:     "invalid role defaults to member",
			params:   SendInvitationParams{WorkspaceID: wsID, Email: "a@b.com", Role: "bogus", InviterID: ownerID},
			wantRole: models.RoleMember,
		},
		{
			name:    "empty email",
			params:  SendInvitationParams{WorkspaceID: wsID, Email: "", InviterID: ownerID},
			wantErr: true,
			errType: apperror.ErrInvalidInput,
		},
		{
			name:    "non-member cannot invite",
			params:  SendInvitationParams{WorkspaceID: wsID, Email: "a@b.com", InviterID: otherID},
			wantErr: true,
			errType: apperror.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockInvitationRepo{
				createFn: func(_ context.Context, workspaceID uuid.UUID, email, role string, invitedBy uuid.UUID, expiresAt time.Time) (models.WorkspaceInvitation, error) {
					return models.WorkspaceInvitation{
						ID: uuid.New(), WorkspaceID: workspaceID, Email: email, Role: role,
						Token: uuid.New(), Status: models.InvitationStatusPending,
						InvitedBy: invitedBy, ExpiresAt: expiresAt, CreatedAt: time.Now().UTC(),
					}, nil
				},
			}
			wsRepo := &mockInvitationWorkspaceRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Workspace, error) { return ws, nil },
				getMemberFn: func(_ context.Context, _, userID uuid.UUID) (models.WorkspaceMember, error) {
					if userID == adminID {
						return models.WorkspaceMember{Role: models.RoleAdmin}, nil
					}
					return models.WorkspaceMember{}, apperror.ErrNotFound
				},
			}
			userRepo := &mockInvitationUserRepo{}
			svc := NewInvitationService(repo, wsRepo, userRepo)

			inv, err := svc.Send(context.Background(), tt.params)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantRole, inv.Role)
			assert.Equal(t, tt.params.Email, inv.Email)
		})
	}
}

func TestInvitationService_Accept(t *testing.T) {
	wsID := uuid.New()
	userID := uuid.New()
	invID := uuid.New()
	token := uuid.New()
	user := models.User{ID: userID, Email: "a@b.com"}

	tests := []struct {
		name    string
		inv     models.WorkspaceInvitation
		wantErr bool
		errType error
	}{
		{
			name: "success",
			inv: models.WorkspaceInvitation{
				ID: invID, WorkspaceID: wsID, Email: "a@b.com", Role: models.RoleMember,
				Token: token, Status: models.InvitationStatusPending, ExpiresAt: time.Now().Add(time.Hour),
			},
		},
		{
			name: "already used",
			inv: models.WorkspaceInvitation{
				ID: invID, Email: "a@b.com", Token: token,
				Status: models.InvitationStatusAccepted, ExpiresAt: time.Now().Add(time.Hour),
			},
			wantErr: true,
			errType: apperror.ErrInvalidInput,
		},
		{
			name: "expired",
			inv: models.WorkspaceInvitation{
				ID: invID, Email: "a@b.com", Token: token,
				Status: models.InvitationStatusPending, ExpiresAt: time.Now().Add(-time.Hour),
			},
			wantErr: true,
			errType: apperror.ErrInvalidInput,
		},
		{
			name: "email mismatch",
			inv: models.WorkspaceInvitation{
				ID: invID, Email: "other@b.com", Token: token,
				Status: models.InvitationStatusPending, ExpiresAt: time.Now().Add(time.Hour),
			},
			wantErr: true,
			errType: apperror.ErrForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockInvitationRepo{
				getByTokenFn: func(_ context.Context, _ uuid.UUID) (models.WorkspaceInvitation, error) { return tt.inv, nil },
				updateStatusFn: func(_ context.Context, id uuid.UUID, status string) (models.WorkspaceInvitation, error) {
					updated := tt.inv
					updated.Status = status
					return updated, nil
				},
			}
			wsRepo := &mockInvitationWorkspaceRepo{
				getMemberFn: func(_ context.Context, _, _ uuid.UUID) (models.WorkspaceMember, error) {
					return models.WorkspaceMember{}, apperror.ErrNotFound
				},
				addMemberFn: func(_ context.Context, _, _ uuid.UUID, _ string) error { return nil },
			}
			userRepo := &mockInvitationUserRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (models.User, error) { return user, nil },
			}
			svc := NewInvitationService(repo, wsRepo, userRepo)

			result, err := svc.Accept(context.Background(), token, userID)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, models.InvitationStatusAccepted, result.Status)
		})
	}
}

func TestInvitationService_Accept_AlreadyMemberIsIdempotent(t *testing.T) {
	wsID := uuid.New()
	userID := uuid.New()
	token := uuid.New()
	inv := models.WorkspaceInvitation{
		ID: uuid.New(), WorkspaceID: wsID, Email: "a@b.com", Role: models.RoleMember,
		Token: token, Status: models.InvitationStatusPending, ExpiresAt: time.Now().Add(time.Hour),
	}
	addMemberCalled := false

	repo := &mockInvitationRepo{
		getByTokenFn: func(_ context.Context, _ uuid.UUID) (models.WorkspaceInvitation, error) { return inv, nil },
		updateStatusFn: func(_ context.Context, id uuid.UUID, status string) (models.WorkspaceInvitation, error) {
			updated := inv
			updated.Status = status
			return updated, nil
		},
	}
	wsRepo := &mockInvitationWorkspaceRepo{
		getMemberFn: func(_ context.Context, _, _ uuid.UUID) (models.WorkspaceMember, error) {
			return models.WorkspaceMember{Role: models.RoleMember}, nil
		},
		addMemberFn: func(_ context.Context, _, _ uuid.UUID, _ string) error {
			addMemberCalled = true
			return nil
		},
	}
	userRepo := &mockInvitationUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.User, error) { return models.User{ID: userID, Email: "a@b.com"}, nil },
	}
	svc := NewInvitationService(repo, wsRepo, userRepo)

	_, err := svc.Accept(context.Background(), token, userID)
	assert.NoError(t, err)
	assert.False(t, addMemberCalled)
}

func TestInvitationService_Cancel(t *testing.T) {
	wsID := uuid.New()
	ownerID := uuid.New()
	otherID := uuid.New()
	invID := uuid.New()
	ws := models.Workspace{ID: wsID, OwnerID: ownerID}

	tests := []struct {
		name        string
		inv         models.WorkspaceInvitation
		requesterID uuid.UUID
		wantErr     bool
		errType     error
	}{
		{
			name:        "owner cancels pending",
			inv:         models.WorkspaceInvitation{ID: invID, WorkspaceID: wsID, Status: models.InvitationStatusPending},
			requesterID: ownerID,
		},
		{
			name:        "non-member forbidden",
			inv:         models.WorkspaceInvitation{ID: invID, WorkspaceID: wsID, Status: models.InvitationStatusPending},
			requesterID: otherID,
			wantErr:     true,
			errType:     apperror.ErrForbidden,
		},
		{
			name:        "not pending",
			inv:         models.WorkspaceInvitation{ID: invID, WorkspaceID: wsID, Status: models.InvitationStatusAccepted},
			requesterID: ownerID,
			wantErr:     true,
			errType:     apperror.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockInvitationRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (models.WorkspaceInvitation, error) { return tt.inv, nil },
				updateStatusFn: func(_ context.Context, id uuid.UUID, status string) (models.WorkspaceInvitation, error) {
					updated := tt.inv
					updated.Status = status
					return updated, nil
				},
			}
			wsRepo := &mockInvitationWorkspaceRepo{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Workspace, error) { return ws, nil },
				getMemberFn: func(_ context.Context, _, _ uuid.UUID) (models.WorkspaceMember, error) {
					return models.WorkspaceMember{}, apperror.ErrNotFound
				},
			}
			userRepo := &mockInvitationUserRepo{}
			svc := NewInvitationService(repo, wsRepo, userRepo)

			err := svc.Cancel(context.Background(), invID, wsID, tt.requesterID)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestInvitationService_ListPending(t *testing.T) {
	wsID := uuid.New()
	invitations := []models.WorkspaceInvitation{
		{ID: uuid.New(), WorkspaceID: wsID, Email: "a@b.com", Status: models.InvitationStatusPending, ExpiresAt: time.Now(), CreatedAt: time.Now()},
		{ID: uuid.New(), WorkspaceID: wsID, Email: "c@d.com", Status: models.InvitationStatusPending, ExpiresAt: time.Now(), CreatedAt: time.Now()},
	}

	repo := &mockInvitationRepo{
		listPendingFn: func(_ context.Context, _ uuid.UUID) ([]models.WorkspaceInvitation, error) { return invitations, nil },
	}
	svc := NewInvitationService(repo, &mockInvitationWorkspaceRepo{}, &mockInvitationUserRepo{})

	out, err := svc.ListPending(context.Background(), wsID)
	assert.NoError(t, err)
	assert.Len(t, out, 2)
}
