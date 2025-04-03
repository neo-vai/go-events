package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"log/slog"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type MockAccountGetter struct {
	mock.Mock
}

func (m *MockAccountGetter) GetAccount(ctx *gin.Context, accountID string) (*account.Account, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}

type MockAPIKeyValidator struct {
	mock.Mock
}

func (m *MockAPIKeyValidator) Validate(ctx *gin.Context, key string) (accountID, role, apiKeyID string, err error) {
	args := m.Called(ctx, key)
	return args.String(0), args.String(1), args.String(2), args.Error(3)
}

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(string(RequestIDKey)))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	assert.NotEmpty(t, requestID)
	assert.Equal(t, requestID, w.Body.String())
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequestID_PropagatesExisting(t *testing.T) {
	r := gin.New()
	r.Use(RequestID())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, c.GetString(string(RequestIDKey)))
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-request-id-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "custom-request-id-123", w.Header().Get("X-Request-ID"))
	assert.Equal(t, "custom-request-id-123", w.Body.String())
}

func TestStructuredLogger_LogsRequest(t *testing.T) {
	// Create a buffer to capture log output
	buf := &strings.Builder{}
	// Save the default logger and restore after test
	originalLogger := slog.Default()
	defer slog.SetDefault(originalLogger)
	// Set a new logger that writes to our buffer
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))

	r := gin.New()
	r.Use(RequestID())
	r.Use(StructuredLogger())
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	req.Header.Set("X-Request-ID", "log-test-123")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// The log output is a single line of JSON. Trim any surrounding whitespace.
	logLine := strings.TrimSpace(buf.String())
	require.NotEmpty(t, logLine, "log output should not be empty")

	var logEntry map[string]interface{}
	err := json.Unmarshal([]byte(logLine), &logEntry)
	require.NoError(t, err)

	assert.Equal(t, "INFO", logEntry["level"])
	assert.Equal(t, float64(200), logEntry["status"])
	assert.Equal(t, "GET", logEntry["method"])
	assert.Equal(t, "/test?foo=bar", logEntry["path"])
	assert.Equal(t, "log-test-123", logEntry["request_id"])
	assert.NotNil(t, logEntry["latency_ms"])
}

