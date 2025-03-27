package event

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/middleware"
	"github.com/neo-vai/go-events/internal/model/event"
	event_service "github.com/neo-vai/go-events/internal/service/event"
)

type EventService interface {
	CreateEvent(ctx context.Context, ev *event.Event) error
	GetByID(ctx context.Context, id string) (*event.Event, error)
	// ListEvents is kept for backward compatibility; prefer ListEventsPaginated.
	ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error)
	ListEventsPaginated(ctx context.Context, accountID string, page, limit int, sort, order, username, apiKeyID, searchQuery string) ([]*event.Event, int64, error)
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
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /events [post]
func (h *Handler) CreateEvent(c *gin.Context) {
	authenticatedAccountID, exists := c.Get(string(middleware.AccountIDKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	ev := &event.Event{
		AccountID: authenticatedAccountID.(string),
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
// @Summary      List events with optional filters and pagination
// @Tags         event
// @Produce      json
// @Param        page        query int    false "Page number (starts from 1)"
// @Param        limit       query int    false "Items per page (max 1000)"
// @Param        sort        query string false "Sort field (createdAt, username, name)"
// @Param        order       query string false "Sort order (ASC, DESC)"
// @Param        user        query string false "Filter by username"
// @Param        api_key_id  query string false "Filter by API key ID"
// @Param        q           query string false "Search query (username, name, payload)"
// @Success      200 {array} EventResponse
// @Header       200 {integer} X-Total-Count "Total number of items (only when paginated)"
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /events [get]
func (h *Handler) ListEvents(c *gin.Context) {
	authenticatedAccountID, exists := c.Get(string(middleware.AccountIDKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	accountID := authenticatedAccountID.(string)

	// Parse pagination parameters.
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))

	// Parse sorting.
	sort := c.DefaultQuery("sort", "createdAt")
	order := c.DefaultQuery("order", "DESC")

	// Parse filters.
	username := c.Query("user")
	apiKeyID := c.Query("api_key_id")
	searchQuery := c.Query("q")

	events, total, err := h.service.ListEventsPaginated(
		c.Request.Context(),
		accountID,
		page,
		limit,
		sort,
		order,
		username,
		apiKeyID,
		searchQuery,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]EventResponse, len(events))
	for i, ev := range events {
		resp[i] = ToEventResponse(ev)
	}

	// Set X-Total-Count header only when pagination is enabled.
	if limit > 0 {
		c.Header("X-Total-Count", strconv.FormatInt(total, 10))
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
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /events/{id} [get]
func (h *Handler) GetEvent(c *gin.Context) {
	// Check authentication first
	authenticatedAccountID, exists := c.Get(string(middleware.AccountIDKey))
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	id := c.Param("id")
	ev, err := h.service.GetByID(c, id)
	if err != nil {
		if errors.Is(err, event_service.ErrEventNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Additional check: event must belong to the authenticated account
	if ev.AccountID != authenticatedAccountID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, ToEventResponse(ev))
}
