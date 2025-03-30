package event

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/broker"
	"github.com/neo-vai/go-events/internal/middleware"
	"github.com/neo-vai/go-events/internal/model/event"
	event_service "github.com/neo-vai/go-events/internal/service/event"
)

type EventService interface {
	GetByID(ctx context.Context, id string) (*event.Event, error)
	ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error)
	ListEventsPaginated(ctx context.Context, accountID string, page, limit int, sort, order, username, apiKeyID, searchQuery string) ([]*event.Event, int64, error)
}

type Handler struct {
	service   EventService
	publisher broker.Publisher
}

func NewHandler(service EventService, publisher broker.Publisher) *Handler {
	return &Handler{
		service:   service,
		publisher: publisher,
	}
}

// CreateEvent godoc
// @Summary      Create event
// @Description  Creates a new event. If authenticated with an API key, the key ID is automatically associated.
// @Tags         event
// @Accept       json
// @Produce      json
// @Param        body body CreateEventRequest true "Event data"
// @Success      202 {object} map[string]string
// @Failure      400 {object} map[string]interface{}
// @Failure      401 {object} map[string]interface{}
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

	apiKeyID, _ := c.Get(string(middleware.APIKeyIDKey))
	var apiKeyIDStr string
	if apiKeyID != nil {
		apiKeyIDStr = apiKeyID.(string)
	}

	ev := &event.Event{
		ID:        uuid.New().String(),
		AccountID: authenticatedAccountID.(string),
		Username:  req.Username,
		APIKeyID:  apiKeyIDStr,
		Name:      req.Name,
		Payload:   req.Payload,
		CreatedAt: time.Now(),
	}

	if err := h.publisher.PublishEvent(c, ev); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish event"})
		return
	}

	c.Header("Location", "/api/v1/events/"+ev.ID)
	c.JSON(http.StatusAccepted, gin.H{
		"id":     ev.ID,
		"status": "queued",
	})
}

// setContentRangeHeader sets the Content-Range header for paginated responses.
func setContentRangeHeader(c *gin.Context, resource string, page, limit int, total int64) {
	start := (page - 1) * limit
	end := start + limit - 1
	if total == 0 {
		c.Header("Content-Range", fmt.Sprintf("%s */0", resource))
		return
	}
	if int64(end) >= total {
		end = int(total) - 1
	}
	c.Header("Content-Range", fmt.Sprintf("%s %d-%d/%d", resource, start, end, total))
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
// @Header       200 {string} Content-Range "resources start-end/total"
// @Header       200 {integer} X-Total-Count "Total number of items (only when paginated)"
// @Failure      401 {object} map[string]interface{}
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

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "0"))

	sort := c.DefaultQuery("sort", "createdAt")
	order := c.DefaultQuery("order", "DESC")

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

	if limit > 0 {
		c.Header("X-Total-Count", strconv.FormatInt(total, 10))
		setContentRangeHeader(c, "events", page, limit, total)
	}
	c.JSON(http.StatusOK, resp)
}

// GetEvent godoc
// @Summary      Get event by ID
// @Tags         event
// @Produce      json
// @Param        id path string true "Event ID"
// @Success      200 {object} EventResponse
// @Failure      401 {object} map[string]interface{}
// @Failure      403 {object} map[string]interface{}
// @Failure      404 {object} map[string]interface{}
// @Failure      500 {object} map[string]interface{}
// @Security     BearerAuth
// @Security     ApiKeyAuth
// @Router       /events/{id} [get]
func (h *Handler) GetEvent(c *gin.Context) {
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

	if ev.AccountID != authenticatedAccountID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
		return
	}

	c.JSON(http.StatusOK, ToEventResponse(ev))
}
