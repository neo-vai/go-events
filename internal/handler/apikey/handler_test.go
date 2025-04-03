package apikey

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/middleware"
	"github.com/neo-vai/go-events/internal/model/apikey"
	apikeyService "github.com/neo-vai/go-events/internal/service/apikey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAPIKeyService is a mock for the APIKeyService interface.
type MockAPIKeyService struct {
	mock.Mock
}

func (m *MockAPIKeyService) Generate(ctx context.Context, accountID string) (*apikey.APIKey, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apikey.APIKey), args.Error(1)
}

func (m *MockAPIKeyService) GetByID(ctx context.Context, id string) (*apikey.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apikey.APIKey), args.Error(1)
}

func (m *MockAPIKeyService) ListByAccount(ctx context.Context, accountID string) ([]*apikey.APIKey, error) {
	args := m.Called(ctx, accountID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*apikey.APIKey), args.Error(1)
}

func (m *MockAPIKeyService) UpdateActive(ctx context.Context, id string, active bool) error {
	args := m.Called(ctx, id, active)
	return args.Error(0)
}

func (m *MockAPIKeyService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func setupAPIKeyHandlerTest() (*gin.Engine, *MockAPIKeyService) {
	gin.SetMode(gin.TestMode)
	svc := new(MockAPIKeyService)
	h := NewHandler(svc)

	r := gin.New()
	protected := r.Group("/api/v1")
	protected.Use(func(c *gin.Context) {
		// Simulate authentication
		accountID := c.GetHeader("X-Test-Account-ID")
		if accountID != "" {
			c.Set(string(middleware.AccountIDKey), accountID)
		}
		c.Next()
	})
	{
		protected.POST("/account/keys", h.GenerateAPIKeyForCurrentAccount)
		protected.GET("/account/keys", h.ListAPIKeysForCurrentAccount)
		protected.PATCH("/account/keys/:key_id", h.UpdateAPIKeyActiveForCurrentAccount)
		protected.DELETE("/account/keys/:key_id", h.DeleteAPIKeyForCurrentAccount)
	}
	return r, svc
}

func TestHandler_GenerateAPIKeyForCurrentAccount_Success(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()
	accountID := uuid.New().String()

	expectedKey := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: uuid.MustParse(accountID),
		PlainKey:  "generated-plain-key",
		Active:    true,
	}
	svc.On("Generate", mock.Anything, accountID).Return(expectedKey, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/account/keys", nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp APIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, expectedKey.ID.String(), resp.ID)
	assert.Equal(t, expectedKey.PlainKey, resp.Key)
	assert.True(t, resp.Active)
	svc.AssertExpectations(t)
}

func TestHandler_GenerateAPIKeyForCurrentAccount_Unauthorized(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/account/keys", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "Generate")
}

func TestHandler_GenerateAPIKeyForCurrentAccount_InvalidAccount(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()
	accountID := uuid.New().String()

	svc.On("Generate", mock.Anything, accountID).Return(nil, apikeyService.ErrInvalidAccountID).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/account/keys", nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid account ID", errResp["error"])
	svc.AssertExpectations(t)
}

func TestHandler_GenerateAPIKeyForCurrentAccount_AccountNotFound(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()
	accountID := uuid.New().String()

	svc.On("Generate", mock.Anything, accountID).Return(nil, apikeyService.ErrAccountNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/account/keys", nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "account not found", errResp["error"])
	svc.AssertExpectations(t)
}

func TestHandler_ListAPIKeysForCurrentAccount_Success(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()
	accountID := uuid.New().String()

	keys := []*apikey.APIKey{
		{ID: uuid.New(), AccountID: uuid.MustParse(accountID), PlainKey: "key1", Active: true},
		{ID: uuid.New(), AccountID: uuid.MustParse(accountID), PlainKey: "key2", Active: false},
	}
	svc.On("ListByAccount", mock.Anything, accountID).Return(keys, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/account/keys", nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []APIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
	assert.Equal(t, keys[0].ID.String(), resp[0].ID)
	// Full key should not be included in list
	assert.Empty(t, resp[0].Key)
	assert.NotEmpty(t, resp[0].KeyPrefix)
	svc.AssertExpectations(t)
}

func TestHandler_ListAPIKeysForCurrentAccount_Unauthorized(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/account/keys", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "ListByAccount")
}

func TestHandler_UpdateAPIKeyActiveForCurrentAccount_Success(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()
	accountID := uuid.New().String()
	keyID := uuid.New().String()

	key := &apikey.APIKey{
		ID:        uuid.MustParse(keyID),
		AccountID: uuid.MustParse(accountID),
		PlainKey:  "somekey",
		Active:    true,
	}
	svc.On("GetByID", mock.Anything, keyID).Return(key, nil).Once()
	svc.On("UpdateActive", mock.Anything, keyID, false).Return(nil).Once()
	// After update, GetByID called again to return updated key
	updatedKey := &apikey.APIKey{
		ID:        uuid.MustParse(keyID),
		AccountID: uuid.MustParse(accountID),
		PlainKey:  "somekey",
		Active:    false,
	}
	svc.On("GetByID", mock.Anything, keyID).Return(updatedKey, nil).Once()

	reqBody := UpdateAPIKeyActiveRequest{Active: boolPtr(false)}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/account/keys/"+keyID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp APIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, keyID, resp.ID)
	assert.False(t, resp.Active)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateAPIKeyActiveForCurrentAccount_KeyNotOwned(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()
	accountID := uuid.New().String()
	otherAccountID := uuid.New().String()
	keyID := uuid.New().String()

	key := &apikey.APIKey{
		ID:        uuid.MustParse(keyID),
		AccountID: uuid.MustParse(otherAccountID),
	}
	svc.On("GetByID", mock.Anything, keyID).Return(key, nil).Once()

	reqBody := UpdateAPIKeyActiveRequest{Active: boolPtr(true)}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/account/keys/"+keyID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "API key does not belong to this account", errResp["error"])
	svc.AssertNotCalled(t, "UpdateActive")
}

func TestHandler_DeleteAPIKeyForCurrentAccount_Success(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()
	accountID := uuid.New().String()
	keyID := uuid.New().String()

	key := &apikey.APIKey{
		ID:        uuid.MustParse(keyID),
		AccountID: uuid.MustParse(accountID),
	}
	svc.On("GetByID", mock.Anything, keyID).Return(key, nil).Once()
	svc.On("Delete", mock.Anything, keyID).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/account/keys/"+keyID, nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_DeleteAPIKeyForCurrentAccount_KeyNotOwned(t *testing.T) {
	r, svc := setupAPIKeyHandlerTest()
	accountID := uuid.New().String()
	otherAccountID := uuid.New().String()
	keyID := uuid.New().String()

	key := &apikey.APIKey{
		ID:        uuid.MustParse(keyID),
		AccountID: uuid.MustParse(otherAccountID),
	}
	svc.On("GetByID", mock.Anything, keyID).Return(key, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/account/keys/"+keyID, nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "API key does not belong to this account", errResp["error"])
	svc.AssertNotCalled(t, "Delete")
}

func boolPtr(b bool) *bool {
	return &b
}
