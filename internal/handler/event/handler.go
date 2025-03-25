package event

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/model/event"
)

type EventService interface {
	CreateEvent(ctx context.Context, ev *event.Event) error
	GetByID(ctx context.Context, id string) (*event.Event, error)
	ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error)
}

type Handler struct {
	service EventService
}

func NewHandler(service EventService) *Handler {
	return &Handler{service: service}
}

// CreateEvent godoc
// @Summary      Create event
// @Tags         event
// @Accept       json
// @Produce      json
// @Param        body body CreateEventRequest true "Event data"
// @Success      201 {object} EventResponse
// @Failure      400 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /events [post]
func (h *Handler) CreateEvent(c *gin.Context) {
	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ev := &event.Event{
		AccountID: req.AccountID,
		Username:  req.Username,
		APIKeyID:  req.APIKeyID,
		Name:      req.Name,
		Payload:   req.Payload,
	}
	if err := h.service.CreateEvent(c, ev); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ToEventResponse(ev))
}

// ListEvents godoc
// @Summary      List events with filters
// @Tags         event
// @Produce      json
// @Param        account_id query string false "Filter by account ID"
// @Param        user query string false "Filter by username"
// @Param        api_key_id query string false "Filter by API key ID"
// @Success      200 {array} EventResponse
// @Failure      500 {object} map[string]interface{}
// @Router       /events [get]
func (h *Handler) ListEvents(c *gin.Context) {
	accountID := c.Query("account_id")
	username := c.Query("user")
	apiKeyID := c.Query("api_key_id")

	events, err := h.service.ListEvents(c, accountID, username, apiKeyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := make([]EventResponse, len(events))
	for i, ev := range events {
		resp[i] = ToEventResponse(ev)
	}
	c.JSON(http.StatusOK, resp)
}

// GetEvent godoc
// @Summary      Get event by ID
// @Tags         event
// @Produce      json
// @Param        id path string true "Event ID"
// @Success      200 {object} EventResponse
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Router       /events/{id} [get]
func (h *Handler) GetEvent(c *gin.Context) {
	id := c.Param("id")
	ev, err := h.service.GetByID(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}
	c.JSON(http.StatusOK, ToEventResponse(ev))
}
