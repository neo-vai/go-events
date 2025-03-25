package apikey

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/model/apikey"
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
// @Description  Creates a new API key for the account
// @Tags         apikey
// @Produce      json
// @Param        id path string true "Account ID"
// @Success      201 {object} APIKeyResponse
// @Failure      500 {object} map[string]interface{}
// @Router       /accounts/{id}/keys [post]
func (h *Handler) GenerateAPIKey(c *gin.Context) {
	accountID := c.Param("id")
	key, err := h.service.Generate(c, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ToAPIKeyResponse(key))
}

// ListAPIKeys godoc
// @Summary      List API keys
// @Description  Returns all API keys belonging to the account
// @Tags         apikey
// @Produce      json
// @Param        id path string true "Account ID"
// @Success      200 {array} APIKeyResponse
// @Failure      500 {object} map[string]interface{}
// @Router       /accounts/{id}/keys [get]
func (h *Handler) ListAPIKeys(c *gin.Context) {
	accountID := c.Param("id")
	keys, err := h.service.ListByAccount(c, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]APIKeyResponse, len(keys))
	for i, k := range keys {
		resp[i] = ToAPIKeyResponse(k)
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
// @Failure      500 {object} map[string]interface{}
// @Router       /accounts/{id}/keys/{key_id} [patch]
func (h *Handler) UpdateAPIKeyActive(c *gin.Context) {
	keyID := c.Param("key_id")
	var req UpdateAPIKeyActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.service.UpdateActive(c, keyID, req.Active); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	key, err := h.service.GetByID(c, keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ToAPIKeyResponse(key))
}

// DeleteAPIKey godoc
// @Summary      Delete API key
// @Tags         apikey
// @Produce      json
// @Param        id path string true "Account ID"
// @Param        key_id path string true "API Key ID"
// @Success      204 "No Content"
// @Failure      500 {object} map[string]interface{}
// @Router       /accounts/{id}/keys/{key_id} [delete]
func (h *Handler) DeleteAPIKey(c *gin.Context) {
	keyID := c.Param("key_id")
	if err := h.service.Delete(c, keyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
