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
	GetByLogin(ctx context.Context, login string) (*account.Account, error)
	VerifyPassword(ctx context.Context, login, password string) (bool, error)
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

type LoginRequest struct {
	Login    string `json:"login" binding:"required,alphanumdash"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	AccountID string `json:"accountId"`
}

// Login godoc
// @Summary      Login and get JWT token
// @Description  Authenticates user and returns JWT token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body LoginRequest true "Login credentials"
// @Success      200 {object} LoginResponse
// @Failure      401 {object} map[string]interface{}
// @Failure      400 {object} map[string]interface{}
// @Router       /login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}
	valid, err := h.accountSvc.VerifyPassword(c, req.Login, req.Password)
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
	acc, err := h.accountSvc.GetByLogin(c, req.Login)
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
