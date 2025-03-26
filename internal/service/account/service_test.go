package account

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/repository/account/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAccountRepository is a mock for AccountRepository.
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

// MockPasswordHasher is a mock for PasswordHasher.
type MockPasswordHasher struct {
	mock.Mock
}

func (m *MockPasswordHasher) Hash(password string) (string, error) {
	args := m.Called(password)
	return args.String(0), args.Error(1)
}

func (m *MockPasswordHasher) Compare(hashedPassword, password string) error {
	args := m.Called(hashedPassword, password)
	return args.Error(0)
}

func TestAccountService_CreateAccount(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	acc := &account.Account{ID: uuid.New(), Name: "Test", Email: "test@ex.com", Login: "test"}
	password := "secret"

	hasher.On("Hash", password).Return("hashed", nil)
	repo.On("Create", ctx, acc).Return(nil).Once()

	err := svc.CreateAccount(ctx, acc, password)
	assert.NoError(t, err)
	assert.Equal(t, "hashed", acc.PasswordHash)
	repo.AssertExpectations(t)
	hasher.AssertExpectations(t)
}

func TestAccountService_CreateAccount_HashingError(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	acc := &account.Account{ID: uuid.New()}
	password := "verylongpasswordthatexceedsbcryptlimit"

	hasher.On("Hash", password).Return("", errors.New("bcrypt error"))
	// Репозиторий не должен вызываться, поэтому не настраиваем ожидания.

	err := svc.CreateAccount(ctx, acc, password)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bcrypt error")
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	hasher.AssertExpectations(t)
}

func TestAccountService_VerifyPassword_Success(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	login := "test"
	password := "correct"
	acc := &account.Account{Login: login, PasswordHash: "hashed"}

	repo.On("GetByLogin", ctx, login).Return(acc, nil).Once()
	hasher.On("Compare", "hashed", password).Return(nil).Once()

	ok, err := svc.VerifyPassword(ctx, login, password)
	assert.NoError(t, err)
	assert.True(t, ok)
	repo.AssertExpectations(t)
	hasher.AssertExpectations(t)
}

func TestAccountService_VerifyPassword_WrongPassword(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	login := "test"
	password := "wrong"
	acc := &account.Account{Login: login, PasswordHash: "hashed"}

	repo.On("GetByLogin", ctx, login).Return(acc, nil).Once()
	hasher.On("Compare", "hashed", password).Return(errors.New("mismatch")).Once()

	ok, err := svc.VerifyPassword(ctx, login, password)
	assert.Error(t, err)
	assert.False(t, ok)
	assert.Contains(t, err.Error(), "invalid password")
	repo.AssertExpectations(t)
	hasher.AssertExpectations(t)
}

func TestAccountService_VerifyPassword_UserNotFound(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	login := "nonexistent"
	repo.On("GetByLogin", ctx, login).Return(nil, errors.New("not found")).Once()

	ok, err := svc.VerifyPassword(ctx, login, "any")
	assert.Error(t, err)
	assert.False(t, ok)
	repo.AssertExpectations(t)
	hasher.AssertNotCalled(t, "Compare", mock.Anything, mock.Anything)
}

func TestAccountService_GetByID(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	id := uuid.New()
	idStr := id.String()
	expected := &account.Account{ID: id}
	repo.On("GetByID", ctx, id).Return(expected, nil).Once()

	got, err := svc.GetByID(ctx, idStr)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestAccountService_GetByID_InvalidID(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	_, err := svc.GetByID(ctx, "not-a-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid account ID")
}

func TestAccountService_GetByLogin(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	expected := &account.Account{ID: uuid.New(), Login: "test"}
	repo.On("GetByLogin", ctx, "test").Return(expected, nil).Once()

	got, err := svc.GetByLogin(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestAccountService_UpdateAccount(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	acc := &account.Account{ID: uuid.New(), Name: "Updated"}
	repo.On("Update", ctx, acc).Return(nil).Once()

	err := svc.UpdateAccount(ctx, acc)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAccountService_DeleteAccount(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	id := uuid.New()
	idStr := id.String()
	repo.On("Delete", ctx, id).Return(nil).Once()

	err := svc.DeleteAccount(ctx, idStr)
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAccountService_DeleteAccount_InvalidID(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	err := svc.DeleteAccount(ctx, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid account ID")
}

func TestAccountService_UpdateAccount_NotFound(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	acc := &account.Account{ID: uuid.New(), Name: "Ghost"}
	repo.On("Update", ctx, acc).Return(postgres.ErrNoRowsAffected).Once()

	err := svc.UpdateAccount(ctx, acc)
	assert.ErrorIs(t, err, ErrAccountNotFound)
	repo.AssertExpectations(t)
}

func TestAccountService_DeleteAccount_NotFound(t *testing.T) {
	repo := new(MockAccountRepository)
	hasher := new(MockPasswordHasher)
	svc := NewAccountService(repo, hasher)

	ctx := context.Background()
	id := uuid.New()
	repo.On("Delete", ctx, id).Return(postgres.ErrNoRowsAffected).Once()

	err := svc.DeleteAccount(ctx, id.String())
	assert.ErrorIs(t, err, ErrAccountNotFound)
	repo.AssertExpectations(t)
}
