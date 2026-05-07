package services

import (
	"context"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockWorkspaceRepo struct {
	createFn    func(ctx context.Context, p repositories.CreateWorkspaceParams) (models.Workspace, error)
	addMemberFn func(ctx context.Context, workspaceID, userID uuid.UUID, role string) error
	getByIDFn   func(ctx context.Context, id uuid.UUID) (models.Workspace, error)
	getMemberFn func(ctx context.Context, workspaceID, userID uuid.UUID) (models.WorkspaceMember, error)
	listByUserFn func(ctx context.Context, userID uuid.UUID) ([]models.Workspace, error)
	updateFn    func(ctx context.Context, id uuid.UUID, name, currency string) (models.Workspace, error)
	deleteFn    func(ctx context.Context, id uuid.UUID) error
}

func (m *mockWorkspaceRepo) Create(ctx context.Context, p repositories.CreateWorkspaceParams) (models.Workspace, error) {
	return m.createFn(ctx, p)
}
func (m *mockWorkspaceRepo) AddMember(ctx context.Context, workspaceID, userID uuid.UUID, role string) error {
	return m.addMemberFn(ctx, workspaceID, userID, role)
}
func (m *mockWorkspaceRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Workspace, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockWorkspaceRepo) GetMember(ctx context.Context, workspaceID, userID uuid.UUID) (models.WorkspaceMember, error) {
	return m.getMemberFn(ctx, workspaceID, userID)
}
func (m *mockWorkspaceRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.Workspace, error) {
	return m.listByUserFn(ctx, userID)
}
func (m *mockWorkspaceRepo) Update(ctx context.Context, id uuid.UUID, name, currency string) (models.Workspace, error) {
	return m.updateFn(ctx, id, name, currency)
}
func (m *mockWorkspaceRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.deleteFn(ctx, id)
}

func TestWorkspaceService_Create(t *testing.T) {
	now := time.Now().UTC()
	ownerID := uuid.New()
	wsID := uuid.New()

	tests := []struct {
		name    string
		params  CreateWorkspaceParams
		wantErr bool
		errType error
	}{
		{
			name:   "success with currency",
			params: CreateWorkspaceParams{Name: "Home", OwnerID: ownerID, Currency: "USD"},
		},
		{
			name:   "default currency COP",
			params: CreateWorkspaceParams{Name: "Work", OwnerID: ownerID},
		},
		{
			name:    "empty name",
			params:  CreateWorkspaceParams{OwnerID: ownerID},
			wantErr: true,
			errType: apperror.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockWorkspaceRepo{
				createFn: func(_ context.Context, p repositories.CreateWorkspaceParams) (models.Workspace, error) {
					currency := p.Currency
					if currency == "" {
						currency = "COP"
					}
					return models.Workspace{ID: wsID, Name: p.Name, OwnerID: p.OwnerID, Currency: currency, CreatedAt: now, UpdatedAt: now}, nil
				},
				addMemberFn: func(_ context.Context, _, _ uuid.UUID, _ string) error { return nil },
			}
			svc := NewWorkspaceService(repo)
			ws, err := svc.Create(context.Background(), tt.params)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.params.Name, ws.Name)
			if tt.params.Currency == "" {
				assert.Equal(t, "COP", ws.Currency)
			}
		})
	}
}

func TestWorkspaceService_Update_OwnerOnly(t *testing.T) {
	ownerID := uuid.New()
	otherID := uuid.New()
	wsID := uuid.New()
	now := time.Now().UTC()
	ws := models.Workspace{ID: wsID, Name: "Original", OwnerID: ownerID, Currency: "COP", CreatedAt: now, UpdatedAt: now}

	repo := &mockWorkspaceRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Workspace, error) { return ws, nil },
		updateFn:  func(_ context.Context, _ uuid.UUID, name, currency string) (models.Workspace, error) {
			return models.Workspace{ID: wsID, Name: name, OwnerID: ownerID, Currency: currency, CreatedAt: now, UpdatedAt: now}, nil
		},
	}
	svc := NewWorkspaceService(repo)

	_, err := svc.Update(context.Background(), UpdateWorkspaceParams{ID: wsID, UserID: otherID, Name: "New"})
	assert.ErrorIs(t, err, apperror.ErrForbidden)

	updated, err := svc.Update(context.Background(), UpdateWorkspaceParams{ID: wsID, UserID: ownerID, Name: "New"})
	assert.NoError(t, err)
	assert.Equal(t, "New", updated.Name)
}
