package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---------- Helper to set JWT_SECRET for tests ----------
func setJWTSecret(t *testing.T, secret string) {
	t.Helper()
	old := os.Getenv("JWT_SECRET")
	os.Setenv("JWT_SECRET", secret)
	t.Cleanup(func() { os.Setenv("JWT_SECRET", old) })
}

// ---------- RequestID ----------
func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(200, c.GetString(string(RequestIDKey)))
	})

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	assert.NotEmpty(t, w.Body.String())
	assert.Equal(t, w.Header().Get("X-Request-ID"), w.Body.String())
}

func TestRequestID_PropagatesExisting(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		c.String(200, c.GetString(string(RequestIDKey)))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "custom-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "custom-123", w.Header().Get("X-Request-ID"))
	assert.Equal(t, "custom-123", w.Body.String())
}

// ---------- CORS ----------
func TestCORS_SetsHeadersAndHandlesPreflight(t *testing.T) {
	r := gin.New()
	r.Use(CORS())
	r.GET("/test", func(c *gin.Context) { c.String(200, "ok") })

	// Preflight
	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")

	// Actual request
	req = httptest.NewRequest("GET", "/test", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

// ---------- StructuredLogger ----------
func TestStructuredLogger_LogsJSON(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.Use(StructuredLogger())
	r.GET("/test", func(c *gin.Context) {
		c.Status(200)
	})

	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	req.Header.Set("X-Request-ID", "log-test")
	w := httptest.NewRecorder()

	// Capture log output
	oldWriter := gin.DefaultWriter
	buf := &strings.Builder{}
	gin.DefaultWriter = buf
	defer func() { gin.DefaultWriter = oldWriter }()

	r.ServeHTTP(w, req)

	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "info", logEntry["level"])
	assert.Equal(t, float64(200), logEntry["status"])
	assert.Equal(t, "GET", logEntry["method"])
	assert.Equal(t, "/test?foo=bar", logEntry["path"])
	assert.Equal(t, "log-test", logEntry["request_id"])
	assert.NotNil(t, logEntry["latency_ms"])
}

// ---------- RateLimiter ----------
func TestRateLimiter_AllowsUpToLimit(t *testing.T) {
	r := gin.New()
	r.Use(RateLimiter(2, time.Minute))
	r.GET("/", func(c *gin.Context) { c.String(200, "ok") })

	ip := "192.168.1.1"
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = ip + ":12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = ip + ":12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimiter_DifferentIPsSeparate(t *testing.T) {
	mu.Lock()
	limiters = make(map[string]*rate.Limiter)
	mu.Unlock()

	r := gin.New()
	r.Use(RateLimiter(1, time.Minute))
	r.GET("/", func(c *gin.Context) { c.String(200, "ok") })

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	req1 := httptest.NewRequest("GET", "/", nil)
	req1.RemoteAddr = ip1 + ":12345"
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	assert.Equal(t, http.StatusOK, w1.Code)

	req2 := httptest.NewRequest("GET", "/", nil)
	req2.RemoteAddr = ip2 + ":12345"
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)
}

// ---------- JWT Generation and Validation ----------
func TestGenerateJWT_And_ValidateJWT(t *testing.T) {
	setJWTSecret(t, "test-secret")
	os.Setenv("JWT_EXPIRES_HOURS", "1")

	accountID := "acc-123"
	role := "user"
	token, err := GenerateJWT(accountID, role)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ValidateJWT(token)
	require.NoError(t, err)
	assert.Equal(t, accountID, claims.AccountID)
	assert.Equal(t, role, claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
}

func TestValidateJWT_InvalidSignature(t *testing.T) {
	setJWTSecret(t, "good-secret")
	token, _ := GenerateJWT("acc", "user")
	setJWTSecret(t, "wrong-secret")
	_, err := ValidateJWT(token)
	assert.Error(t, err)
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	setJWTSecret(t, "secret")
	os.Setenv("JWT_EXPIRES_HOURS", "-1") // negative to expire immediately
	token, _ := GenerateJWT("acc", "user")
	time.Sleep(1 * time.Second)
	_, err := ValidateJWT(token)
	assert.Error(t, err)
	os.Setenv("JWT_EXPIRES_HOURS", "24") // restore
}

// ---------- JWTAuth Middleware ----------
func TestJWTAuth_ValidToken(t *testing.T) {
	setJWTSecret(t, "secret")
	r := gin.New()
	r.Use(JWTAuth())
	r.GET("/protected", func(c *gin.Context) {
		c.String(200, c.GetString(string(AccountIDKey)))
	})

	token, _ := GenerateJWT("test-account", "user")
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test-account", w.Body.String())
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	r := gin.New()
	r.Use(JWTAuth())
	r.GET("/protected", func(c *gin.Context) {})
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	r := gin.New()
	r.Use(JWTAuth())
	r.GET("/protected", func(c *gin.Context) {})
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	setJWTSecret(t, "secret")
	os.Setenv("JWT_EXPIRES_HOURS", "-1")
	token, _ := GenerateJWT("acc", "user")
	os.Setenv("JWT_EXPIRES_HOURS", "24")
	r := gin.New()
	r.Use(JWTAuth())
	r.GET("/protected", func(c *gin.Context) {})
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ---------- UniversalAuth ----------
func TestUniversalAuth_JWT_Success(t *testing.T) {
	setJWTSecret(t, "secret")
	validator := func(c *gin.Context, key string) (string, string, error) {
		return "", "", nil // not called
	}
	r := gin.New()
	r.Use(UniversalAuth(validator))
	r.GET("/", func(c *gin.Context) {
		c.String(200, c.GetString(string(AccountIDKey)))
	})

	token, _ := GenerateJWT("acc-jwt", "user")
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "acc-jwt", w.Body.String())
}

func TestUniversalAuth_APIKey_Success(t *testing.T) {
	validator := func(c *gin.Context, key string) (string, string, error) {
		if key == "valid-key" {
			return "acc-apikey", "user", nil
		}
		return "", "", nil
	}
	r := gin.New()
	r.Use(UniversalAuth(validator))
	r.GET("/", func(c *gin.Context) {
		c.String(200, c.GetString(string(AccountIDKey)))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "valid-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "acc-apikey", w.Body.String())
}

func TestUniversalAuth_NoCredentials(t *testing.T) {
	validator := func(c *gin.Context, key string) (string, string, error) {
		return "", "", nil
	}
	r := gin.New()
	r.Use(UniversalAuth(validator))
	r.GET("/", func(c *gin.Context) {})
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUniversalAuth_BothInvalid(t *testing.T) {
	validator := func(c *gin.Context, key string) (string, string, error) {
		if key == "bad" {
			return "", "", nil
		}
		return "", "", nil
	}
	r := gin.New()
	r.Use(UniversalAuth(validator))
	r.GET("/", func(c *gin.Context) {})
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer bad")
	req.Header.Set("X-API-Key", "bad")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUniversalAuth_InvalidJWT_RejectedEvenWithValidAPIKey(t *testing.T) {
	setJWTSecret(t, "secret")
	validator := func(c *gin.Context, key string) (string, string, error) {
		if key == "good-key" {
			return "acc-apikey", "user", nil
		}
		return "", "", nil
	}
	r := gin.New()
	r.Use(UniversalAuth(validator))
	r.GET("/", func(c *gin.Context) {
		c.String(200, c.GetString(string(AccountIDKey)))
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	req.Header.Set("X-API-Key", "good-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var errResp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.NoError(t, err)
	assert.Equal(t, "invalid or expired token", errResp["error"])
}

// ---------- OwnerCheck ----------
func TestOwnerCheck_AllowsOwner(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(AccountIDKey), "user-123")
		c.Next()
	})
	r.Use(OwnerCheck())
	r.GET("/resource/:id", func(c *gin.Context) {
		c.String(200, "ok")
	})

	req := httptest.NewRequest("GET", "/resource/user-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestOwnerCheck_DeniesNonOwner(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(AccountIDKey), "user-123")
		c.Next()
	})
	r.Use(OwnerCheck())
	r.GET("/resource/:id", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/resource/user-456", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestOwnerCheck_MissingAccountID(t *testing.T) {
	r := gin.New()
	r.Use(OwnerCheck())
	r.GET("/resource/:id", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/resource/123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestOwnerCheck_MissingResourceID(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(AccountIDKey), "user-123")
		c.Next()
	})
	r.Use(OwnerCheck())
	r.GET("/resource/", func(c *gin.Context) {}) // no :id param

	req := httptest.NewRequest("GET", "/resource/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
