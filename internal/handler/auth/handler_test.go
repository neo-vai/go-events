package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestLogin_Success(t *testing.T) {
	svc := new(mockAccountService)
	router := setupAuthRouter(svc)

	accID := uuid.New() // теперь это uuid.UUID
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
	assert.Equal(t, accID.String(), resp.AccountID) // сравниваем строковое представление
}
