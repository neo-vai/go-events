package account

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
	"github.com/neo-vai/go-events/internal/model/account"
	accountService "github.com/neo-vai/go-events/internal/service/account"
	"github.com/neo-vai/go-events/internal/validator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAccountService is a mock for the AccountService interface used by handler.
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

// Helper to set mock authenticated account in context.
func setAuthAccount(c *gin.Context, accountID string) {
	c.Set(string(middleware.AccountIDKey), accountID)
}

func setupHandlerTest() (*gin.Engine, *MockAccountService) {
	gin.SetMode(gin.TestMode)
	validator.RegisterCustomValidators()
	svc := new(MockAccountService)
	h := NewHandler(svc)

	r := gin.New()
	r.Use(middleware.ValidationErrorHandler())
	// Public route
	r.POST("/api/v1/accounts", h.CreateAccount)

	// Protected routes (mock auth middleware)
	protected := r.Group("/api/v1")
	protected.Use(func(c *gin.Context) {
		// Simulate authentication by setting account ID from header for testing
		accountID := c.GetHeader("X-Test-Account-ID")
		if accountID != "" {
			setAuthAccount(c, accountID)
		}
		c.Next()
	})
	{
		protected.GET("/account", h.GetCurrentAccount)
		protected.PATCH("/account", h.UpdateCurrentAccount)
		protected.DELETE("/account", h.DeleteCurrentAccount)
	}
	return r, svc
}

// ---------- CreateAccount (public) ----------
func TestHandler_CreateAccount_Success(t *testing.T) {
	r, svc := setupHandlerTest()

	reqBody := CreateAccountRequest{
		Email:    "test@example.com",
		Password: "StrongPass123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	createdAcc := &account.Account{
		ID:     uuid.New(),
		Email:  "test@example.com",
		Role:   "user",
		Active: true,
	}

	svc.On("CreateAccount", mock.Anything, mock.AnythingOfType("*account.Account"), "StrongPass123").
		Run(func(args mock.Arguments) {
			acc := args.Get(1).(*account.Account)
			assert.Equal(t, "test@example.com", acc.Email)
			// Simulate ID generation in service
			acc.ID = createdAcc.ID
		}).
		Return(nil).Once()

	svc.On("GetByID", mock.Anything, createdAcc.ID.String()).Return(createdAcc, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var resp AccountResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", resp.Email)
	assert.Equal(t, "user", resp.Role)
	assert.NotEmpty(t, resp.ID)
	svc.AssertExpectations(t)
}

func TestHandler_CreateAccount_InvalidJSON(t *testing.T) {
	r, svc := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "CreateAccount")
}

func TestHandler_CreateAccount_MissingFields(t *testing.T) {
	r, svc := setupHandlerTest()

	reqBody := map[string]string{"email": "test@example.com"} // missing password
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	svc.AssertNotCalled(t, "CreateAccount")
}

func TestHandler_CreateAccount_EmailConflict(t *testing.T) {
	r, svc := setupHandlerTest()

	reqBody := CreateAccountRequest{
		Email:    "duplicate@example.com",
		Password: "StrongPass123",
	}
	jsonBody, _ := json.Marshal(reqBody)

	svc.On("CreateAccount", mock.Anything, mock.Anything, "StrongPass123").
		Return(accountService.ErrEmailAlreadyExists).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/accounts", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "email already exists", errResp["error"])
	svc.AssertExpectations(t)
}

// ---------- GetCurrentAccount (protected) ----------
func TestHandler_GetCurrentAccount_Success(t *testing.T) {
	r, svc := setupHandlerTest()
	accountID := uuid.New().String()

	acc := &account.Account{
		ID:     uuid.MustParse(accountID),
		Email:  "user@example.com",
		Role:   "user",
		Active: true,
	}
	svc.On("GetByID", mock.Anything, accountID).Return(acc, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/account", nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AccountResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, accountID, resp.ID)
	assert.Equal(t, "user@example.com", resp.Email)
	svc.AssertExpectations(t)
}

func TestHandler_GetCurrentAccount_Unauthorized(t *testing.T) {
	r, svc := setupHandlerTest()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/account", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	svc.AssertNotCalled(t, "GetByID")
}

func TestHandler_GetCurrentAccount_NotFound(t *testing.T) {
	r, svc := setupHandlerTest()
	accountID := uuid.New().String()

	svc.On("GetByID", mock.Anything, accountID).Return(nil, accountService.ErrAccountNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/account", nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}

// ---------- UpdateCurrentAccount (protected) ----------
func TestHandler_UpdateCurrentAccount_Success(t *testing.T) {
	r, svc := setupHandlerTest()
	accountID := uuid.New().String()

	existing := &account.Account{
		ID:     uuid.MustParse(accountID),
		Email:  "old@example.com",
		Role:   "user",
		Active: true,
	}
	svc.On("GetByID", mock.Anything, accountID).Return(existing, nil).Once()
	svc.On("UpdateAccount", mock.Anything, mock.MatchedBy(func(acc *account.Account) bool {
		return acc.ID.String() == accountID && acc.Email == "new@example.com"
	})).Return(nil).Once()

	reqBody := UpdateAccountRequest{Email: "new@example.com"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/account", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp AccountResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "new@example.com", resp.Email)
	svc.AssertExpectations(t)
}

func TestHandler_UpdateCurrentAccount_EmailConflict(t *testing.T) {
	r, svc := setupHandlerTest()
	accountID := uuid.New().String()

	existing := &account.Account{ID: uuid.MustParse(accountID), Email: "old@example.com"}
	svc.On("GetByID", mock.Anything, accountID).Return(existing, nil).Once()
	svc.On("UpdateAccount", mock.Anything, mock.Anything).Return(accountService.ErrEmailAlreadyExists).Once()

	reqBody := UpdateAccountRequest{Email: "duplicate@example.com"}
	jsonBody, _ := json.Marshal(reqBody)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PATCH", "/api/v1/account", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
	var errResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &errResp)
	assert.Equal(t, "email already exists", errResp["error"])
	svc.AssertExpectations(t)
}

// ---------- DeleteCurrentAccount (protected) ----------
func TestHandler_DeleteCurrentAccount_Success(t *testing.T) {
	r, svc := setupHandlerTest()
	accountID := uuid.New().String()

	svc.On("DeleteAccount", mock.Anything, accountID).Return(nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/account", nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	svc.AssertExpectations(t)
}

func TestHandler_DeleteCurrentAccount_NotFound(t *testing.T) {
	r, svc := setupHandlerTest()
	accountID := uuid.New().String()

	svc.On("DeleteAccount", mock.Anything, accountID).Return(accountService.ErrAccountNotFound).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/account", nil)
	req.Header.Set("X-Test-Account-ID", accountID)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	svc.AssertExpectations(t)
}
