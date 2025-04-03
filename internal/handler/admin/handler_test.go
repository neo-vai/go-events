package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/middleware"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/neo-vai/go-events/internal/model/event"
	"github.com/neo-vai/go-events/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAccountService struct {
	mock.Mock
}

func (m *MockAccountService) CreateAccount(ctx context.Context, acc *account.Account, password string) error {
	args := m.Called(ctx, acc, password)
	return args.Error(0)
}

func (m *MockAccountService) GetByID(ctx context.Context, id string) (*account.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}

func (m *MockAccountService) UpdateAccountAdmin(ctx context.Context, id string, updates map[string]interface{}) (*account.Account, error) {
	args := m.Called(ctx, id, updates)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}

func (m *MockAccountService) DeleteAccount(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAccountService) ListAccounts(ctx context.Context, offset, limit int, sort, order string, filters map[string]interface{}) ([]*account.Account, int64, error) {
	args := m.Called(ctx, offset, limit, sort, order, filters)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*account.Account), args.Get(1).(int64), args.Error(2)
}

type MockEventService struct {
	mock.Mock
}

func (m *MockEventService) GetByID(ctx context.Context, id string) (*event.Event, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*event.Event), args.Error(1)
}

func (m *MockEventService) ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error) {
	args := m.Called(ctx, accountID, username, apiKeyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*event.Event), args.Error(1)
}

func (m *MockEventService) ListAllEvents(ctx context.Context, offset, limit int, sort, order string, filters map[string]interface{}) ([]*event.Event, int64, error) {
	args := m.Called(ctx, offset, limit, sort, order, filters)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*event.Event), args.Get(1).(int64), args.Error(2)
}

type MockAPIKeyService struct {
	mock.Mock
}

func (m *MockAPIKeyService) GetByID(ctx context.Context, id string) (*apikey.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apikey.APIKey), args.Error(1)
}

func (m *MockAPIKeyService) UpdateActive(ctx context.Context, id string, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func (m *MockAPIKeyService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAPIKeyService) ListAllAPIKeys(ctx context.Context, offset, limit int, sort, order string, filters map[string]interface{}) ([]*apikey.APIKey, int64, error) {
	args := m.Called(ctx, offset, limit, sort, order, filters)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*apikey.APIKey), args.Get(1).(int64), args.Error(2)
}

type MockStatsService struct {
	mock.Mock
}

func (m *MockStatsService) GetStats(ctx context.Context) (*StatsResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*StatsResponse), args.Error(1)
}

func setupAdminHandlerTest() (*gin.Engine, *MockAccountService, *MockEventService, *MockAPIKeyService, *MockStatsService) {
	gin.SetMode(gin.TestMode)
	validator.RegisterCustomValidators()
	accountSvc := new(MockAccountService)
	eventSvc := new(MockEventService)
	apiKeySvc := new(MockAPIKeyService)
	statsSvc := new(MockStatsService)
	h := NewHandler(accountSvc, eventSvc, apiKeySvc, statsSvc)

	r := gin.New()
	r.Use(middleware.ValidationErrorHandler())
	adminGroup := r.Group("/api/v1/admin")
	adminGroup.Use(func(c *gin.Context) {
		c.Set(string(middleware.AccountIDKey), "admin-id")
		c.Set(string(middleware.RoleKey), "admin")
		c.Next()
	})
	{
		adminGroup.GET("/accounts", h.ListAccounts)
		adminGroup.GET("/accounts/:id", h.GetAccount)
		adminGroup.POST("/accounts", h.CreateAccount)
		adminGroup.PUT("/accounts/:id", h.UpdateAccount)
		adminGroup.DELETE("/accounts/:id", h.DeleteAccount)

		adminGroup.GET("/events", h.ListEvents)
		adminGroup.GET("/events/:id", h.GetEvent)

		adminGroup.GET("/api-keys", h.ListAPIKeys)
		adminGroup.PUT("/api-keys/:id", h.UpdateAPIKey)
		adminGroup.DELETE("/api-keys/:id", h.DeleteAPIKey)

		adminGroup.GET("/stats", h.GetStats)
	}
	return r, accountSvc, eventSvc, apiKeySvc, statsSvc
}

