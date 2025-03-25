package apikey

import (
	"context"
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
