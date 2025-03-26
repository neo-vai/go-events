package apikey

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/apikey"
	apikey_service "github.com/neo-vai/go-events/internal/service/apikey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAPIKeyService is a mock for APIKeyService.
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

func setupAPIKeyRouter(service APIKeyService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(service)
	r.POST("/api/v1/accounts/:id/keys", h.GenerateAPIKey)
	r.GET("/api/v1/accounts/:id/keys", h.ListAPIKeys)
	r.PATCH("/api/v1/accounts/:id/keys/:key_id", h.UpdateAPIKeyActive)
	r.DELETE("/api/v1/accounts/:id/keys/:key_id", h.DeleteAPIKey)
	return r
}

// ---------- GenerateAPIKey ----------
func TestGenerateAPIKey_Success(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	accountID := uuid.New().String()
	expectedKey := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: uuid.MustParse(accountID),
		Key:       "generated-key-123",
		Active:    true,
	}

	svc.On("Generate", mock.Anything, accountID).Return(expectedKey, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts/"+accountID+"/keys", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp APIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, expectedKey.ID.String(), resp.ID)
	assert.Equal(t, expectedKey.Key, resp.Key)
	assert.True(t, resp.Active)
	svc.AssertExpectations(t)
}

func TestGenerateAPIKey_InvalidAccountID(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	svc.On("Generate", mock.Anything, "invalid").Return(nil, apikey_service.ErrInvalidAccountID).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts/invalid/keys", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid account ID", errResp["error"])
	svc.AssertExpectations(t)
}

