package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	allowedOrigins := strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",")
	originsMap := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		originsMap[strings.TrimSpace(o)] = true
	}

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if originsMap[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		}
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
