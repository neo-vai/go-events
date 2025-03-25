package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RateLimiter(2, time.Minute))
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	// First two requests should pass
	for i := 0; i < 2; i++ {
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
		w = httptest.NewRecorder()
	}
	// Third request should be rate limited
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		id := c.GetString(string(RequestIDKey))
		c.Header("X-Request-ID", id)
		c.String(200, "ok")
	})

	// Without header – should generate
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))

	// With header – should propagate
	w = httptest.NewRecorder()
	req.Header.Set("X-Request-ID", "custom-id")
	r.ServeHTTP(w, req)
	assert.Equal(t, "custom-id", w.Header().Get("X-Request-ID"))
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS())
	r.OPTIONS("/test", func(c *gin.Context) { c.Status(204) })
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Access-Control-Request-Method", "GET")
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestJWTAuth_ValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(JWTAuth())
	r.GET("/protected", func(c *gin.Context) {
		c.String(200, "ok")
	})

	// Generate a valid token
	token, _ := GenerateJWT("test-account")
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestJWTAuth_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(JWTAuth())
	r.GET("/protected", func(c *gin.Context) { c.String(200, "ok") })

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/protected", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
