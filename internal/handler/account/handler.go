package account

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/middleware"
	"github.com/neo-vai/go-events/internal/model/account"
	account_service "github.com/neo-vai/go-events/internal/service/account"
)

type AccountService interface {
	CreateAccount(ctx context.Context, acc *account.Account, password string) error
	GetByID(ctx context.Context, id string) (*account.Account, error)
	UpdateAccount(ctx context.Context, acc *account.Account) error
	DeleteAccount(ctx context.Context, id string) error
}

type Handler struct {
	service AccountService
}

func NewHandler(service AccountService) *Handler {
	return &Handler{service: service}
}

// CreateAccount godoc
// @Summary      Create account
// @Description  Creates a new account with hashed password
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        body body CreateAccountRequest true "Account data"
// @Success      201 {object} AccountResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      409 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /accounts [post]
func (h *Handler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	acc := &account.Account{
		ID:    uuid.New(),
		Email: req.Email,
	}

	err := h.service.CreateAccount(c, acc, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if errors.Is(err, account_service.ErrEmailAlreadyExists) {
			status = http.StatusConflict
			message = "email already exists"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusCreated, ToAccountResponse(acc))
}

// GetCurrentAccount godoc
// @Summary      Get current account details
// @Tags         account
// @Produce      json
// @Success      200 {object} AccountResponse
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /account [get]
func (h *Handler) GetCurrentAccount(c *gin.Context) {
	accountID, exists := c.Get(string(middleware.AccountIDKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	acc, err := h.service.GetByID(c, accountID.(string))
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if errors.Is(err, account_service.ErrAccountNotFound) ||
			errors.Is(err, account_service.ErrInvalidAccountID) {
			status = http.StatusNotFound
			message = "account not found"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	c.JSON(http.StatusOK, ToAccountResponse(acc))
}

// UpdateCurrentAccount godoc
// @Summary      Update current account
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        body body UpdateAccountRequest true "Fields to update"
// @Success      200 {object} AccountResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      409 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /account [patch]
func (h *Handler) UpdateCurrentAccount(c *gin.Context) {
	accountID, exists := c.Get(string(middleware.AccountIDKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	existing, err := h.service.GetByID(c, accountID.(string))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	if req.Email != "" {
		existing.Email = req.Email
	}

	err = h.service.UpdateAccount(c, existing)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if errors.Is(err, account_service.ErrEmailAlreadyExists) {
			status = http.StatusConflict
			message = "email already exists"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusOK, ToAccountResponse(existing))
}

// DeleteCurrentAccount godoc
// @Summary      Delete current account
// @Tags         account
// @Produce      json
// @Success      204 "No Content"
// @Failure      401 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /account [delete]
func (h *Handler) DeleteCurrentAccount(c *gin.Context) {
	accountID, exists := c.Get(string(middleware.AccountIDKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.service.DeleteAccount(c, accountID.(string))
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if errors.Is(err, account_service.ErrAccountNotFound) ||
			errors.Is(err, account_service.ErrInvalidAccountID) {
			status = http.StatusNotFound
			message = "account not found"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	c.Status(http.StatusNoContent)
}
