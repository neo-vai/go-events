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
	r := gin.Default()
	h := NewHandler(service)
	r.POST("/api/v1/accounts", h.CreateAccount)
	r.GET("/api/v1/accounts/:id", h.GetAccount)
	r.PUT("/api/v1/accounts/:id", h.UpdateAccount)
	r.DELETE("/api/v1/accounts/:id", h.DeleteAccount)
	return r
}

func TestCreateAccount(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)

	reqBody := CreateAccountRequest{
		Name:     "John",
		Email:    "john@example.com",
		Login:    "johnny",
		Password: "pass123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("CreateAccount", mock.Anything, mock.AnythingOfType("*account.Account"), "pass123").
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

func TestGetAccount_NotFound(t *testing.T) {
	svc := new(MockAccountService)
	router := setupRouter(svc)
	id := uuid.New().String()

	svc.On("GetByID", mock.Anything, id).Return(nil, errors.New("not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/accounts/"+id, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
