package auth

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
	"github.com/neo-vai/go-events/internal/middleware"
	"github.com/neo-vai/go-events/internal/model/account"
	accountService "github.com/neo-vai/go-events/internal/service/account"
	"github.com/neo-vai/go-events/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAccountService struct {
	mock.Mock
}

func (m *MockAccountService) GetByEmail(ctx context.Context, email string) (*account.Account, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}

func (m *MockAccountService) VerifyPassword(ctx context.Context, email, password string) (bool, error) {
	args := m.Called(ctx, email, password)
	return args.Bool(0), args.Error(1)
}

func setupAuthHandlerTest() (*gin.Engine, *MockAccountService) {
	gin.SetMode(gin.TestMode)
	validator.RegisterCustomValidators()
	svc := new(MockAccountService)
	h := NewHandler(svc, "test-jwt-secret-with-minimum-32-chars-long", 24)

	r := gin.New()
	r.Use(middleware.ValidationErrorHandler())
	r.POST("/api/v1/login", h.Login)
	return r, svc
}

func TestLogin_Success(t *testing.T) {
	r, svc := setupAuthHandlerTest()
	accountID := uuid.New()

	reqBody := LoginRequest{
		Email:    "user@example.com",
		Password: "CorrectPass123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("VerifyPassword", mock.Anything, "user@example.com", "CorrectPass123").
		Return(true, nil).Once()
	svc.On("GetByEmail", mock.Anything, "user@example.com").
		Return(&account.Account{ID: accountID, Role: "user"}, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, accountID.String(), resp.AccountID)

	svc.AssertExpectations(t)
}

func TestLogin_InvalidJSON(t *testing.T) {
	r, svc := setupAuthHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "VerifyPassword")
}

func TestLogin_MissingEmail(t *testing.T) {
	r, svc := setupAuthHandlerTest()

	reqBody := map[string]string{"password": "Pass123"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "VerifyPassword")
}

func TestLogin_MissingPassword(t *testing.T) {
	r, svc := setupAuthHandlerTest()

	reqBody := map[string]string{"email": "user@example.com"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "VerifyPassword")
}

func TestLogin_InvalidEmailFormat(t *testing.T) {
	r, svc := setupAuthHandlerTest()

	reqBody := LoginRequest{
		Email:    "not-an-email",
		Password: "Pass123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "VerifyPassword")
}

func TestLogin_WrongPassword(t *testing.T) {
	r, svc := setupAuthHandlerTest()

	reqBody := LoginRequest{
		Email:    "user@example.com",
		Password: "WrongPass",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("VerifyPassword", mock.Anything, "user@example.com", "WrongPass").
		Return(false, accountService.ErrInvalidPassword).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid credentials", errResp["error"])

	svc.AssertExpectations(t)
}

func TestLogin_AccountNotFound(t *testing.T) {
	r, svc := setupAuthHandlerTest()

	reqBody := LoginRequest{
		Email:    "nonexistent@example.com",
		Password: "Pass123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("VerifyPassword", mock.Anything, "nonexistent@example.com", "Pass123").
		Return(false, accountService.ErrAccountNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "invalid credentials", errResp["error"])

	svc.AssertExpectations(t)
}

func TestLogin_InactiveAccount(t *testing.T) {
	r, svc := setupAuthHandlerTest()

	reqBody := LoginRequest{
		Email:    "inactive@example.com",
		Password: "Pass123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("VerifyPassword", mock.Anything, "inactive@example.com", "Pass123").
		Return(false, accountService.ErrAccountInactive).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "account is inactive", errResp["error"])

	svc.AssertExpectations(t)
}

func TestLogin_GetByEmailFails(t *testing.T) {
	r, svc := setupAuthHandlerTest()

	reqBody := LoginRequest{
		Email:    "user@example.com",
		Password: "CorrectPass123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("VerifyPassword", mock.Anything, "user@example.com", "CorrectPass123").
		Return(true, nil).Once()
	svc.On("GetByEmail", mock.Anything, "user@example.com").
		Return(nil, errors.New("database error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "failed to retrieve account", errResp["error"])

	svc.AssertExpectations(t)
}

func TestLogin_AdminRoleInToken(t *testing.T) {
	r, svc := setupAuthHandlerTest()
	accountID := uuid.New()

	reqBody := LoginRequest{
		Email:    "admin@example.com",
		Password: "AdminPass123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("VerifyPassword", mock.Anything, "admin@example.com", "AdminPass123").
		Return(true, nil).Once()
	svc.On("GetByEmail", mock.Anything, "admin@example.com").
		Return(&account.Account{ID: accountID, Role: "admin"}, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, accountID.String(), resp.AccountID)

	svc.AssertExpectations(t)
}
