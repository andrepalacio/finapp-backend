package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthMiddleware(t *testing.T) {
	jwtManager := pkgauth.NewJWTManager("this-is-a-32-character-secret!!!", time.Minute, time.Hour)
	userID := uuid.New()
	pair, err := jwtManager.GenerateTokenPair(userID)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		authHeader     string
		wantStatus     int
		wantNextCalled bool
	}{
		{
			name:           "missing header",
			authHeader:     "",
			wantStatus:     http.StatusUnauthorized,
			wantNextCalled: false,
		},
		{
			name:           "malformed header no bearer prefix",
			authHeader:     pair.AccessToken,
			wantStatus:     http.StatusUnauthorized,
			wantNextCalled: false,
		},
		{
			name:           "invalid token",
			authHeader:     "Bearer not-a-real-token",
			wantStatus:     http.StatusUnauthorized,
			wantNextCalled: false,
		},
		{
			name:           "valid token",
			authHeader:     "Bearer " + pair.AccessToken,
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			var gotUserID uuid.UUID

			r := gin.New()
			r.GET("/protected", AuthMiddleware(jwtManager), func(c *gin.Context) {
				nextCalled = true
				gotUserID = UserIDFromContext(c)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantNextCalled, nextCalled)
			if tt.wantNextCalled {
				assert.Equal(t, userID, gotUserID)
			}
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	t.Run("allowed origin gets header set", func(t *testing.T) {
		os.Setenv("ALLOWED_ORIGINS", "https://app.example.com,https://other.example.com")
		r := gin.New()
		r.Use(CORSMiddleware())
		nextCalled := false
		r.GET("/x", func(c *gin.Context) {
			nextCalled = true
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://app.example.com")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, "https://app.example.com", w.Header().Get("Access-Control-Allow-Origin"))
		assert.True(t, nextCalled)
	})

	t.Run("disallowed origin gets no header", func(t *testing.T) {
		os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
		r := gin.New()
		r.Use(CORSMiddleware())
		r.GET("/x", func(c *gin.Context) { c.Status(http.StatusOK) })

		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://evil.example.com")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("options request aborts with 204", func(t *testing.T) {
		os.Setenv("ALLOWED_ORIGINS", "https://app.example.com")
		r := gin.New()
		r.Use(CORSMiddleware())
		nextCalled := false
		r.OPTIONS("/x", func(c *gin.Context) {
			nextCalled = true
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodOptions, "/x", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.False(t, nextCalled)
	})
}

func TestLoggerMiddleware(t *testing.T) {
	logger := zap.NewNop()
	nextCalled := false

	r := gin.New()
	r.Use(LoggerMiddleware(logger))
	r.GET("/x", func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	w := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		r.ServeHTTP(w, req)
	})
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, w.Code)
}

type mockMemberChecker struct {
	isMemberFn func(ctx context.Context, workspaceID, userID uuid.UUID) bool
}

func (m *mockMemberChecker) IsMember(ctx context.Context, workspaceID, userID uuid.UUID) bool {
	return m.isMemberFn(ctx, workspaceID, userID)
}

func TestWorkspaceMiddleware(t *testing.T) {
	validWSID := uuid.New()
	validUserID := uuid.New()

	tests := []struct {
		name           string
		workspaceParam string
		setUser        bool
		isMember       bool
		wantStatus     int
		wantNextCalled bool
	}{
		{
			name:           "invalid workspace_id",
			workspaceParam: "not-a-uuid",
			setUser:        true,
			isMember:       true,
			wantStatus:     http.StatusBadRequest,
			wantNextCalled: false,
		},
		{
			name:           "no authenticated user",
			workspaceParam: validWSID.String(),
			setUser:        false,
			isMember:       true,
			wantStatus:     http.StatusUnauthorized,
			wantNextCalled: false,
		},
		{
			name:           "not a member",
			workspaceParam: validWSID.String(),
			setUser:        true,
			isMember:       false,
			wantStatus:     http.StatusForbidden,
			wantNextCalled: false,
		},
		{
			name:           "is a member",
			workspaceParam: validWSID.String(),
			setUser:        true,
			isMember:       true,
			wantStatus:     http.StatusOK,
			wantNextCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nextCalled := false
			var gotWSID uuid.UUID

			checker := &mockMemberChecker{
				isMemberFn: func(_ context.Context, _, _ uuid.UUID) bool { return tt.isMember },
			}

			r := gin.New()
			if tt.setUser {
				r.Use(func(c *gin.Context) {
					c.Set(userIDKey, validUserID)
					c.Next()
				})
			}
			r.GET("/:workspace_id", WorkspaceMiddleware(checker), func(c *gin.Context) {
				nextCalled = true
				gotWSID = WorkspaceIDFromContext(c)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/"+tt.workspaceParam, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
			assert.Equal(t, tt.wantNextCalled, nextCalled)
			if tt.wantNextCalled {
				assert.Equal(t, validWSID, gotWSID)
			}
		})
	}
}
