package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ── Mock repository ───────────────────────────────────────────────────────────

type mockUserRepo struct {
	users     map[string]models.User
	createErr error
}

func newMockRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]models.User)}
}

func (m *mockUserRepo) Create(_ context.Context, p repositories.CreateUserParams) (models.User, error) {
	if m.createErr != nil {
		return models.User{}, m.createErr
	}
	if _, exists := m.users[p.Email]; exists {
		return models.User{}, apperror.ErrConflict
	}
	u := models.User{ID: uuid.New(), Email: p.Email, PasswordHash: p.PasswordHash, Name: p.Name}
	m.users[p.Email] = u
	return u, nil
}

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (models.User, error) {
	u, ok := m.users[email]
	if !ok {
		return models.User{}, apperror.ErrNotFound
	}
	return u, nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (models.User, error) {
	for _, u := range m.users {
		if u.ID == id {
			return u, nil
		}
	}
	return models.User{}, apperror.ErrNotFound
}

func (m *mockUserRepo) Update(_ context.Context, userID uuid.UUID, name, email string) (models.User, error) {
	u, ok := m.users[email]
	if ok && u.ID != userID {
		return models.User{}, apperror.ErrConflict
	}
	for i, user := range m.users {
		if user.ID == userID {
			user.Name = name
			user.Email = email
			m.users[i] = user
			delete(m.users, i)
			m.users[email] = user
			return user, nil
		}
	}
	return models.User{}, apperror.ErrNotFound
}

// ── Test helpers ──────────────────────────────────────────────────────────────

func newTestService(t *testing.T) (*services.AuthService, *mockUserRepo) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwt := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, 7*24*time.Hour)
	repo := newMockRepo()
	return services.NewAuthService(repo, rdb, jwt, bcrypt.DefaultCost), repo
}

// ── Register ──────────────────────────────────────────────────────────────────

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name    string
		params  services.RegisterParams
		wantErr error
	}{
		{
			name:   "valid registration",
			params: services.RegisterParams{Email: "user@example.com", Password: "password123", Name: "Test User"},
		},
		{
			name:    "password too short",
			params:  services.RegisterParams{Email: "user@example.com", Password: "short", Name: "Test User"},
			wantErr: apperror.ErrInvalidInput,
		},
		{
			name:    "password too long (bcrypt 72-byte truncation guard)",
			params:  services.RegisterParams{Email: "user@example.com", Password: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Name: "Test User"},
			wantErr: apperror.ErrInvalidInput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService(t)
			pair, err := svc.Register(context.Background(), tt.params)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, pair.AccessToken)
			assert.NotEmpty(t, pair.RefreshToken)
		})
	}
}

func TestAuthService_Register_EmailNormalized(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Register(context.Background(), services.RegisterParams{
		Email: "USER@Example.com", Password: "password123", Name: "User",
	})
	require.NoError(t, err)

	_, err = svc.Login(context.Background(), services.LoginParams{
		Email: "user@example.com", Password: "password123",
	})
	assert.NoError(t, err, "login with lowercase of originally-uppercase email must succeed")
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	svc, _ := newTestService(t)
	p := services.RegisterParams{Email: "dup@example.com", Password: "password123", Name: "User"}

	_, err := svc.Register(context.Background(), p)
	require.NoError(t, err)

	_, err = svc.Register(context.Background(), p)
	assert.ErrorIs(t, err, apperror.ErrConflict)
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestAuthService_Login(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Register(context.Background(), services.RegisterParams{
		Email: "login@example.com", Password: "password123", Name: "User",
	})
	require.NoError(t, err)

	tests := []struct {
		name    string
		params  services.LoginParams
		wantErr error
	}{
		{
			name:   "correct credentials",
			params: services.LoginParams{Email: "login@example.com", Password: "password123"},
		},
		{
			name:    "wrong password",
			params:  services.LoginParams{Email: "login@example.com", Password: "wrongpass"},
			wantErr: apperror.ErrUnauthorized,
		},
		{
			name:    "email not found returns unauthorized (no user enumeration)",
			params:  services.LoginParams{Email: "noexist@example.com", Password: "password123"},
			wantErr: apperror.ErrUnauthorized,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pair, err := svc.Login(context.Background(), tt.params)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.NotEmpty(t, pair.AccessToken)
			assert.NotEmpty(t, pair.RefreshToken)
		})
	}
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func TestAuthService_Refresh(t *testing.T) {
	svc, _ := newTestService(t)

	pair, err := svc.Register(context.Background(), services.RegisterParams{
		Email: "refresh@example.com", Password: "password123", Name: "User",
	})
	require.NoError(t, err)

	t.Run("valid refresh token returns new pair", func(t *testing.T) {
		newPair, err := svc.Refresh(context.Background(), pair.RefreshToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newPair.AccessToken)
		assert.NotEmpty(t, newPair.RefreshToken)
		assert.NotEqual(t, pair.RefreshToken, newPair.RefreshToken)
	})

	t.Run("reused token is rejected (token rotation)", func(t *testing.T) {
		_, err := svc.Refresh(context.Background(), pair.RefreshToken)
		assert.ErrorIs(t, err, apperror.ErrUnauthorized)
	})
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	svc, _ := newTestService(t)
	_, err := svc.Refresh(context.Background(), "not-a-valid-jwt")
	assert.ErrorIs(t, err, apperror.ErrUnauthorized)
}

// ── Logout ────────────────────────────────────────────────────────────────────

func TestAuthService_Logout(t *testing.T) {
	svc, _ := newTestService(t)

	pair, err := svc.Register(context.Background(), services.RegisterParams{
		Email: "logout@example.com", Password: "password123", Name: "User",
	})
	require.NoError(t, err)

	t.Run("logout invalidates refresh token", func(t *testing.T) {
		require.NoError(t, svc.Logout(context.Background(), pair.RefreshToken))
		_, err := svc.Refresh(context.Background(), pair.RefreshToken)
		assert.ErrorIs(t, err, apperror.ErrUnauthorized)
	})

	t.Run("logout of unknown token is idempotent", func(t *testing.T) {
		assert.NoError(t, svc.Logout(context.Background(), "nonexistent-token"))
	})
}

// ── bcrypt cost sanity ────────────────────────────────────────────────────────

func TestPasswordHash_NotStoredInPlaintext(t *testing.T) {
	svc, repo := newTestService(t)
	p := "mypassword123"

	_, err := svc.Register(context.Background(), services.RegisterParams{
		Email: "hash@example.com", Password: p, Name: "User",
	})
	require.NoError(t, err)

	u, err := repo.GetByEmail(context.Background(), "hash@example.com")
	require.NoError(t, err)
	assert.NotEqual(t, p, u.PasswordHash)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(p)))
}
