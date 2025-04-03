package admin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/neo-vai/go-events/internal/model/event"
	"github.com/neo-vai/go-events/internal/pagination"
)

type AccountService interface {
	CreateAccount(ctx context.Context, acc *account.Account, password string) error
	GetByID(ctx context.Context, id string) (*account.Account, error)
	UpdateAccountAdmin(ctx context.Context, id string, updates map[string]interface{}) (*account.Account, error)
	DeleteAccount(ctx context.Context, id string) error
	ListAccounts(ctx context.Context, offset, limit int, sort, order string, filters map[string]interface{}) ([]*account.Account, int64, error)
}

type EventService interface {
	GetByID(ctx context.Context, id string) (*event.Event, error)
	ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error)
	ListAllEvents(ctx context.Context, offset, limit int, sort, order string, filters map[string]interface{}) ([]*event.Event, int64, error)
}

type APIKeyService interface {
	GetByID(ctx context.Context, id string) (*apikey.APIKey, error)
	UpdateActive(ctx context.Context, id string, active bool) error
	Delete(ctx context.Context, id string) error
	ListAllAPIKeys(ctx context.Context, offset, limit int, sort, order string, filters map[string]interface{}) ([]*apikey.APIKey, int64, error)
}

type StatsService interface {
	GetStats(ctx context.Context) (*StatsResponse, error)
}

type Handler struct {
	accountSvc AccountService
	eventSvc   EventService
	apiKeySvc  APIKeyService
	statsSvc   StatsService
}

func NewHandler(
	accountSvc AccountService,
	eventSvc EventService,
	apiKeySvc APIKeyService,
	statsSvc StatsService,
) *Handler {
	return &Handler{
		accountSvc: accountSvc,
		eventSvc:   eventSvc,
		apiKeySvc:  apiKeySvc,
		statsSvc:   statsSvc,
	}
}

// setContentRangeHeader sets the Content-Range header for paginated responses.
func setContentRangeHeader(c *gin.Context, resource string, offset, limit int, total int64) {
	if total == 0 {
		c.Header("Content-Range", fmt.Sprintf("%s */0", resource))
		return
	}
	end := offset + limit - 1
	if int64(end) >= total {
		end = int(total) - 1
	}
	c.Header("Content-Range", fmt.Sprintf("%s %d-%d/%d", resource, offset, end, total))
}

// ---------- Accounts ----------

// ListAccounts godoc
// @Summary List all accounts (admin)
// @Tags admin
// @Produce json
// @Param _start query int false "Start index (0-based)"
// @Param _end query int false "End index (exclusive)"
// @Param _sort query string false "Sort field (id, email, role, active, createdAt)"
// @Param _order query string false "Sort order (ASC/DESC)"
// @Param filter query string false "JSON filter: {q, role, active}"
// @Success 200 {array} AccountResponse
// @Header 200 {string} Content-Range "resources start-end/total"
// @Header 200 {integer} X-Total-Count "Total number of items"
// @Security BearerAuth
// @Router /admin/accounts [get]
func (h *Handler) ListAccounts(c *gin.Context) {
	params, err := pagination.ParseReactAdminParams(c)
	if err != nil {
		return
	}

	// Map sort field names to database column names
	sortMap := map[string]string{
		"id":        "id",
		"email":     "email",
		"role":      "role",
		"active":    "active",
		"createdAt": "created_at",
	}
	sort := params.SortField
	if dbSort, ok := sortMap[sort]; ok {
		sort = dbSort
	} else {
		sort = "created_at"
	}
	order := params.SortOrder
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	accounts, total, err := h.accountSvc.ListAccounts(c, params.Offset, params.Limit, sort, order, params.Filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]AccountResponse, len(accounts))
	for i, acc := range accounts {
		resp[i] = ToAccountResponse(acc)
	}
	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	setContentRangeHeader(c, "accounts", params.Offset, params.Limit, total)
	c.JSON(http.StatusOK, resp)
}

