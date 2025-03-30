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
)

type AccountService interface {
	CreateAccount(ctx context.Context, acc *account.Account, password string) error
	GetByID(ctx context.Context, id string) (*account.Account, error)
	UpdateAccountAdmin(ctx context.Context, id string, updates map[string]interface{}) (*account.Account, error)
	DeleteAccount(ctx context.Context, id string) error
	ListAccounts(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*account.Account, int64, error)
}

type EventService interface {
	GetByID(ctx context.Context, id string) (*event.Event, error)
	ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error)
	ListAllEvents(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*event.Event, int64, error)
}

type APIKeyService interface {
	GetByID(ctx context.Context, id string) (*apikey.APIKey, error)
	UpdateActive(ctx context.Context, id string, active bool) error
	Delete(ctx context.Context, id string) error
	ListAllAPIKeys(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*apikey.APIKey, int64, error)
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

// ---------- Accounts ----------

// ListAccounts godoc
// @Summary List all accounts (admin)
// @Tags admin
// @Produce json
// @Param _page query int false "Page number"
// @Param _limit query int false "Items per page"
// @Param _sort query string false "Sort field"
// @Param _order query string false "Sort order (ASC/DESC)"
// @Param _q query string false "Search query"
// @Param role query string false "Filter by role"
// @Param active query bool false "Filter by active status"
// @Success 200 {array} AccountResponse
// @Header 200 {string} Content-Range "resources start-end/total"
// @Header 200 {integer} X-Total-Count "Total number of items"
// @Security BearerAuth
// @Router /admin/accounts [get]
func (h *Handler) ListAccounts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("_page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("_limit", "10"))
	sort := c.DefaultQuery("_sort", "createdAt")
	order := c.DefaultQuery("_order", "DESC")

	sortMap := map[string]string{
		"id":        "id",
		"name":      "name",
		"email":     "email",
		"login":     "login",
		"role":      "role",
		"active":    "active",
		"createdAt": "created_at",
	}
	if dbSort, ok := sortMap[sort]; ok {
		sort = dbSort
	} else {
		sort = "created_at"
	}
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	filters := make(map[string]interface{})
	if q := c.Query("_q"); q != "" {
		filters["q"] = q
	}
	if role := c.Query("role"); role != "" {
		filters["role"] = role
	}
	if activeStr := c.Query("active"); activeStr != "" {
		active, _ := strconv.ParseBool(activeStr)
		filters["active"] = active
	}

	accounts, total, err := h.accountSvc.ListAccounts(c, page, limit, sort, order, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]AccountResponse, len(accounts))
	for i, acc := range accounts {
		resp[i] = ToAccountResponse(acc)
	}
	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	setContentRangeHeader(c, "accounts", page, limit, total)
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
		Name:  req.Name,
		Email: req.Email,
		Login: req.Login,
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
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Login != "" {
		updates["login"] = req.Login
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
// @Param _page query int false "Page number"
// @Param _limit query int false "Items per page"
// @Param _sort query string false "Sort field"
// @Param _order query string false "Sort order (ASC/DESC)"
// @Param _q query string false "Search query"
// @Param account_id query string false "Filter by account ID"
// @Param api_key_id query string false "Filter by API key ID"
// @Success 200 {array} EventResponse
// @Header 200 {string} Content-Range "resources start-end/total"
// @Header 200 {integer} X-Total-Count
// @Security BearerAuth
// @Router /admin/events [get]
func (h *Handler) ListEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("_page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("_limit", "10"))
	sort := c.DefaultQuery("_sort", "createdAt")
	order := c.DefaultQuery("_order", "DESC")

	filters := make(map[string]interface{})
	if q := c.Query("_q"); q != "" {
		filters["q"] = q
	}
	if accountID := c.Query("account_id"); accountID != "" {
		filters["account_id"] = accountID
	}
	if apiKeyID := c.Query("api_key_id"); apiKeyID != "" {
		filters["api_key_id"] = apiKeyID
	}

	events, total, err := h.eventSvc.ListAllEvents(c, page, limit, sort, order, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]EventResponse, len(events))
	for i, ev := range events {
		resp[i] = ToEventResponse(ev, "") // accountName omitted for simplicity
	}
	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	setContentRangeHeader(c, "events", page, limit, total)
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
	c.JSON(http.StatusOK, ToEventResponse(ev, ""))
}

// ---------- API Keys ----------

// ListAPIKeys godoc
// @Summary List all API keys (admin)
// @Tags admin
// @Produce json
// @Param _page query int false "Page number"
// @Param _limit query int false "Items per page"
// @Param _sort query string false "Sort field"
// @Param _order query string false "Sort order (ASC/DESC)"
// @Param _q query string false "Search query"
// @Param account_id query string false "Filter by account ID"
// @Param active query bool false "Filter by active status"
// @Success 200 {array} APIKeyResponse
// @Header 200 {string} Content-Range "resources start-end/total"
// @Header 200 {integer} X-Total-Count
// @Security BearerAuth
// @Router /admin/api-keys [get]
func (h *Handler) ListAPIKeys(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("_page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("_limit", "10"))
	sort := c.DefaultQuery("_sort", "createdAt")
	order := c.DefaultQuery("_order", "DESC")

	filters := make(map[string]interface{})
	if q := c.Query("_q"); q != "" {
		filters["q"] = q
	}
	if accountID := c.Query("account_id"); accountID != "" {
		filters["account_id"] = accountID
	}
	if activeStr := c.Query("active"); activeStr != "" {
		active, _ := strconv.ParseBool(activeStr)
		filters["active"] = active
	}

	keys, total, err := h.apiKeySvc.ListAllAPIKeys(c, page, limit, sort, order, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]APIKeyResponse, len(keys))
	for i, key := range keys {
		resp[i] = ToAPIKeyResponse(key, "")
	}
	c.Header("X-Total-Count", strconv.FormatInt(total, 10))
	setContentRangeHeader(c, "api-keys", page, limit, total)
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
