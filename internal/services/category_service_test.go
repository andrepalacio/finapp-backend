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

type mockCategoryRepo struct {
	createFn         func(ctx context.Context, p repositories.CreateCategoryParams) (models.Category, error)
	getByIDFn        func(ctx context.Context, id uuid.UUID) (models.Category, error)
	listForWorkspaceFn func(ctx context.Context, workspaceID uuid.UUID) ([]models.Category, error)
	updateFn         func(ctx context.Context, p repositories.UpdateCategoryParams) (models.Category, error)
	deleteFn         func(ctx context.Context, id, workspaceID uuid.UUID) error
}

func (m *mockCategoryRepo) Create(ctx context.Context, p repositories.CreateCategoryParams) (models.Category, error) {
	return m.createFn(ctx, p)
}
func (m *mockCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (models.Category, error) {
	return m.getByIDFn(ctx, id)
}
func (m *mockCategoryRepo) ListForWorkspace(ctx context.Context, workspaceID uuid.UUID) ([]models.Category, error) {
	return m.listForWorkspaceFn(ctx, workspaceID)
}
func (m *mockCategoryRepo) Update(ctx context.Context, p repositories.UpdateCategoryParams) (models.Category, error) {
	return m.updateFn(ctx, p)
}
func (m *mockCategoryRepo) Delete(ctx context.Context, id, workspaceID uuid.UUID) error {
	return m.deleteFn(ctx, id, workspaceID)
}

func TestCategoryService_Create(t *testing.T) {
	wsID := uuid.New()
	now := time.Now().UTC()

	tests := []struct {
		name    string
		params  CreateCategoryParams
		wantErr bool
		errType error
	}{
		{name: "success", params: CreateCategoryParams{WorkspaceID: wsID, Name: "Gym", Type: "expense"}},
		{name: "invalid type", params: CreateCategoryParams{WorkspaceID: wsID, Name: "X", Type: "bad"}, wantErr: true, errType: apperror.ErrInvalidInput},
		{name: "empty name", params: CreateCategoryParams{WorkspaceID: wsID, Type: "income"}, wantErr: true, errType: apperror.ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCategoryRepo{
				createFn: func(_ context.Context, p repositories.CreateCategoryParams) (models.Category, error) {
					wsid := p.WorkspaceID
					return models.Category{ID: uuid.New(), WorkspaceID: &wsid, Name: p.Name, Type: p.Type, CreatedAt: now}, nil
				},
			}
			svc := NewCategoryService(repo)
			cat, err := svc.Create(context.Background(), tt.params)
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.errType)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.params.Name, cat.Name)
		})
	}
}

func TestCategoryService_Delete_SystemCategoryBlocked(t *testing.T) {
	wsID := uuid.New()
	catID := uuid.New()
	systemCat := models.Category{ID: catID, Name: "Alimentacion", Type: "expense", IsSystem: true}

	repo := &mockCategoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Category, error) { return systemCat, nil },
	}
	svc := NewCategoryService(repo)

	err := svc.Delete(context.Background(), catID, wsID)
	assert.ErrorIs(t, err, apperror.ErrForbidden)
}

func TestCategoryService_Update_SystemCategoryBlocked(t *testing.T) {
	wsID := uuid.New()
	catID := uuid.New()
	systemCat := models.Category{ID: catID, Name: "Salario", Type: "income", IsSystem: true}

	repo := &mockCategoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Category, error) { return systemCat, nil },
	}
	svc := NewCategoryService(repo)

	_, err := svc.Update(context.Background(), UpdateCategoryParams{ID: catID, WorkspaceID: wsID, Name: "Renamed", Type: "income"})
	assert.ErrorIs(t, err, apperror.ErrForbidden)
}

func TestCategoryService_Update_WrongWorkspace(t *testing.T) {
	wsID := uuid.New()
	otherWS := uuid.New()
	catID := uuid.New()
	cat := models.Category{ID: catID, WorkspaceID: &wsID, Name: "Custom", Type: "expense", IsSystem: false}

	repo := &mockCategoryRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (models.Category, error) { return cat, nil },
	}
	svc := NewCategoryService(repo)

	_, err := svc.Update(context.Background(), UpdateCategoryParams{ID: catID, WorkspaceID: otherWS, Name: "New", Type: "expense"})
	assert.ErrorIs(t, err, apperror.ErrForbidden)
}
