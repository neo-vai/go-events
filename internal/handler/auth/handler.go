package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/middleware"
	"github.com/neo-vai/go-events/internal/model/account"
	account_service "github.com/neo-vai/go-events/internal/service/account"
)

type AccountService interface {
	GetByEmail(ctx context.Context, email string) (*account.Account, error)
	VerifyPassword(ctx context.Context, email, password string) (bool, error)
}

type Handler struct {
	accountSvc      AccountService
	jwtSecret       string
	jwtExpiresHours int
}

func NewHandler(accountSvc AccountService, jwtSecret string, jwtExpiresHours int) *Handler {
	return &Handler{
		accountSvc:      accountSvc,
		jwtSecret:       jwtSecret,
		jwtExpiresHours: jwtExpiresHours,
	}
}

// LoginRequest contains the credentials for authentication.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"Passw0rd!"`
}

// LoginResponse contains the JWT token and account ID.
type LoginResponse struct {
	Token     string `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	AccountID string `json:"accountId" example:"550e8400-e29b-41d4-a716-446655440000"`
}

// Login godoc
// @Summary      Login and obtain JWT token
// @Description  Authenticates a user with email and password, returning a JWT token for use in subsequent requests.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Login credentials"
// @Success      200 {object} LoginResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{} "Invalid credentials"
// @Failure      403 {object} map[string]interface{} "Account is inactive"
// @Failure      500 {object} map[string]interface{}
// @Router       /login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}
	valid, err := h.accountSvc.VerifyPassword(c, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, account_service.ErrAccountInactive) {
			c.JSON(http.StatusForbidden, gin.H{"error": "account is inactive"})
			return
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	if !valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}
	acc, err := h.accountSvc.GetByEmail(c, req.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve account"})
		return
	}
	token, err := middleware.GenerateJWT(acc.ID.String(), acc.Role, h.jwtSecret, h.jwtExpiresHours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		AccountID: acc.ID.String(),
	})
}
