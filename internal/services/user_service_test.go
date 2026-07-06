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
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepository struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (models.User, error)
	updateFn  func(ctx context.Context, userID uuid.UUID, name, email string) (models.User, error)
}

func (m *mockUserRepository) Create(ctx context.Context, params repositories.CreateUserParams) (models.User, error) {
	return models.User{}, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (models.User, error) {
	return models.User{}, nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockUserRepository) Update(ctx context.Context, userID uuid.UUID, name, email string) (models.User, error) {
	return m.updateFn(ctx, userID, name, email)
}

func (m *mockUserRepository) UpdatePassword(ctx context.Context, userID uuid.UUID, passwordHash string) (models.User, error) {
	return models.User{}, nil
}

func (m *mockUserRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func TestUserService_GetProfile(t *testing.T) {
	now := time.Now().UTC()
	userID := uuid.New()

	tests := []struct {
		name    string
		userID  uuid.UUID
		mockFn  func(ctx context.Context, id uuid.UUID) (models.User, error)
		want    UserProfile
		wantErr bool
		errType error
	}{
		{
			name:   "user found",
			userID: userID,
			mockFn: func(ctx context.Context, id uuid.UUID) (models.User, error) {
				return models.User{
					ID:        id,
					Email:     "john@example.com",
					Name:      "John Doe",
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
			want: UserProfile{
				ID:        userID,
				Email:     "john@example.com",
				Name:      "John Doe",
				CreatedAt: now.Format("2006-01-02T15:04:05Z"),
				UpdatedAt: now.Format("2006-01-02T15:04:05Z"),
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			userID: userID,
			mockFn: func(ctx context.Context, id uuid.UUID) (models.User, error) {
				return models.User{}, apperror.ErrNotFound
			},
			wantErr: true,
			errType: apperror.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepository{getByIDFn: tt.mockFn}
			svc := NewUserService(repo, 10)

			got, err := svc.GetProfile(context.Background(), GetProfileParams{UserID: tt.userID})
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errType, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want.ID, got.ID)
			assert.Equal(t, tt.want.Email, got.Email)
			assert.Equal(t, tt.want.Name, got.Name)
		})
	}
}

func TestUserService_UpdateProfile(t *testing.T) {
	now := time.Now().UTC()
	userID := uuid.New()

	currentUser := models.User{
		ID:        userID,
		Email:     "john@example.com",
		Name:      "John Doe",
		CreatedAt: now,
		UpdatedAt: now,
	}
	defaultGetByID := func(_ context.Context, id uuid.UUID) (models.User, error) {
		return currentUser, nil
	}

	tests := []struct {
		name      string
		params    UpdateProfileParams
		getByIDFn func(ctx context.Context, id uuid.UUID) (models.User, error)
		mockFn    func(ctx context.Context, userID uuid.UUID, name, email string) (models.User, error)
		want      UserProfile
		wantErr   bool
		errType   error
	}{
		{
			name: "both name and email updated",
			params: UpdateProfileParams{
				UserID: userID,
				Name:   "Jane Doe",
				Email:  "jane@example.com",
			},
			getByIDFn: defaultGetByID,
			mockFn: func(_ context.Context, _ uuid.UUID, _, _ string) (models.User, error) {
				return models.User{ID: userID, Email: "jane@example.com", Name: "Jane Doe", CreatedAt: now, UpdatedAt: now}, nil
			},
			want:    UserProfile{ID: userID, Email: "jane@example.com", Name: "Jane Doe", CreatedAt: now.UTC().Format(time.RFC3339), UpdatedAt: now.UTC().Format(time.RFC3339)},
			wantErr: false,
		},
		{
			name: "name only — email preserved from current user",
			params: UpdateProfileParams{
				UserID: userID,
				Name:   "Jane Doe",
			},
			getByIDFn: defaultGetByID,
			mockFn: func(_ context.Context, _ uuid.UUID, name, email string) (models.User, error) {
				// email should be the current user's email (merged by service)
				assert.Equal(t, "john@example.com", email)
				return models.User{ID: userID, Email: email, Name: name, CreatedAt: now, UpdatedAt: now}, nil
			},
			want:    UserProfile{ID: userID, Email: "john@example.com", Name: "Jane Doe", CreatedAt: now.UTC().Format(time.RFC3339), UpdatedAt: now.UTC().Format(time.RFC3339)},
			wantErr: false,
		},
		{
			name: "email only — name preserved from current user",
			params: UpdateProfileParams{
				UserID: userID,
				Email:  "newemail@example.com",
			},
			getByIDFn: defaultGetByID,
			mockFn: func(_ context.Context, _ uuid.UUID, name, email string) (models.User, error) {
				assert.Equal(t, "John Doe", name)
				return models.User{ID: userID, Email: email, Name: name, CreatedAt: now, UpdatedAt: now}, nil
			},
			want:    UserProfile{ID: userID, Email: "newemail@example.com", Name: "John Doe", CreatedAt: now.UTC().Format(time.RFC3339), UpdatedAt: now.UTC().Format(time.RFC3339)},
			wantErr: false,
		},
		{
			name:    "empty params error (no GetByID called)",
			params:  UpdateProfileParams{UserID: userID},
			mockFn:  func(_ context.Context, _ uuid.UUID, _, _ string) (models.User, error) { return models.User{}, nil },
			wantErr: true,
			errType: apperror.ErrInvalidInput,
		},
		{
			name: "duplicate email (conflict)",
			params: UpdateProfileParams{
				UserID: userID,
				Email:  "taken@example.com",
			},
			getByIDFn: defaultGetByID,
			mockFn: func(_ context.Context, _ uuid.UUID, _, _ string) (models.User, error) {
				return models.User{}, apperror.ErrConflict
			},
			wantErr: true,
			errType: apperror.ErrConflict,
		},
		{
			name: "email normalized to lowercase",
			params: UpdateProfileParams{
				UserID: userID,
				Email:  "NEW@EXAMPLE.COM",
			},
			getByIDFn: defaultGetByID,
			mockFn: func(_ context.Context, _ uuid.UUID, _, email string) (models.User, error) {
				assert.Equal(t, "new@example.com", email, "email must be lowercased before repo call")
				return models.User{ID: userID, Email: email, Name: currentUser.Name, CreatedAt: now, UpdatedAt: now}, nil
			},
			want:    UserProfile{ID: userID, Email: "new@example.com", Name: "John Doe", CreatedAt: now.UTC().Format(time.RFC3339), UpdatedAt: now.UTC().Format(time.RFC3339)},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getByID := tt.getByIDFn
			if getByID == nil {
				getByID = func(_ context.Context, _ uuid.UUID) (models.User, error) {
					panic("GetByID should not be called for this test case")
				}
			}
			repo := &mockUserRepository{getByIDFn: getByID, updateFn: tt.mockFn}
			svc := NewUserService(repo, 10)

			got, err := svc.UpdateProfile(context.Background(), tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errType, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want.ID, got.ID)
			assert.Equal(t, tt.want.Email, got.Email)
			assert.Equal(t, tt.want.Name, got.Name)
		})
	}
}

func TestUserService_ChangePassword(t *testing.T) {
	userID := uuid.New()
	now := time.Now().UTC()

	oldPassword := "oldpass123"
	oldHash, _ := bcrypt.GenerateFromPassword([]byte(oldPassword), 10)

	tests := []struct {
		name    string
		params  ChangePasswordParams
		wantErr bool
		errType error
	}{
		{
			name: "success",
			params: ChangePasswordParams{
				UserID:          userID,
				CurrentPassword: "oldpass123",
				NewPassword:     "newpass456",
			},
			wantErr: false,
		},
		{
			name: "wrong current password",
			params: ChangePasswordParams{
				UserID:          userID,
				CurrentPassword: "wrongpass",
				NewPassword:     "newpass456",
			},
			wantErr: true,
			errType: apperror.ErrUnauthorized,
		},
		{
			name: "new password too short",
			params: ChangePasswordParams{
				UserID:          userID,
				CurrentPassword: "oldpass123",
				NewPassword:     "short",
			},
			wantErr: true,
			errType: apperror.ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepository{
				getByIDFn: func(_ context.Context, _ uuid.UUID) (models.User, error) {
					return models.User{
						ID:           userID,
						PasswordHash: string(oldHash),
						CreatedAt:    now,
						UpdatedAt:    now,
					}, nil
				},
			}
			svc := NewUserService(repo, 10)
			err := svc.ChangePassword(context.Background(), tt.params)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Equal(t, tt.errType, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestUserService_Delete(t *testing.T) {
	userID := uuid.New()

	repo := &mockUserRepository{}
	svc := NewUserService(repo, 10)

	err := svc.Delete(context.Background(), userID)
	assert.NoError(t, err)
}
