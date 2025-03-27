package apikey

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/model/apikey"
	apikey_service "github.com/neo-vai/go-events/internal/service/apikey"
)

type APIKeyService interface {
	Generate(ctx context.Context, accountID string) (*apikey.APIKey, error)
	GetByID(ctx context.Context, id string) (*apikey.APIKey, error)
	ListByAccount(ctx context.Context, accountID string) ([]*apikey.APIKey, error)
	UpdateActive(ctx context.Context, id string, active bool) error
	Delete(ctx context.Context, id string) error
}

type Handler struct {
	service APIKeyService
}

func NewHandler(service APIKeyService) *Handler {
	return &Handler{service: service}
}

// GenerateAPIKey godoc
// @Summary      Generate API key
// @Description  Creates a new API key for the account. The full key is returned only once.
// @Tags         apikey
// @Produce      json
// @Param        id path string true "Account ID"
// @Success      201 {object} APIKeyResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      409 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /accounts/{id}/keys [post]
func (h *Handler) GenerateAPIKey(c *gin.Context) {
	accountID := c.Param("id")
	key, err := h.service.Generate(c, accountID)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		switch {
		case errors.Is(err, apikey_service.ErrInvalidAccountID):
			status = http.StatusBadRequest
			message = "invalid account ID"
		case errors.Is(err, apikey_service.ErrAccountNotFound):
			status = http.StatusNotFound
			message = "account not found"
		case errors.Is(err, apikey_service.ErrKeyAlreadyExists):
			status = http.StatusConflict
			message = "API key already exists"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	// Return full key only on creation
	c.JSON(http.StatusCreated, ToAPIKeyResponse(key, true))
}

// ListAPIKeys godoc
// @Summary      List API keys
// @Description  Returns all API keys belonging to the account (only key prefixes).
// @Tags         apikey
// @Produce      json
// @Param        id path string true "Account ID"
// @Success      200 {array} APIKeyResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /accounts/{id}/keys [get]
func (h *Handler) ListAPIKeys(c *gin.Context) {
	accountID := c.Param("id")
	keys, err := h.service.ListByAccount(c, accountID)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if errors.Is(err, apikey_service.ErrInvalidAccountID) {
			status = http.StatusBadRequest
			message = "invalid account ID"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	resp := make([]APIKeyResponse, len(keys))
	for i, k := range keys {
		resp[i] = ToAPIKeyResponse(k, false)
	}
	c.JSON(http.StatusOK, resp)
}

// UpdateAPIKeyActive godoc
// @Summary      Activate / deactivate API key
// @Tags         apikey
// @Accept       json
// @Produce      json
// @Param        id path string true "Account ID"
// @Param        key_id path string true "API Key ID"
// @Param        body body UpdateAPIKeyActiveRequest true "Active status"
// @Success      200 {object} APIKeyResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /accounts/{id}/keys/{key_id} [patch]
func (h *Handler) UpdateAPIKeyActive(c *gin.Context) {
	accountID := c.Param("id")
	keyID := c.Param("key_id")

	// Verify that the key belongs to the account specified in the URL.
	key, err := h.service.GetByID(c, keyID)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if errors.Is(err, apikey_service.ErrInvalidKeyID) {
			status = http.StatusBadRequest
			message = "invalid key ID"
		} else if errors.Is(err, apikey_service.ErrKeyNotFound) {
			status = http.StatusNotFound
			message = "API key not found"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	if key.AccountID.String() != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "API key does not belong to this account"})
		return
	}

	var req UpdateAPIKeyActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateActive(c, keyID, *req.Active); err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		switch {
		case errors.Is(err, apikey_service.ErrInvalidKeyID):
			status = http.StatusBadRequest
			message = "invalid key ID"
		case errors.Is(err, apikey_service.ErrKeyNotFound):
			status = http.StatusNotFound
			message = "API key not found"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	// Retrieve the updated key to return.
	updatedKey, err := h.service.GetByID(c, keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch updated key"})
		return
	}
	c.JSON(http.StatusOK, ToAPIKeyResponse(updatedKey, false))
}

// DeleteAPIKey godoc
// @Summary      Delete API key
// @Tags         apikey
// @Produce      json
// @Param        id path string true "Account ID"
// @Param        key_id path string true "API Key ID"
// @Success      204 "No Content"
// @Failure      400 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /accounts/{id}/keys/{key_id} [delete]
func (h *Handler) DeleteAPIKey(c *gin.Context) {
	accountID := c.Param("id")
	keyID := c.Param("key_id")

	// Verify that the key belongs to the account specified in the URL.
	key, err := h.service.GetByID(c, keyID)
	if err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		if errors.Is(err, apikey_service.ErrInvalidKeyID) {
			status = http.StatusBadRequest
			message = "invalid key ID"
		} else if errors.Is(err, apikey_service.ErrKeyNotFound) {
			status = http.StatusNotFound
			message = "API key not found"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	if key.AccountID.String() != accountID {
		c.JSON(http.StatusForbidden, gin.H{"error": "API key does not belong to this account"})
		return
	}

	if err := h.service.Delete(c, keyID); err != nil {
		status := http.StatusInternalServerError
		message := err.Error()
		switch {
		case errors.Is(err, apikey_service.ErrInvalidKeyID):
			status = http.StatusBadRequest
			message = "invalid key ID"
		case errors.Is(err, apikey_service.ErrKeyNotFound):
			status = http.StatusNotFound
			message = "API key not found"
		}
		c.JSON(status, gin.H{"error": message})
		return
	}
	c.Status(http.StatusNoContent)
}
