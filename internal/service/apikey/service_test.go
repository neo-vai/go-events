package apikey

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAPIKeyRepository struct {
	mock.Mock
}

func (m *MockAPIKeyRepository) Create(ctx context.Context, key *apikey.APIKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}
func (m *MockAPIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apikey.APIKey), args.Error(1)
}
func (m *MockAPIKeyRepository) GetByAccountID(ctx context.Context, accountID uuid.UUID) ([]*apikey.APIKey, error) {
	args := m.Called(ctx, accountID)
	return args.Get(0).([]*apikey.APIKey), args.Error(1)
}
func (m *MockAPIKeyRepository) GetByKey(ctx context.Context, key string) (*apikey.APIKey, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*apikey.APIKey), args.Error(1)
}
func (m *MockAPIKeyRepository) Update(ctx context.Context, key *apikey.APIKey) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}
func (m *MockAPIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestAPIKeyService_Generate(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	accountID := uuid.New().String()

	repo.On("Create", ctx, mock.AnythingOfType("*apikey.APIKey")).Return(nil).Once()
	key, err := svc.Generate(ctx, accountID)
	assert.NoError(t, err)
	assert.NotEmpty(t, key.ID)
	assert.NotEmpty(t, key.Key)
	assert.True(t, key.Active)
	assert.Equal(t, accountID, key.AccountID.String())
	repo.AssertExpectations(t)
}

func TestAPIKeyService_Generate_InvalidAccountID(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	_, err := svc.Generate(ctx, "not-a-uuid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid account ID")
}

func TestAPIKeyService_Generate_RepoError(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	accountID := uuid.New().String()
	repo.On("Create", ctx, mock.Anything).Return(errors.New("db error")).Once()
	_, err := svc.Generate(ctx, accountID)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestAPIKeyService_GetByID(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	id := uuid.New()
	expected := &apikey.APIKey{ID: id, Key: "key"}
	repo.On("GetByID", ctx, id).Return(expected, nil).Once()
	key, err := svc.GetByID(ctx, id.String())
	assert.NoError(t, err)
	assert.Equal(t, expected, key)
}

func TestAPIKeyService_GetByID_InvalidID(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	_, err := svc.GetByID(ctx, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key ID")
}

func TestAPIKeyService_UpdateActive(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	id := uuid.New()
	key := &apikey.APIKey{ID: id, Active: true}
	repo.On("GetByID", ctx, id).Return(key, nil).Once()
	repo.On("Update", ctx, key).Return(nil).Once()
	err := svc.UpdateActive(ctx, id.String(), false)
	assert.NoError(t, err)
	assert.False(t, key.Active)
	repo.AssertExpectations(t)
}

func TestAPIKeyService_UpdateActive_InvalidID(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	err := svc.UpdateActive(ctx, "bad", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key ID")
}

func TestAPIKeyService_UpdateActive_KeyNotFound(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	id := uuid.New()
	repo.On("GetByID", ctx, id).Return(nil, errors.New("not found")).Once()
	err := svc.UpdateActive(ctx, id.String(), true)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}

func TestAPIKeyService_Delete(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	id := uuid.New()
	repo.On("Delete", ctx, id).Return(nil).Once()
	err := svc.Delete(ctx, id.String())
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAPIKeyService_Delete_InvalidID(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	err := svc.Delete(ctx, "bad")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid key ID")
}

func TestAPIKeyService_ListByAccount(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	accountID := uuid.New()
	keys := []*apikey.APIKey{{ID: uuid.New()}, {ID: uuid.New()}}
	repo.On("GetByAccountID", ctx, accountID).Return(keys, nil).Once()
	result, err := svc.ListByAccount(ctx, accountID.String())
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestAPIKeyService_ListByAccount_InvalidAccountID(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	_, err := svc.ListByAccount(ctx, "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid account ID")
}

func TestAPIKeyService_ValidateAPIKey(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	accountID := uuid.New()
	keyStr := "valid-key"
	apiKeyObj := &apikey.APIKey{AccountID: accountID, Key: keyStr, Active: true}

	repo.On("GetByKey", ctx, keyStr).Return(apiKeyObj, nil).Once()
	gotAccountID, err := svc.ValidateAPIKey(ctx, keyStr)
	assert.NoError(t, err)
	assert.Equal(t, accountID.String(), gotAccountID)

	// inactive key
	apiKeyObj.Active = false
	repo.On("GetByKey", ctx, keyStr).Return(apiKeyObj, nil).Once()
	_, err = svc.ValidateAPIKey(ctx, keyStr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inactive")
}

func TestAPIKeyService_ValidateAPIKey_NotFound(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	repo.On("GetByKey", ctx, "missing").Return(nil, errors.New("not found")).Once()
	_, err := svc.ValidateAPIKey(ctx, "missing")
	assert.Error(t, err)
	repo.AssertExpectations(t)
}
