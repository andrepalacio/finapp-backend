package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/andrespalacio/finapp-backend/internal/middleware"
	"github.com/andrespalacio/finapp-backend/internal/models"
	"github.com/andrespalacio/finapp-backend/internal/repositories"
	"github.com/andrespalacio/finapp-backend/internal/services"
	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockUserRepoForHandlerTest struct {
	getByIDFn func(ctx context.Context, id uuid.UUID) (models.User, error)
	updateFn  func(ctx context.Context, userID uuid.UUID, name, email string) (models.User, error)
}

func (m *mockUserRepoForHandlerTest) Create(ctx context.Context, params repositories.CreateUserParams) (models.User, error) {
	return models.User{}, nil
}

func (m *mockUserRepoForHandlerTest) GetByEmail(ctx context.Context, email string) (models.User, error) {
	return models.User{}, nil
}

func (m *mockUserRepoForHandlerTest) GetByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockUserRepoForHandlerTest) Update(ctx context.Context, userID uuid.UUID, name, email string) (models.User, error) {
	return m.updateFn(ctx, userID, name, email)
}

func TestUserHandler_GetProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	userID := uuid.New()
	jwtManager := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, 7*24*time.Hour)
	token, _ := jwtManager.GenerateTokenPair(userID)

	tests := []struct {
		name           string
		userID         uuid.UUID
		mockFn         func(ctx context.Context, id uuid.UUID) (models.User, error)
		expectedStatus int
	}{
		{
			name:   "success",
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
			expectedStatus: http.StatusOK,
		},
		{
			name:   "user not found",
			userID: userID,
			mockFn: func(ctx context.Context, id uuid.UUID) (models.User, error) {
				return models.User{}, apperror.ErrNotFound
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockUserRepoForHandlerTest{getByIDFn: tt.mockFn}
			svc := services.NewUserService(repo)
			handler := NewUserHandler(svc)

			router := gin.New()
			router.GET("/user/profile", middleware.AuthMiddleware(jwtManager), handler.GetProfile)

			req := httptest.NewRequest("GET", "/user/profile", nil)
			req.Header.Set("Authorization", "Bearer "+token.AccessToken)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedStatus == http.StatusOK {
				var resp services.UserProfile
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)
				assert.Equal(t, tt.userID, resp.ID)
				assert.Equal(t, "john@example.com", resp.Email)
			}
		})
	}
}

func TestUserHandler_UpdateProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now().UTC()
	userID := uuid.New()
	jwtManager := pkgauth.NewJWTManager("test-secret-that-is-long-enough-32chars!!", 15*time.Minute, 7*24*time.Hour)
	token, _ := jwtManager.GenerateTokenPair(userID)

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
		name           string
		userID         uuid.UUID
		requestBody    UpdateProfileRequest
		getByIDFn      func(ctx context.Context, id uuid.UUID) (models.User, error)
		mockFn         func(ctx context.Context, userID uuid.UUID, name, email string) (models.User, error)
		expectedStatus int
	}{
		{
			name:   "success",
			userID: userID,
			requestBody: UpdateProfileRequest{
				Name:  "Jane Doe",
				Email: "jane@example.com",
			},
			getByIDFn: defaultGetByID,
			mockFn: func(_ context.Context, _ uuid.UUID, name, email string) (models.User, error) {
				return models.User{
					ID:        userID,
					Email:     email,
					Name:      name,
					CreatedAt: now,
					UpdatedAt: now,
				}, nil
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid body",
			userID:         userID,
			mockFn:         func(_ context.Context, _ uuid.UUID, _, _ string) (models.User, error) { return models.User{}, nil },
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "duplicate email",
			userID: userID,
			requestBody: UpdateProfileRequest{
				Email: "taken@example.com",
			},
			getByIDFn: defaultGetByID,
			mockFn: func(_ context.Context, _ uuid.UUID, _, _ string) (models.User, error) {
				return models.User{}, apperror.ErrConflict
			},
			expectedStatus: http.StatusConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getByID := tt.getByIDFn
			if getByID == nil {
				getByID = func(_ context.Context, _ uuid.UUID) (models.User, error) {
					return models.User{}, apperror.ErrInvalidInput
				}
			}
			repo := &mockUserRepoForHandlerTest{getByIDFn: getByID, updateFn: tt.mockFn}
			svc := services.NewUserService(repo)
			handler := NewUserHandler(svc)

			router := gin.New()
			router.PUT("/user/profile", middleware.AuthMiddleware(jwtManager), handler.UpdateProfile)

			bodyBytes, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("PUT", "/user/profile", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token.AccessToken)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