func TestAdminHandler_ListAccounts_Success(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()

	acc1 := &account.Account{ID: uuid.New(), Email: "user1@example.com", Role: "user", Active: true, CreatedAt: time.Now()}
	acc2 := &account.Account{ID: uuid.New(), Email: "user2@example.com", Role: "admin", Active: true, CreatedAt: time.Now()}

	accountSvc.On("ListAccounts", mock.Anything, 0, 20, "id", "ASC", mock.Anything).
		Return([]*account.Account{acc1, acc2}, int64(2), nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/accounts", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2", w.Header().Get("X-Total-Count"))

	var resp []AccountResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
	assert.Equal(t, "user1@example.com", resp[0].Email)
	assert.Equal(t, "user2@example.com", resp[1].Email)

	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_ListAccounts_WithPagination(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()

	accountSvc.On("ListAccounts", mock.Anything, 10, 10, "email", "ASC", mock.Anything).
		Return([]*account.Account{}, int64(0), nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/accounts?_start=10&_end=20&_sort=email&_order=ASC", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_GetAccount_Success(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()
	accountID := uuid.New().String()

	acc := &account.Account{ID: uuid.MustParse(accountID), Email: "user@example.com", Role: "user", Active: true, CreatedAt: time.Now()}
	accountSvc.On("GetByID", mock.Anything, accountID).Return(acc, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/accounts/"+accountID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp AccountResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, accountID, resp.ID)
	assert.Equal(t, "user@example.com", resp.Email)

	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_GetAccount_NotFound(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()
	accountID := uuid.New().String()

	accountSvc.On("GetByID", mock.Anything, accountID).Return(nil, errors.New("not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/accounts/"+accountID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_CreateAccount_Success(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()

	reqBody := CreateAccountRequest{
		Email:    "newadmin@example.com",
		Password: "StrongPass123",
		Role:     "admin",
	}
	jsonBody, _ := json.Marshal(reqBody)

	createdAcc := &account.Account{
		ID:     uuid.New(),
		Email:  "newadmin@example.com",
		Role:   "admin",
		Active: true,
	}

	accountSvc.On("CreateAccount", mock.Anything, mock.MatchedBy(func(acc *account.Account) bool {
		return acc.Email == "newadmin@example.com" && acc.Role == "admin"
	}), "StrongPass123").Return(nil).Once()
	accountSvc.On("GetByID", mock.Anything, mock.Anything).Return(createdAcc, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/admin/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp AccountResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "newadmin@example.com", resp.Email)
	assert.Equal(t, "admin", resp.Role)

	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_CreateAccount_InvalidJSON(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/admin/accounts", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	accountSvc.AssertNotCalled(t, "CreateAccount")
}

func TestAdminHandler_CreateAccount_Conflict(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()

	reqBody := CreateAccountRequest{
		Email:    "duplicate@example.com",
		Password: "StrongPass123",
		Role:     "user",
	}
	jsonBody, _ := json.Marshal(reqBody)

	accountSvc.On("CreateAccount", mock.Anything, mock.Anything, "StrongPass123").
		Return(errors.New("email already exists")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/admin/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_UpdateAccount_Success(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()
	accountID := uuid.New().String()

	reqBody := UpdateAccountRequest{
		Email:  "updated@example.com",
		Role:   "admin",
		Active: boolPtr(false),
	}
	jsonBody, _ := json.Marshal(reqBody)

	updatedAcc := &account.Account{
		ID:     uuid.MustParse(accountID),
		Email:  "updated@example.com",
		Role:   "admin",
		Active: false,
	}
	accountSvc.On("UpdateAccountAdmin", mock.Anything, accountID, mock.MatchedBy(func(updates map[string]interface{}) bool {
		return updates["email"] == "updated@example.com" &&
			updates["role"] == "admin" &&
			updates["active"] == false
	})).Return(updatedAcc, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/admin/accounts/"+accountID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp AccountResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, accountID, resp.ID)
	assert.Equal(t, "updated@example.com", resp.Email)
	assert.Equal(t, "admin", resp.Role)
	assert.False(t, resp.Active)

	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_UpdateAccount_NotFound(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()
	accountID := uuid.New().String()

	reqBody := UpdateAccountRequest{Email: "updated@example.com"}
	jsonBody, _ := json.Marshal(reqBody)

	accountSvc.On("UpdateAccountAdmin", mock.Anything, accountID, mock.Anything).
		Return(nil, errors.New("account not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/admin/accounts/"+accountID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_DeleteAccount_Success(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()
	accountID := uuid.New().String()

	accountSvc.On("DeleteAccount", mock.Anything, accountID).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/admin/accounts/"+accountID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, accountID, resp["id"])

	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_DeleteAccount_NotFound(t *testing.T) {
	r, accountSvc, _, _, _ := setupAdminHandlerTest()
	accountID := uuid.New().String()

	accountSvc.On("DeleteAccount", mock.Anything, accountID).Return(errors.New("not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/admin/accounts/"+accountID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	accountSvc.AssertExpectations(t)
}

func TestAdminHandler_ListEvents_Success(t *testing.T) {
	r, _, eventSvc, _, _ := setupAdminHandlerTest()

	ev1 := &event.Event{ID: "ev1", AccountID: uuid.New().String(), Username: "user1", Name: "login", Payload: `{}`, CreatedAt: time.Now()}
	ev2 := &event.Event{ID: "ev2", AccountID: uuid.New().String(), Username: "user2", Name: "logout", Payload: `{}`, CreatedAt: time.Now()}

	eventSvc.On("ListAllEvents", mock.Anything, 0, 20, "id", "ASC", mock.Anything).
		Return([]*event.Event{ev1, ev2}, int64(2), nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/events", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2", w.Header().Get("X-Total-Count"))

	var resp []EventResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)

	eventSvc.AssertExpectations(t)
}

func TestAdminHandler_GetEvent_Success(t *testing.T) {
	r, _, eventSvc, _, _ := setupAdminHandlerTest()
	eventID := uuid.New().String()

	ev := &event.Event{ID: eventID, AccountID: uuid.New().String(), Username: "user1", Name: "login", Payload: `{"ip":"1.2.3.4"}`, CreatedAt: time.Now()}
	eventSvc.On("GetByID", mock.Anything, eventID).Return(ev, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/events/"+eventID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp EventResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, eventID, resp.ID)
	assert.Equal(t, "user1", resp.Username)

	eventSvc.AssertExpectations(t)
}

func TestAdminHandler_GetEvent_NotFound(t *testing.T) {
	r, _, eventSvc, _, _ := setupAdminHandlerTest()
	eventID := uuid.New().String()

	eventSvc.On("GetByID", mock.Anything, eventID).Return(nil, errors.New("not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/events/"+eventID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	eventSvc.AssertExpectations(t)
}

func TestAdminHandler_ListAPIKeys_Success(t *testing.T) {
	r, _, _, apiKeySvc, _ := setupAdminHandlerTest()

	key1 := &apikey.APIKey{ID: uuid.New(), AccountID: uuid.New(), KeyHash: "hash1", Active: true, CreatedAt: time.Now()}
	key2 := &apikey.APIKey{ID: uuid.New(), AccountID: uuid.New(), KeyHash: "hash2", Active: false, CreatedAt: time.Now()}

	apiKeySvc.On("ListAllAPIKeys", mock.Anything, 0, 20, "id", "ASC", mock.Anything).
		Return([]*apikey.APIKey{key1, key2}, int64(2), nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/api-keys", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "2", w.Header().Get("X-Total-Count"))

	var resp []APIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)

	apiKeySvc.AssertExpectations(t)
}

func TestAdminHandler_UpdateAPIKey_Success(t *testing.T) {
	r, _, _, apiKeySvc, _ := setupAdminHandlerTest()
	keyID := uuid.New().String()

	reqBody := UpdateAPIKeyRequest{Active: boolPtr(false)}
	jsonBody, _ := json.Marshal(reqBody)

	apiKeySvc.On("UpdateActive", mock.Anything, keyID, false).Return(nil).Once()
	apiKeySvc.On("GetByID", mock.Anything, keyID).Return(&apikey.APIKey{
		ID:        uuid.MustParse(keyID),
		AccountID: uuid.New(),
		KeyHash:   "hash1",
		Active:    false,
		CreatedAt: time.Now(),
	}, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/admin/api-keys/"+keyID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp APIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, keyID, resp.ID)
	assert.False(t, resp.Active)

	apiKeySvc.AssertExpectations(t)
}

func TestAdminHandler_UpdateAPIKey_NotFound(t *testing.T) {
	r, _, _, apiKeySvc, _ := setupAdminHandlerTest()
	keyID := uuid.New().String()

	reqBody := UpdateAPIKeyRequest{Active: boolPtr(true)}
	jsonBody, _ := json.Marshal(reqBody)

	apiKeySvc.On("UpdateActive", mock.Anything, keyID, true).Return(errors.New("not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/admin/api-keys/"+keyID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	apiKeySvc.AssertExpectations(t)
}

func TestAdminHandler_DeleteAPIKey_Success(t *testing.T) {
	r, _, _, apiKeySvc, _ := setupAdminHandlerTest()
	keyID := uuid.New().String()

	apiKeySvc.On("Delete", mock.Anything, keyID).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/admin/api-keys/"+keyID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, keyID, resp["id"])

	apiKeySvc.AssertExpectations(t)
}

func TestAdminHandler_DeleteAPIKey_NotFound(t *testing.T) {
	r, _, _, apiKeySvc, _ := setupAdminHandlerTest()
	keyID := uuid.New().String()

	apiKeySvc.On("Delete", mock.Anything, keyID).Return(errors.New("not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/admin/api-keys/"+keyID, nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	apiKeySvc.AssertExpectations(t)
}

func TestAdminHandler_GetStats_Success(t *testing.T) {
	r, _, _, _, statsSvc := setupAdminHandlerTest()

	stats := &StatsResponse{
		TotalAccounts:  42,
		ActiveAccounts: 38,
		TotalEvents:    15243,
		EventsToday:    512,
		TotalAPIKeys:   75,
		ActiveAPIKeys:  60,
	}
	statsSvc.On("GetStats", mock.Anything).Return(stats, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp StatsResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, int64(42), resp.TotalAccounts)
	assert.Equal(t, int64(38), resp.ActiveAccounts)
	assert.Equal(t, int64(15243), resp.TotalEvents)
	assert.Equal(t, int64(512), resp.EventsToday)
	assert.Equal(t, int64(75), resp.TotalAPIKeys)
	assert.Equal(t, int64(60), resp.ActiveAPIKeys)

	statsSvc.AssertExpectations(t)
}

func TestAdminHandler_GetStats_Error(t *testing.T) {
	r, _, _, _, statsSvc := setupAdminHandlerTest()

	statsSvc.On("GetStats", mock.Anything).Return(nil, errors.New("database error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/admin/stats", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	statsSvc.AssertExpectations(t)
}

func boolPtr(b bool) *bool {
	return &b
}
