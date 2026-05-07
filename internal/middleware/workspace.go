package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MemberChecker interface {
	IsMember(ctx context.Context, workspaceID, userID uuid.UUID) bool
}

const workspaceIDKey = "workspaceID"

func WorkspaceIDFromContext(c *gin.Context) uuid.UUID {
	id, _ := c.Get(workspaceIDKey)
	uid, _ := id.(uuid.UUID)
	return uid
}

// WorkspaceMiddleware verifies the authenticated user belongs to the workspace
// in :workspace_id and injects the workspace UUID into context.
func WorkspaceMiddleware(checker MemberChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		wsIDStr := c.Param("workspace_id")
		wsID, err := uuid.Parse(wsIDStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "invalid workspace_id",
				"code":  "INVALID_INPUT",
			})
			return
		}

		userID := UserIDFromContext(c)
		if userID == uuid.Nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "unauthorized",
				"code":  "UNAUTHORIZED",
			})
			return
		}

		if !checker.IsMember(c.Request.Context(), wsID, userID) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "not a member of this workspace",
				"code":  "FORBIDDEN",
			})
			return
		}

		c.Set(workspaceIDKey, wsID)
		c.Next()
	}
}