func TestCORS_AllowsAllOrigins(t *testing.T) {
	r := gin.New()
	r.Use(CORS([]string{"*"}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.OPTIONS("/test", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
}

func TestCORS_RestrictedOrigins(t *testing.T) {
	r := gin.New()
	r.Use(CORS([]string{"https://trusted.com", "https://app.trusted.com"}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://trusted.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://trusted.com", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))

	req = httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://untrusted.com")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_EmptyAllowedOrigins(t *testing.T) {
	r := gin.New()
	r.Use(CORS([]string{}))
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestGenerateJWT_And_ValidateJWT(t *testing.T) {
	jwtSecret := "test-secret-key-with-minimum-32-characters-long"
	accountID := "acc-123"
	role := "user"

	token, err := GenerateJWT(accountID, role, jwtSecret, 1)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ValidateJWT(token, jwtSecret)
	require.NoError(t, err)
	assert.Equal(t, accountID, claims.AccountID)
	assert.Equal(t, role, claims.Role)
	assert.NotNil(t, claims.ExpiresAt)
	assert.NotNil(t, claims.IssuedAt)
}

func TestValidateJWT_InvalidSignature(t *testing.T) {
	jwtSecret := "correct-secret-key-with-minimum-32-chars"
	token, err := GenerateJWT("acc-123", "user", jwtSecret, 1)
	require.NoError(t, err)

	_, err = ValidateJWT(token, "wrong-secret-key-with-minimum-32-chars")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature is invalid")
}

func TestValidateJWT_ExpiredToken(t *testing.T) {
	jwtSecret := "test-secret-key-with-minimum-32-characters-long"

	claims := Claims{
		AccountID: "acc-123",
		Role:      "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
	require.NoError(t, err)

	_, err = ValidateJWT(tokenString, jwtSecret)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestValidateJWT_MalformedToken(t *testing.T) {
	jwtSecret := "test-secret-key-with-minimum-32-characters-long"

	_, err := ValidateJWT("not-a-valid-jwt", jwtSecret)
	assert.Error(t, err)
}

func TestJWTAuth_Success(t *testing.T) {
	jwtSecret := "test-secret-key-with-minimum-32-characters-long"
	token, err := GenerateJWT("acc-123", "admin", jwtSecret, 1)
	require.NoError(t, err)

	r := gin.New()
	r.Use(JWTAuth(jwtSecret))
	r.GET("/protected", func(c *gin.Context) {
		accountID, _ := c.Get(string(AccountIDKey))
		role, _ := c.Get(string(RoleKey))
		c.JSON(http.StatusOK, gin.H{
			"account_id": accountID,
			"role":       role,
		})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "acc-123", resp["account_id"])
	assert.Equal(t, "admin", resp["role"])
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	r := gin.New()
	r.Use(JWTAuth("secret"))
	r.GET("/protected", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "missing authorization header", errResp["error"])
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	r := gin.New()
	r.Use(JWTAuth("secret"))
	r.GET("/protected", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "InvalidFormat token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid authorization format", errResp["error"])
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	r := gin.New()
	r.Use(JWTAuth("secret"))
	r.GET("/protected", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid or expired token", errResp["error"])
}

func TestUniversalAuth_JWTSuccess(t *testing.T) {
	jwtSecret := "test-secret-key-with-minimum-32-characters-long"
	token, err := GenerateJWT("acc-123", "user", jwtSecret, 1)
	require.NoError(t, err)

	validator := func(ctx *gin.Context, key string) (string, string, string, error) {
		return "", "", "", nil
	}
	accountGetter := func(ctx *gin.Context, accountID string) (*account.Account, error) {
		return &account.Account{ID: [16]byte{}, Email: "test@example.com", Role: "user", Active: true}, nil
	}

	r := gin.New()
	r.Use(UniversalAuth(validator, jwtSecret, accountGetter))
	r.GET("/protected", func(c *gin.Context) {
		accountID, _ := c.Get(string(AccountIDKey))
		role, _ := c.Get(string(RoleKey))
		c.JSON(http.StatusOK, gin.H{
			"account_id": accountID,
			"role":       role,
		})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "acc-123", resp["account_id"])
	assert.Equal(t, "user", resp["role"])
}

func TestUniversalAuth_APIKeySuccess(t *testing.T) {
	jwtSecret := "test-secret-key-with-minimum-32-characters-long"

	validator := func(ctx *gin.Context, key string) (string, string, string, error) {
		if key == "valid-api-key" {
			return "acc-456", "user", "key-789", nil
		}
		return "", "", "", nil
	}
	accountGetter := func(ctx *gin.Context, accountID string) (*account.Account, error) {
		return &account.Account{ID: [16]byte{}, Email: "test@example.com", Role: "user", Active: true}, nil
	}

	r := gin.New()
	r.Use(UniversalAuth(validator, jwtSecret, accountGetter))
	r.GET("/protected", func(c *gin.Context) {
		accountID, _ := c.Get(string(AccountIDKey))
		role, _ := c.Get(string(RoleKey))
		apiKeyID, _ := c.Get(string(APIKeyIDKey))
		c.JSON(http.StatusOK, gin.H{
			"account_id": accountID,
			"role":       role,
			"api_key_id": apiKeyID,
		})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-API-Key", "valid-api-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "acc-456", resp["account_id"])
	assert.Equal(t, "user", resp["role"])
	assert.Equal(t, "key-789", resp["api_key_id"])
}

func TestUniversalAuth_NoCredentials(t *testing.T) {
	jwtSecret := "test-secret-key-with-minimum-32-characters-long"
	validator := func(ctx *gin.Context, key string) (string, string, string, error) {
		return "", "", "", nil
	}
	accountGetter := func(ctx *gin.Context, accountID string) (*account.Account, error) {
		return nil, nil
	}

	r := gin.New()
	r.Use(UniversalAuth(validator, jwtSecret, accountGetter))
	r.GET("/protected", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestUniversalAuth_InvalidJWT_PreventsAPIKeyFallback(t *testing.T) {
	jwtSecret := "test-secret-key-with-minimum-32-characters-long"
	validatorCalled := false
	validator := func(ctx *gin.Context, key string) (string, string, string, error) {
		validatorCalled = true
		return "acc-api", "user", "key-123", nil
	}
	accountGetter := func(ctx *gin.Context, accountID string) (*account.Account, error) {
		return &account.Account{ID: [16]byte{}, Email: "test@example.com", Role: "user", Active: true}, nil
	}

	r := gin.New()
	r.Use(UniversalAuth(validator, jwtSecret, accountGetter))
	r.GET("/protected", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer invalid.token")
	req.Header.Set("X-API-Key", "valid-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.False(t, validatorCalled, "API key validator should not be called when invalid JWT is present")

	var errResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid or expired token", errResp["error"])
}

func TestUniversalAuth_InactiveAccount(t *testing.T) {
	jwtSecret := "test-secret-key-with-minimum-32-characters-long"
	token, err := GenerateJWT("acc-inactive", "user", jwtSecret, 1)
	require.NoError(t, err)

	validator := func(ctx *gin.Context, key string) (string, string, string, error) {
		return "", "", "", nil
	}
	accountGetter := func(ctx *gin.Context, accountID string) (*account.Account, error) {
		return &account.Account{ID: [16]byte{}, Email: "test@example.com", Role: "user", Active: false}, nil
	}

	r := gin.New()
	r.Use(UniversalAuth(validator, jwtSecret, accountGetter))
	r.GET("/protected", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "account is inactive", errResp["error"])
}

func TestOwnerCheck_AllowsOwner(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(AccountIDKey), "user-123")
		c.Next()
	})
	r.Use(OwnerCheck())
	r.GET("/resource/:id", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
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

	var errResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "access denied", errResp["error"])
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
	r.GET("/resource/", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/resource/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "resource id missing", errResp["error"])
}

func TestRequireAdmin_AllowsAdmin(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(RoleKey), "admin")
		c.Next()
	})
	r.Use(RequireAdmin())
	r.GET("/admin", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequireAdmin_DeniesNonAdmin(t *testing.T) {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(RoleKey), "user")
		c.Next()
	})
	r.Use(RequireAdmin())
	r.GET("/admin", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResp map[string]string
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "admin access required", errResp["error"])
}

func TestRequireAdmin_MissingRole(t *testing.T) {
	r := gin.New()
	r.Use(RequireAdmin())
	r.GET("/admin", func(c *gin.Context) {})

	req := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
}
