package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/andrespalacio/finapp-backend/internal/handlers"
	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ── In-memory UserRepository ──────────────────────────────────────────────────

type inMemoryUserRepo struct {
	users map[string]models.User
}

func newInMemoryRepo() *inMemoryUserRepo {
	return &inMemoryUserRepo{users: make(map[string]models.User)}
}

func (r *inMemoryUserRepo) Create(_ context.Context, p repositories.CreateUserParams) (models.User, error) {
	if _, exists := r.users[p.Email]; exists {
		return models.User{}, apperror.ErrConflict
	}
	u := models.User{ID: uuid.New(), Email: p.Email, PasswordHash: p.PasswordHash, Name: p.Name}
	r.users[p.Email] = u
	return u, nil
}

func (r *inMemoryUserRepo) GetByEmail(_ context.Context, email string) (models.User, error) {
	u, ok := r.users[email]
	if !ok {
		return models.User{}, apperror.ErrNotFound
	}
	return u, nil
}

func (r *inMemoryUserRepo) GetByID(_ context.Context, id uuid.UUID) (models.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return models.User{}, apperror.ErrNotFound
}

func (r *inMemoryUserRepo) Update(_ context.Context, userID uuid.UUID, name, email string) (models.User, error) {
	u, ok := r.users[email]
	if ok && u.ID != userID {
		return models.User{}, apperror.ErrConflict
	}
	for key, user := range r.users {
		if user.ID == userID {
			user.Name = name
			user.Email = email
			delete(r.users, key)
			r.users[email] = user
			return user, nil
		}
	}
	return models.User{}, apperror.ErrNotFound
}

// ── Test helpers ──────────────────────────────────────────────────────────────

func setupRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	jwt := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, 7*24*time.Hour)
	svc := services.NewAuthService(newInMemoryRepo(), rdb, jwt, 10)
	h := handlers.NewAuthHandler(svc)

	r := gin.New()
	r.POST("/auth/register", h.Register)
	r.POST("/auth/login", h.Login)
	r.POST("/auth/refresh", h.Refresh)
	r.POST("/auth/logout", h.Logout)

	return r
}

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// ── Register ──────────────────────────────────────────────────────────────────

func TestAuthHandler_Register(t *testing.T) {
	r := setupRouter(t)

	t.Run("valid body returns 201 with tokens", func(t *testing.T) {
		w := postJSON(r, "/auth/register", map[string]string{
			"email": "new@example.com", "password": "password123", "name": "New User",
		})
		assert.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp["access_token"])
		assert.NotEmpty(t, resp["refresh_token"])
	})

	t.Run("missing email returns 400", func(t *testing.T) {
		w := postJSON(r, "/auth/register", map[string]string{
			"password": "password123", "name": "User",
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid email format returns 400", func(t *testing.T) {
		w := postJSON(r, "/auth/register", map[string]string{
			"email": "not-an-email", "password": "password123", "name": "User",
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("password too short returns 400", func(t *testing.T) {
		w := postJSON(r, "/auth/register", map[string]string{
			"email": "short@example.com", "password": "short", "name": "User",
		})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("duplicate email returns 409", func(t *testing.T) {
		body := map[string]string{
			"email": "dup@example.com", "password": "password123", "name": "User",
		}
		postJSON(r, "/auth/register", body)
		w := postJSON(r, "/auth/register", body)
		assert.Equal(t, http.StatusConflict, w.Code)
	})
}

// ── Login ─────────────────────────────────────────────────────────────────────

func TestAuthHandler_Login(t *testing.T) {
	r := setupRouter(t)

	postJSON(r, "/auth/register", map[string]string{
		"email": "login@example.com", "password": "password123", "name": "User",
	})

	t.Run("correct credentials returns 200 with tokens", func(t *testing.T) {
		w := postJSON(r, "/auth/login", map[string]string{
			"email": "login@example.com", "password": "password123",
		})
		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]string
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.NotEmpty(t, resp["access_token"])
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		w := postJSON(r, "/auth/login", map[string]string{
			"email": "login@example.com", "password": "wrongpass",
		})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("unknown email returns 401", func(t *testing.T) {
		w := postJSON(r, "/auth/login", map[string]string{
			"email": "nobody@example.com", "password": "password123",
		})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ── Refresh ───────────────────────────────────────────────────────────────────

func TestAuthHandler_Refresh(t *testing.T) {
	r := setupRouter(t)

	regResp := postJSON(r, "/auth/register", map[string]string{
		"email": "refresh@example.com", "password": "password123", "name": "User",
	})
	require.Equal(t, http.StatusCreated, regResp.Code)
	var tokens map[string]string
	require.NoError(t, json.Unmarshal(regResp.Body.Bytes(), &tokens))

	t.Run("valid refresh token returns 200 with new tokens", func(t *testing.T) {
		w := postJSON(r, "/auth/refresh", map[string]string{
			"refresh_token": tokens["refresh_token"],
		})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("missing refresh_token returns 400", func(t *testing.T) {
		w := postJSON(r, "/auth/refresh", map[string]string{})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		w := postJSON(r, "/auth/refresh", map[string]string{
			"refresh_token": "not-a-valid-token",
		})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ── Logout ────────────────────────────────────────────────────────────────────

func TestAuthHandler_Logout(t *testing.T) {
	r := setupRouter(t)

	regResp := postJSON(r, "/auth/register", map[string]string{
		"email": "logout@example.com", "password": "password123", "name": "User",
	})
	require.Equal(t, http.StatusCreated, regResp.Code)
	var tokens map[string]string
	require.NoError(t, json.Unmarshal(regResp.Body.Bytes(), &tokens))

	t.Run("valid logout returns 204", func(t *testing.T) {
		w := postJSON(r, "/auth/logout", map[string]string{
			"refresh_token": tokens["refresh_token"],
		})
		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("after logout refresh token is rejected", func(t *testing.T) {
		w := postJSON(r, "/auth/refresh", map[string]string{
			"refresh_token": tokens["refresh_token"],
		})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}
