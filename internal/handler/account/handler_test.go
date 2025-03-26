// file: internal/handler/account/handler_test.go

package account

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
	"github.com/neo-vai/go-events/internal/model/account"
	account_service "github.com/neo-vai/go-events/internal/service/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
func (m *MockAccountService) UpdateAccount(ctx context.Context, acc *account.Account) error {
	args := m.Called(ctx, acc)
	return args.Error(0)
}
func (m *MockAccountService) DeleteAccount(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func setupRouter(service AccountService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewHandler(service)
	r.POST("/api/v1/accounts", h.CreateAccount)
	r.GET("/api/v1/accounts/:id", h.GetAccount)
	r.PUT("/api/v1/accounts/:id", h.UpdateAccount)
	r.DELETE("/api/v1/accounts/:id", h.DeleteAccount)
	return r
}

// ---------- CreateAccount ----------
func TestCreateAccount_Success(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	reqBody := CreateAccountRequest{
		Name:     "John Doe",
		Email:    "john@example.com",
		Login:    "johndoe",
		Password: "secure123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("CreateAccount", mock.Anything, mock.AnythingOfType("*account.Account"), "secure123").
		Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp AccountResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, reqBody.Name, resp.Name)
	assert.Equal(t, reqBody.Email, resp.Email)
	assert.Equal(t, reqBody.Login, resp.Login)
	assert.NotEmpty(t, resp.ID)
	svc.AssertExpectations(t)
}

func TestCreateAccount_EmailConflict(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	reqBody := CreateAccountRequest{
		Name:     "John Doe",
		Email:    "duplicate@example.com",
		Login:    "unique_login",
		Password: "secure123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("CreateAccount", mock.Anything, mock.AnythingOfType("*account.Account"), "secure123").
		Return(account_service.ErrEmailAlreadyExists).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "email already exists", errResp["error"])
	svc.AssertExpectations(t)
}

func TestCreateAccount_LoginConflict(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	reqBody := CreateAccountRequest{
		Name:     "John Doe",
		Email:    "unique@example.com",
		Login:    "duplicate_login",
		Password: "secure123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("CreateAccount", mock.Anything, mock.AnythingOfType("*account.Account"), "secure123").
		Return(account_service.ErrLoginAlreadyExists).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "login already exists", errResp["error"])
	svc.AssertExpectations(t)
}

func TestCreateAccount_InvalidJSON(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "CreateAccount", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateAccount_MissingRequiredFields(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	reqBody := CreateAccountRequest{
		Name:  "John",
		Email: "john@example.com",
		// missing Login and Password
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "CreateAccount", mock.Anything, mock.Anything, mock.Anything)
}

func TestCreateAccount_ServiceError(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	reqBody := CreateAccountRequest{
		Name:     "John",
		Email:    "john@example.com",
		Login:    "johndoe",
		Password: "pass",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("CreateAccount", mock.Anything, mock.Anything, "pass").
		Return(errors.New("db connection error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}

// ---------- GetAccount ----------
func TestGetAccount_Success(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	id := uuid.New()
	acc := &account.Account{
		ID:    id,
		Name:  "John",
		Email: "john@example.com",
		Login: "johndoe",
	}

	svc.On("GetByID", mock.Anything, id.String()).Return(acc, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/accounts/"+id.String(), nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AccountResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, id.String(), resp.ID)
	assert.Equal(t, acc.Name, resp.Name)
	svc.AssertExpectations(t)
}

func TestGetAccount_NotFound(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	id := uuid.New().String()
	svc.On("GetByID", mock.Anything, id).Return(nil, account_service.ErrAccountNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/accounts/"+id, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

// ---------- UpdateAccount ----------
func TestUpdateAccount_Success(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	id := uuid.New()
	existing := &account.Account{
		ID:    id,
		Name:  "Old Name",
		Email: "old@example.com",
		Login: "oldlogin",
	}

	updateReq := UpdateAccountRequest{
		Name:  "New Name",
		Email: "new@example.com",
	}

	jsonBody, _ := json.Marshal(updateReq)

	svc.On("GetByID", mock.Anything, id.String()).Return(existing, nil).Once()
	svc.On("UpdateAccount", mock.Anything, mock.MatchedBy(func(acc *account.Account) bool {
		return acc.ID == id && acc.Name == "New Name" && acc.Email == "new@example.com"
	})).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/accounts/"+id.String(), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AccountResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "New Name", resp.Name)
	assert.Equal(t, "new@example.com", resp.Email)
	svc.AssertExpectations(t)
}

func TestUpdateAccount_PartialUpdate(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	id := uuid.New()
	existing := &account.Account{
		ID:    id,
		Name:  "Old",
		Email: "old@example.com",
		Login: "oldlogin",
	}

	updateReq := UpdateAccountRequest{
		Name: "New Name",
		// Email not provided
	}

	jsonBody, _ := json.Marshal(updateReq)

	svc.On("GetByID", mock.Anything, id.String()).Return(existing, nil).Once()
	svc.On("UpdateAccount", mock.Anything, mock.MatchedBy(func(acc *account.Account) bool {
		return acc.ID == id && acc.Name == "New Name" && acc.Email == "old@example.com"
	})).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/accounts/"+id.String(), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AccountResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "New Name", resp.Name)
	assert.Equal(t, "old@example.com", resp.Email)
	svc.AssertExpectations(t)
}

func TestUpdateAccount_EmailConflict(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	id := uuid.New()
	existing := &account.Account{
		ID:    id,
		Name:  "Old",
		Email: "old@example.com",
		Login: "oldlogin",
	}

	updateReq := UpdateAccountRequest{
		Email: "duplicate@example.com",
	}
	jsonBody, _ := json.Marshal(updateReq)

	svc.On("GetByID", mock.Anything, id.String()).Return(existing, nil).Once()
	svc.On("UpdateAccount", mock.Anything, mock.Anything).Return(account_service.ErrEmailAlreadyExists).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/accounts/"+id.String(), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "email already exists", errResp["error"])
	svc.AssertExpectations(t)
}

func TestUpdateAccount_InvalidJSON(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/accounts/"+uuid.New().String(), bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestUpdateAccount_NotFound(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	id := uuid.New().String()
	updateReq := UpdateAccountRequest{Name: "New"}
	jsonBody, _ := json.Marshal(updateReq)

	svc.On("GetByID", mock.Anything, id).Return(nil, account_service.ErrAccountNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/accounts/"+id, bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

func TestUpdateAccount_ServiceError(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	id := uuid.New()
	existing := &account.Account{ID: id}
	updateReq := UpdateAccountRequest{Name: "New"}
	jsonBody, _ := json.Marshal(updateReq)

	svc.On("GetByID", mock.Anything, id.String()).Return(existing, nil).Once()
	svc.On("UpdateAccount", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/v1/accounts/"+id.String(), bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}

// ---------- DeleteAccount ----------
func TestDeleteAccount_Success(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	id := uuid.New().String()
	svc.On("DeleteAccount", mock.Anything, id).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/accounts/"+id, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	svc.AssertExpectations(t)
}

func TestDeleteAccount_NotFound(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	id := uuid.New().String()
	svc.On("DeleteAccount", mock.Anything, id).Return(account_service.ErrAccountNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/accounts/"+id, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}
