package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggerMiddleware_LevelsAndRequestID(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		header    string
		wantLevel zapcore.Level
	}{
		{"2xx logs info", http.StatusOK, "", zapcore.InfoLevel},
		{"4xx logs warn", http.StatusNotFound, "", zapcore.WarnLevel},
		{"5xx logs error", http.StatusInternalServerError, "", zapcore.ErrorLevel},
		{"reuses incoming request id", http.StatusOK, "req-123", zapcore.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(zapcore.DebugLevel)
			r := gin.New()
			r.Use(LoggerMiddleware(zap.New(core)))
			r.GET("/x", func(c *gin.Context) { c.Status(tt.status) })

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tt.header != "" {
				req.Header.Set("X-Request-ID", tt.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			entries := logs.All()
			assert.Len(t, entries, 1)
			assert.Equal(t, tt.wantLevel, entries[0].Level)

			id := entries[0].ContextMap()["request_id"]
			assert.NotEmpty(t, id)
			assert.Equal(t, id, w.Header().Get("X-Request-ID"))
			if tt.header != "" {
				assert.Equal(t, tt.header, id)
			}
		})
	}
}