func TestGenerateAPIKey_AccountNotFound(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	accountID := uuid.New().String()
	svc.On("Generate", mock.Anything, accountID).Return(nil, apikey_service.ErrAccountNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts/"+accountID+"/keys", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "account not found", errResp["error"])
	svc.AssertExpectations(t)
}

func TestGenerateAPIKey_KeyAlreadyExists(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	accountID := uuid.New().String()
	svc.On("Generate", mock.Anything, accountID).Return(nil, apikey_service.ErrKeyAlreadyExists).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts/"+accountID+"/keys", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "API key already exists", errResp["error"])
	svc.AssertExpectations(t)
}

func TestGenerateAPIKey_ServiceError(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	accountID := uuid.New().String()
	svc.On("Generate", mock.Anything, accountID).Return(nil, errors.New("db error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts/"+accountID+"/keys", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}

// ---------- ListAPIKeys ----------
func TestListAPIKeys_Success(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	accountID := uuid.New().String()
	keys := []*apikey.APIKey{
		{ID: uuid.New(), AccountID: uuid.MustParse(accountID), Key: "key1", Active: true},
		{ID: uuid.New(), AccountID: uuid.MustParse(accountID), Key: "key2", Active: false},
	}

	svc.On("ListByAccount", mock.Anything, accountID).Return(keys, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/accounts/"+accountID+"/keys", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []APIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp, 2)
	assert.Equal(t, "key1", resp[0].Key)
	assert.False(t, resp[1].Active)
	svc.AssertExpectations(t)
}

func TestListAPIKeys_Empty(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	accountID := uuid.New().String()
	svc.On("ListByAccount", mock.Anything, accountID).Return([]*apikey.APIKey{}, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/accounts/"+accountID+"/keys", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp []APIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Empty(t, resp)
	svc.AssertExpectations(t)
}

func TestListAPIKeys_InvalidAccountID(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	svc.On("ListByAccount", mock.Anything, "invalid").Return(nil, apikey_service.ErrInvalidAccountID).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/accounts/invalid/keys", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid account ID", errResp["error"])
	svc.AssertExpectations(t)
}

func TestListAPIKeys_ServiceError(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	accountID := uuid.New().String()
	svc.On("ListByAccount", mock.Anything, accountID).Return(nil, errors.New("db error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/accounts/"+accountID+"/keys", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}

// ---------- UpdateAPIKeyActive ----------
func TestUpdateAPIKeyActive_Success(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	accountID := uuid.New().String()
	keyID := uuid.New().String()
	activeVal := false
	reqBody := UpdateAPIKeyActiveRequest{Active: &activeVal}
	jsonBody, _ := json.Marshal(reqBody)

	updatedKey := &apikey.APIKey{
		ID:        uuid.MustParse(keyID),
		AccountID: uuid.MustParse(accountID),
		Key:       "some-key",
		Active:    false,
	}

	svc.On("UpdateActive", mock.Anything, keyID, false).Return(nil).Once()
	svc.On("GetByID", mock.Anything, keyID).Return(updatedKey, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/accounts/"+accountID+"/keys/"+keyID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp APIKeyResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, keyID, resp.ID)
	assert.False(t, resp.Active)
	svc.AssertExpectations(t)
}

func TestUpdateAPIKeyActive_InvalidJSON(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/accounts/acc/keys/key", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "UpdateActive", mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateAPIKeyActive_InvalidKeyID(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	activeVal := true
	reqBody := UpdateAPIKeyActiveRequest{Active: &activeVal}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("UpdateActive", mock.Anything, "invalid", true).Return(apikey_service.ErrInvalidKeyID).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/accounts/acc/keys/invalid", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid key ID", errResp["error"])
	svc.AssertExpectations(t)
}

func TestUpdateAPIKeyActive_KeyNotFound(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	keyID := uuid.New().String()
	activeVal := true
	reqBody := UpdateAPIKeyActiveRequest{Active: &activeVal}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("UpdateActive", mock.Anything, keyID, true).Return(apikey_service.ErrKeyNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/accounts/acc/keys/"+keyID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "API key not found", errResp["error"])
	svc.AssertExpectations(t)
}

func TestUpdateAPIKeyActive_UpdateError(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	keyID := uuid.New().String()
	activeVal := true
	reqBody := UpdateAPIKeyActiveRequest{Active: &activeVal}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("UpdateActive", mock.Anything, keyID, true).Return(errors.New("db error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/accounts/acc/keys/"+keyID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}

func TestUpdateAPIKeyActive_GetByIDError(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	keyID := uuid.New().String()
	activeVal := true
	reqBody := UpdateAPIKeyActiveRequest{Active: &activeVal}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("UpdateActive", mock.Anything, keyID, true).Return(nil).Once()
	svc.On("GetByID", mock.Anything, keyID).Return(nil, apikey_service.ErrKeyNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/accounts/acc/keys/"+keyID, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "API key not found", errResp["error"])
	svc.AssertExpectations(t)
}

// ---------- DeleteAPIKey ----------
func TestDeleteAPIKey_Success(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	accountID := uuid.New().String()
	keyID := uuid.New().String()

	svc.On("Delete", mock.Anything, keyID).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/accounts/"+accountID+"/keys/"+keyID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	svc.AssertExpectations(t)
}

func TestDeleteAPIKey_InvalidKeyID(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	svc.On("Delete", mock.Anything, "invalid").Return(apikey_service.ErrInvalidKeyID).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/accounts/acc/keys/invalid", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid key ID", errResp["error"])
	svc.AssertExpectations(t)
}

func TestDeleteAPIKey_KeyNotFound(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	keyID := uuid.New().String()
	svc.On("Delete", mock.Anything, keyID).Return(apikey_service.ErrKeyNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/accounts/acc/keys/"+keyID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "API key not found", errResp["error"])
	svc.AssertExpectations(t)
}

func TestDeleteAPIKey_ServiceError(t *testing.T) {
	svc := new(MockAPIKeyService)
	router := setupAPIKeyRouter(svc)

	keyID := uuid.New().String()
	svc.On("Delete", mock.Anything, keyID).Return(errors.New("db error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/accounts/acc/keys/"+keyID, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}