// GetAccount godoc
// @Summary Get single account (admin)
// @Tags admin
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} AccountResponse
// @Security BearerAuth
// @Router /admin/accounts/{id} [get]
func (h *Handler) GetAccount(c *gin.Context) {
	id := c.Param("id")
	acc, err := h.accountSvc.GetByID(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	c.JSON(http.StatusOK, ToAccountResponse(acc))
}

// CreateAccount godoc
// @Summary Create new account (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Param body body CreateAccountRequest true "Account data"
// @Success 201 {object} AccountResponse
// @Security BearerAuth
// @Router /admin/accounts [post]
func (h *Handler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	acc := &account.Account{
		Email: req.Email,
		Role:  req.Role,
	}
	err := h.accountSvc.CreateAccount(c, acc, req.Password)
	if err != nil {
		status := http.StatusInternalServerError
		msg := err.Error()
		if strings.Contains(msg, "already exists") {
			status = http.StatusConflict
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusCreated, ToAccountResponse(acc))
}

// UpdateAccount godoc
// @Summary Update account (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "Account ID"
// @Param body body UpdateAccountRequest true "Fields to update"
// @Success 200 {object} AccountResponse
// @Security BearerAuth
// @Router /admin/accounts/{id} [put]
func (h *Handler) UpdateAccount(c *gin.Context) {
	id := c.Param("id")
	var req UpdateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	updates := make(map[string]interface{})
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Active != nil {
		updates["active"] = *req.Active
	}

	acc, err := h.accountSvc.UpdateAccountAdmin(c, id, updates)
	if err != nil {
		status := http.StatusInternalServerError
		msg := err.Error()
		if strings.Contains(msg, "not found") {
			status = http.StatusNotFound
		} else if strings.Contains(msg, "already exists") {
			status = http.StatusConflict
		} else if strings.Contains(msg, "invalid role") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": msg})
		return
	}
	c.JSON(http.StatusOK, ToAccountResponse(acc))
}

// DeleteAccount godoc
// @Summary Delete account (admin)
// @Tags admin
// @Produce json
// @Param id path string true "Account ID"
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /admin/accounts/{id} [delete]
func (h *Handler) DeleteAccount(c *gin.Context) {
	id := c.Param("id")
	err := h.accountSvc.DeleteAccount(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "account not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

// ---------- Events ----------

// ListEvents godoc
// @Summary List all events (admin)
// @Tags admin
// @Produce json
// @Param _start query int false "Start index (0-based)"
// @Param _end query int false "End index (exclusive)"
// @Param _sort query string false "Sort field (id, createdAt, username, name)"
// @Param _order query string false "Sort order (ASC/DESC)"
// @Param filter query string false "JSON filter: {q, account_id, api_key_id}"
// @Success 200 {array} EventResponse
// @Header 200 {string} Content-Range "resources start-end/total"
// @Header 200 {integer} X-Total-Count
// @Security BearerAuth
// @Router /admin/events [get]
func (h *Handler) ListEvents(c *gin.Context) {
	params, err := pagination.ParseReactAdminParams(c)
	if err != nil {
		return
	}

	events, total, err := h.eventSvc.ListAllEvents(c, params.Offset, params.Limit, params.SortField, params.SortOrder, params.Filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]EventResponse, len(events))
	for i, ev := range events {
		resp[i] = ToEventResponse(ev)
	}
	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	setContentRangeHeader(c, "events", params.Offset, params.Limit, total)
	c.JSON(http.StatusOK, resp)
}

// GetEvent godoc
// @Summary Get single event (admin)
// @Tags admin
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} EventResponse
// @Security BearerAuth
// @Router /admin/events/{id} [get]
func (h *Handler) GetEvent(c *gin.Context) {
	id := c.Param("id")
	ev, err := h.eventSvc.GetByID(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "event not found"})
		return
	}
	c.JSON(http.StatusOK, ToEventResponse(ev))
}

// ---------- API Keys ----------

// ListAPIKeys godoc
// @Summary List all API keys (admin)
// @Tags admin
// @Produce json
// @Param _start query int false "Start index (0-based)"
// @Param _end query int false "End index (exclusive)"
// @Param _sort query string false "Sort field (id, createdAt, active)"
// @Param _order query string false "Sort order (ASC/DESC)"
// @Param filter query string false "JSON filter: {account_id, active}"
// @Success 200 {array} APIKeyResponse
// @Header 200 {string} Content-Range "resources start-end/total"
// @Header 200 {integer} X-Total-Count
// @Security BearerAuth
// @Router /admin/api-keys [get]
func (h *Handler) ListAPIKeys(c *gin.Context) {
	params, err := pagination.ParseReactAdminParams(c)
	if err != nil {
		return
	}

	keys, total, err := h.apiKeySvc.ListAllAPIKeys(c, params.Offset, params.Limit, params.SortField, params.SortOrder, params.Filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]APIKeyResponse, len(keys))
	for i, key := range keys {
		resp[i] = ToAPIKeyResponse(key, "")
	}
	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	setContentRangeHeader(c, "api-keys", params.Offset, params.Limit, total)
	c.JSON(http.StatusOK, resp)
}

// UpdateAPIKey godoc
// @Summary Update API key active status (admin)
// @Tags admin
// @Accept json
// @Produce json
// @Param id path string true "API Key ID"
// @Param body body UpdateAPIKeyRequest true "Active status"
// @Success 200 {object} APIKeyResponse
// @Security BearerAuth
// @Router /admin/api-keys/{id} [put]
func (h *Handler) UpdateAPIKey(c *gin.Context) {
	id := c.Param("id")
	var req UpdateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		c.Abort()
		return
	}

	err := h.apiKeySvc.UpdateActive(c, id, *req.Active)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}
	key, _ := h.apiKeySvc.GetByID(c, id)
	c.JSON(http.StatusOK, ToAPIKeyResponse(key, ""))
}

// DeleteAPIKey godoc
// @Summary Delete API key (admin)
// @Tags admin
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200 {object} map[string]string
// @Security BearerAuth
// @Router /admin/api-keys/{id} [delete]
func (h *Handler) DeleteAPIKey(c *gin.Context) {
	id := c.Param("id")
	err := h.apiKeySvc.Delete(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "API key not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id})
}

// ---------- Stats ----------

// GetStats godoc
// @Summary Get system statistics (admin)
// @Tags admin
// @Produce json
// @Success 200 {object} StatsResponse
// @Security BearerAuth
// @Router /admin/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	stats, err := h.statsSvc.GetStats(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}
