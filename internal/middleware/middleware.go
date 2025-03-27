package middleware

import (
	"log/slog"
	"net/http"
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
	RoleKey      contextKey = "role"
)

type Claims struct {
	AccountID string `json:"account_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token using the provided secret and expiration hours.
func GenerateJWT(accountID, role, jwtSecret string, expiresHours int) (string, error) {
	claims := Claims{
		AccountID: accountID,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// ValidateJWT parses and validates the token using the provided secret.
func ValidateJWT(tokenString, jwtSecret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}

// JWTAuth middleware (kept for backward compatibility, but not used in router now).
func JWTAuth(jwtSecret string) gin.HandlerFunc {
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
		claims, err := ValidateJWT(parts[1], jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		c.Set(string(AccountIDKey), claims.AccountID)
		c.Set(string(RoleKey), claims.Role)
		c.Next()
	}
}

// UniversalAuth tries JWT first if Authorization header is present, otherwise falls back to API key.
// It requires jwtSecret for JWT validation.
func UniversalAuth(apiKeyValidator func(ctx *gin.Context, key string) (accountID, role string, err error), jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				claims, err := ValidateJWT(parts[1], jwtSecret)
				if err == nil {
					c.Set(string(AccountIDKey), claims.AccountID)
					c.Set(string(RoleKey), claims.Role)
					c.Next()
					return
				}
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
				return
			}
		}

		apiKey := c.GetHeader("X-API-Key")
		if apiKey != "" && apiKeyValidator != nil {
			accountID, role, err := apiKeyValidator(c, apiKey)
			if err == nil && accountID != "" {
				c.Set(string(AccountIDKey), accountID)
				c.Set(string(RoleKey), role)
				c.Next()
				return
			}
		}

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

// StructuredLogger logs requests using slog.
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

		slog.LogAttrs(c.Request.Context(), slog.LevelInfo, "http request",
			slog.Int("status", status),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.String("ip", c.ClientIP()),
			slog.Int64("latency_ms", latency.Milliseconds()),
			slog.String("request_id", requestID.(string)),
		)
	}
}

// NewRateLimiter creates a rate limiter middleware with configurable requests per duration.
func NewRateLimiter(requests int, per time.Duration) gin.HandlerFunc {
	var (
		limiters = make(map[string]*rate.Limiter)
		mu       sync.RWMutex
	)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.RLock()
		limiter, exists := limiters[ip]
		mu.RUnlock()
		if !exists {
			mu.Lock()
			limiter = rate.NewLimiter(rate.Every(per/time.Duration(requests)), requests)
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

// CORS middleware with configurable allowed origins.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowOrigin := ""
		if len(allowedOrigins) == 0 || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*") {
			allowOrigin = "*"
		} else {
			for _, o := range allowedOrigins {
				if o == origin {
					allowOrigin = origin
					break
				}
			}
		}
		if allowOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		}
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

// RequireAdmin checks if the authenticated user has admin role.
func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get(string(RoleKey))
		if !exists || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}
