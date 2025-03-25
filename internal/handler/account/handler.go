package account

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
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
// @Failure      500 {object} map[string]interface{}
// @Router       /accounts [post]
func (h *Handler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	acc := &account.Account{
		ID:    uuid.New(),
		Name:  req.Name,
		Email: req.Email,
		Login: req.Login,
	}

	if err := h.service.CreateAccount(c, acc, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Router       /accounts/{id} [get]
func (h *Handler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	acc, err := h.service.GetByID(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
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
// @Failure      500 {object} map[string]interface{}
// @Router       /accounts/{id} [put]
func (h *Handler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	if err := h.service.UpdateAccount(c, existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
// @Router       /accounts/{id} [delete]
func (h *Handler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteAccount(c, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
