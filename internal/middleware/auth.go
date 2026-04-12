package middleware

import (
	"strings"

	pkgauth "github.com/andrespalacio/finapp-backend/pkg/auth"
	"github.com/andrespalacio/finapp-backend/pkg/apperror"
	"github.com/andrespalacio/finapp-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const userIDKey = "userID"

func AuthMiddleware(jwt *pkgauth.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			response.Error(c, apperror.ErrUnauthorized.StatusCode, apperror.ErrUnauthorized.Code, apperror.ErrUnauthorized.Message)
			c.Abort()
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwt.ValidateAccessToken(token)
		if err != nil {
			response.Error(c, apperror.ErrUnauthorized.StatusCode, apperror.ErrUnauthorized.Code, apperror.ErrUnauthorized.Message)
			c.Abort()
			return
		}

		c.Set(userIDKey, claims.UserID)
		c.Next()
	}
}

func UserIDFromContext(c *gin.Context) uuid.UUID {
	val, _ := c.Get(userIDKey)
	id, _ := val.(uuid.UUID)
	return id
}
