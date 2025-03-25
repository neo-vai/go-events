package account

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

type MockAccountRepository struct {
	mock.Mock
}

func (m *MockAccountRepository) Create(ctx context.Context, acc *account.Account) error {
	args := m.Called(ctx, acc)
	return args.Error(0)
}
func (m *MockAccountRepository) GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}
func (m *MockAccountRepository) GetByLogin(ctx context.Context, login string) (*account.Account, error) {
	args := m.Called(ctx, login)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*account.Account), args.Error(1)
}
func (m *MockAccountRepository) Update(ctx context.Context, acc *account.Account) error {
	args := m.Called(ctx, acc)
	return args.Error(0)
}
func (m *MockAccountRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestAccountService_CreateAccount(t *testing.T) {
	repo := new(MockAccountRepository)
	svc := NewAccountService(repo)

	ctx := context.Background()
	acc := &account.Account{ID: uuid.New(), Name: "Test", Email: "test@ex.com", Login: "test"}
	password := "secret"

	repo.On("Create", ctx, mock.AnythingOfType("*account.Account")).Return(nil).Once()
	err := svc.CreateAccount(ctx, acc, password)
	assert.NoError(t, err)
	assert.NotEmpty(t, acc.PasswordHash)
	repo.AssertExpectations(t)
}

func TestAccountService_VerifyPassword_Success(t *testing.T) {
	repo := new(MockAccountRepository)
	svc := NewAccountService(repo)

	ctx := context.Background()
	login := "test"
	password := "correct"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	acc := &account.Account{Login: login, PasswordHash: string(hashed)}

	repo.On("GetByLogin", ctx, login).Return(acc, nil).Once()
	ok, err := svc.VerifyPassword(ctx, login, password)
	assert.NoError(t, err)
	assert.True(t, ok)
	repo.AssertExpectations(t)
}

func TestAccountService_VerifyPassword_WrongPassword(t *testing.T) {
	repo := new(MockAccountRepository)
	svc := NewAccountService(repo)

	ctx := context.Background()
	login := "test"
	password := "correct"
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	acc := &account.Account{Login: login, PasswordHash: string(hashed)}

	repo.On("GetByLogin", ctx, login).Return(acc, nil).Once()
	ok, err := svc.VerifyPassword(ctx, login, "wrong")
	assert.Error(t, err)
	assert.False(t, ok)
	repo.AssertExpectations(t)
}

func TestAccountService_GetByID(t *testing.T) {
	repo := new(MockAccountRepository)
	svc := NewAccountService(repo)
	ctx := context.Background()
	id := uuid.New()
	idStr := id.String()
	expected := &account.Account{ID: id}
	repo.On("GetByID", ctx, id).Return(expected, nil).Once()
	got, err := svc.GetByID(ctx, idStr)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}
