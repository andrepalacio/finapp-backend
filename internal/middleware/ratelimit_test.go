package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimitMiddleware_FailsOpenWhenRedisUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1", // nothing listens here: Incr will error immediately
		DialTimeout: 200 * time.Millisecond,
	})
	defer rdb.Close()

	r := gin.New()
	nextCalled := false
	r.GET("/ping", RateLimitMiddleware(rdb, 10, time.Minute), func(c *gin.Context) {
		nextCalled = true
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.True(t, nextCalled, "request should proceed to next handler when redis is unavailable (fail open)")
	assert.Equal(t, http.StatusOK, w.Code)
}

func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	return redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestRateLimitMiddleware_AllowsUnderLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := newTestRedis(t)
	defer rdb.Close()

	r := gin.New()
	calls := 0
	r.GET("/ping", RateLimitMiddleware(rdb, 3, time.Minute), func(c *gin.Context) {
		calls++
		c.Status(http.StatusOK)
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
	assert.Equal(t, 3, calls)
}

func TestRateLimitMiddleware_BlocksOverLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := newTestRedis(t)
	defer rdb.Close()

	r := gin.New()
	calls := 0
	r.GET("/ping", RateLimitMiddleware(rdb, 2, time.Minute), func(c *gin.Context) {
		calls++
		c.Status(http.StatusOK)
	})

	var lastCode int
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		lastCode = w.Code
	}

	assert.Equal(t, http.StatusTooManyRequests, lastCode)
	assert.Equal(t, 2, calls, "next handler should not run once the limit is exceeded")
}

func TestRateLimitMiddleware_KeyedPerClientAndRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rdb := newTestRedis(t)
	defer rdb.Close()

	r := gin.New()
	r.GET("/a", RateLimitMiddleware(rdb, 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/b", RateLimitMiddleware(rdb, 1, time.Minute), func(c *gin.Context) { c.Status(http.StatusOK) })

	// First hit on /a consumes its own limit; /b is a different route key and should still be allowed.
	req := httptest.NewRequest(http.MethodGet, "/a", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/b", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
