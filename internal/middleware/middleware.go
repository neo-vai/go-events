package middleware

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type contextKey string

const (
	AccountIDKey contextKey = "accountID"
	RequestIDKey contextKey = "requestID"
)

type Claims struct {
	AccountID string `json:"account_id"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token.
func GenerateJWT(accountID string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-change-me"
	}
	expiresHours := 24
	if val := os.Getenv("JWT_EXPIRES_HOURS"); val != "" {
		// simple conversion; you may use strconv.Atoi
	}
	claims := Claims{
		AccountID: accountID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateJWT parses and validates the token.
func ValidateJWT(tokenString string) (*Claims, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "default-secret-change-me"
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

// JWTAuth middleware (kept for backward compatibility, but not used in router now).
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}
		claims, err := ValidateJWT(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(string(AccountIDKey), claims.AccountID)
		c.Next()
	}
}

// UniversalAuth tries JWT first, then API key.
// UniversalAuth tries JWT first, then API key.
func UniversalAuth(apiKeyValidator func(ctx *gin.Context, key string) (string, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Try JWT from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				claims, err := ValidateJWT(parts[1])
				if err == nil {
					c.Set(string(AccountIDKey), claims.AccountID)
					c.Next()
					return
				}
			}
		}

		// 2. Try API key from X-API-Key header
		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" && apiKeyValidator != nil {
			accountID, err := apiKeyValidator(c, apiKey)
			if err == nil {
				c.Set(string(AccountIDKey), accountID)
				c.Next()
				return
			}
		}

		// 3. No valid credentials
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	}
}

// RequestID middleware.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(string(RequestIDKey), requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// StructuredLogger logs requests in JSON.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		if raw != "" {
			path = path + "?" + raw
		}
		c.Next()
		latency := time.Since(start)
		status := c.Writer.Status()
		requestID, _ := c.Get(string(RequestIDKey))
		entry := map[string]interface{}{
			"level":      "info",
			"time":       time.Now().UTC().Format(time.RFC3339),
			"status":     status,
			"method":     c.Request.Method,
			"path":       path,
			"ip":         c.ClientIP(),
			"latency_ms": latency.Milliseconds(),
			"request_id": requestID,
		}
		enc, _ := json.Marshal(entry)
		gin.DefaultWriter.Write(append(enc, '\n'))
	}
}

var (
	limiters = make(map[string]*rate.Limiter)
	mu       sync.RWMutex
)

func RateLimiter(requests int, duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.RLock()
		limiter, exists := limiters[ip]
		mu.RUnlock()
		if !exists {
			mu.Lock()
			limiter = rate.NewLimiter(rate.Limit(requests)/rate.Limit(duration.Seconds()), requests)
			limiters[ip] = limiter
			mu.Unlock()
		}
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.Next()
	}
}

// CORS middleware.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// OwnerCheck ensures the authenticated account matches the resource ID.
func OwnerCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		accountID, exists := c.Get(string(AccountIDKey))
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		resourceID := c.Param("id")
		if resourceID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "resource id missing"})
			return
		}
		if accountID != resourceID {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "access denied"})
			return
		}
		c.Next()
	}
}
