package account

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		Name:  req.Name,
		Email: req.Email,
		Login: req.Login,
	}

	err := h.service.CreateAccount(c, acc, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		switch {
		case errors.Is(err, account_service.ErrEmailAlreadyExists):
			status = http.StatusConflict
			message = "email already exists"
		case errors.Is(err, account_service.ErrLoginAlreadyExists):
			status = http.StatusConflict
			message = "login already exists"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusCreated, ToAccountResponse(acc))
}

// GetAccount godoc
// @Summary      Get account by ID
// @Tags         account
// @Produce      json
// @Param        id path string true "Account ID"
// @Success      200 {object} AccountResponse
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /accounts/{id} [get]
func (h *Handler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	acc, err := h.service.GetByID(c, id)
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

// UpdateAccount godoc
// @Summary      Update account
// @Tags         account
// @Accept       json
// @Produce      json
// @Param        id path string true "Account ID"
// @Param        body body UpdateAccountRequest true "Fields to update"
// @Success      200 {object} AccountResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      409 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /accounts/{id} [put]
func (h *Handler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	existing, err := h.service.GetByID(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Email != "" {
		existing.Email = req.Email
	}
	if req.Login != "" {
		existing.Login = req.Login
	}

	err = h.service.UpdateAccount(c, existing)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		switch {
		case errors.Is(err, account_service.ErrEmailAlreadyExists):
			status = http.StatusConflict
			message = "email already exists"
		case errors.Is(err, account_service.ErrLoginAlreadyExists):
			status = http.StatusConflict
			message = "login already exists"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}

	c.JSON(http.StatusOK, ToAccountResponse(existing))
}

// DeleteAccount godoc
// @Summary      Delete account
// @Tags         account
// @Produce      json
// @Param        id path string true "Account ID"
// @Success      204 "No Content"
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /accounts/{id} [delete]
func (h *Handler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	err := h.service.DeleteAccount(c, id)
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
