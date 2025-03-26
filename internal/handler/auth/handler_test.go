package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockAccountService struct {
	mock.Mock
}

func (m *mockAccountService) GetByLogin(ctx context.Context, login string) (*account.Account, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}
func (m *mockAccountService) VerifyPassword(ctx context.Context, login, password string) (bool, error) {
	args := m.Called(ctx, login, password)
	return args.Bool(0), args.Error(1)
}

func setupAuthRouter(svc AccountService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	h := NewHandler(svc)
	r.POST("/api/v1/login", h.Login)
	return r
}

func TestMain(m *testing.M) {
	// Set a default JWT secret for tests
	os.Setenv("JWT_SECRET", "test-secret")
	os.Setenv("JWT_EXPIRES_HOURS", "1")
	code := m.Run()
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_EXPIRES_HOURS")
	os.Exit(code)
}

func TestLogin_Success(t *testing.T) {
	svc := new(mockAccountService)
	router := setupAuthRouter(svc)

	accID := uuid.New()
	svc.On("VerifyPassword", mock.Anything, "john", "secret").Return(true, nil)
	svc.On("GetByLogin", mock.Anything, "john").Return(&account.Account{ID: accID}, nil)

	reqBody := LoginRequest{Login: "john", Password: "secret"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp LoginResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp.Token)
	assert.Equal(t, accID.String(), resp.AccountID)
	svc.AssertExpectations(t)
}

func TestLogin_InvalidJSON(t *testing.T) {
	svc := new(mockAccountService)
	router := setupAuthRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "VerifyPassword", mock.Anything, mock.Anything, mock.Anything)
}

func TestLogin_MissingFields(t *testing.T) {
	svc := new(mockAccountService)
	router := setupAuthRouter(svc)

	reqBody := LoginRequest{Login: "john"} // missing password
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "VerifyPassword", mock.Anything, mock.Anything, mock.Anything)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := new(mockAccountService)
	router := setupAuthRouter(svc)

	svc.On("VerifyPassword", mock.Anything, "john", "wrong").Return(false, nil).Once()

	reqBody := LoginRequest{Login: "john", Password: "wrong"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertExpectations(t)
}

func TestLogin_VerifyPasswordError(t *testing.T) {
	svc := new(mockAccountService)
	router := setupAuthRouter(svc)

	svc.On("VerifyPassword", mock.Anything, "john", "secret").Return(false, errors.New("db error")).Once()

	reqBody := LoginRequest{Login: "john", Password: "secret"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code) // handler returns 401 for any non-success
	svc.AssertExpectations(t)
}

func TestLogin_GetByLoginError(t *testing.T) {
	svc := new(mockAccountService)
	router := setupAuthRouter(svc)

	svc.On("VerifyPassword", mock.Anything, "john", "secret").Return(true, nil).Once()
	svc.On("GetByLogin", mock.Anything, "john").Return(nil, errors.New("not found")).Once()

	reqBody := LoginRequest{Login: "john", Password: "secret"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/login", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	svc.AssertExpectations(t)
}
