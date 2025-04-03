package middleware

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
)

type contextKey string

const (
	AccountIDKey contextKey = "accountID"
	RequestIDKey contextKey = "requestID"
	RoleKey      contextKey = "role"
	APIKeyIDKey  contextKey = "apiKeyID"
)

// AccountGetter retrieves an account by its ID. Used for active status check.
type AccountGetter func(ctx *gin.Context, accountID string) (*account.Account, error)

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
// It requires jwtSecret for JWT validation and an accountGetter to verify the account is active.
func UniversalAuth(
	apiKeyValidator func(ctx *gin.Context, key string) (accountID, role, apiKeyID string, err error),
	jwtSecret string,
	accountGetter AccountGetter,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		var accountID, role, apiKeyID string
		var err error

		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
				claims, jwtErr := ValidateJWT(parts[1], jwtSecret)
				if jwtErr == nil {
					accountID = claims.AccountID
					role = claims.Role
				} else {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
					return
				}
			}
		}

		if accountID == "" {
			apiKey := c.GetHeader("X-API-Key")
			if apiKey != "" && apiKeyValidator != nil {
				accountID, role, apiKeyID, err = apiKeyValidator(c, apiKey)
				if err != nil || accountID == "" {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
					return
				}
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
		}

		// Verify the account exists and is active.
		if accountGetter != nil {
			acc, accErr := accountGetter(c, accountID)
			if accErr != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "account not found"})
				return
			}
			if !acc.Active {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
				return
			}
			// Optionally update role from the fresh account data (if not set by JWT).
			if role == "" {
				role = acc.Role
			}
		}

		c.Set(string(AccountIDKey), accountID)
		c.Set(string(RoleKey), role)
		if apiKeyID != "" {
			c.Set(string(APIKeyIDKey), apiKeyID)
		}
		c.Next()
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

// CORS middleware with configurable allowed origins.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		allowOrigin := ""
		allowCredentials := false

		// If exactly one entry "*", allow all origins without credentials.
		if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
			allowOrigin = "*"
			allowCredentials = false
		} else if len(allowedOrigins) > 0 {
			// Otherwise check against the list.
			for _, o := range allowedOrigins {
				if o == origin {
					allowOrigin = origin
					allowCredentials = true
					break
				}
			}
		}
		// If allowedOrigins is empty, we leave allowOrigin empty (no CORS header).

		if allowOrigin != "" {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		}
		if allowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key, X-Request-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Range, X-Total-Count, X-Request-ID")

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

// ValidationErrorHandler intercepts validation errors and returns a structured response.
func ValidationErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		// If response already written, do nothing
		if c.Writer.Written() {
			return
		}

		var validationErrors []gin.H
		for _, e := range c.Errors {
			if ve, ok := e.Err.(validator.ValidationErrors); ok {
				for _, fe := range ve {
					validationErrors = append(validationErrors, gin.H{
						"field":   fe.Field(),
						"message": formatValidationError(fe),
					})
				}
			} else {
				// For non-validation errors (e.g., JSON syntax), return a generic bad request
				c.JSON(http.StatusBadRequest, gin.H{"error": e.Err.Error()})
				return
			}
		}

		if len(validationErrors) > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"errors": validationErrors})
		}
	}
}

func formatValidationError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "json":
		return "Invalid JSON format"
	case "strongpassword":
		return "Password must be at least 8 characters and contain both letters and digits"
	case "alphanumdash":
		return "Only letters, numbers, hyphens and underscores are allowed"
	default:
		return fe.Error()
	}
}
