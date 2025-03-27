package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/neo-vai/go-events/internal/repository/apikey/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockAPIKeyRepository struct {
	mock.Mock
}

func (m *MockAPIKeyRepository) ListAll(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*apikey.APIKey, int64, error) {
	args := m.Called(ctx, page, limit, sort, order, filters)
	return args.Get(0).([]*apikey.APIKey), args.Get(1).(int64), args.Error(2)
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
func (m *MockAPIKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*apikey.APIKey, error) {
	args := m.Called(ctx, keyHash)
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

	// Mock the Create call and capture the key to verify hash
	repo.On("Create", ctx, mock.AnythingOfType("*apikey.APIKey")).Return(nil).Run(func(args mock.Arguments) {
		key := args.Get(1).(*apikey.APIKey)
		assert.NotEmpty(t, key.KeyHash)
		assert.NotEmpty(t, key.PlainKey)
		// Verify that hash matches the plain key
		hasher := sha256.New()
		hasher.Write([]byte(key.PlainKey))
		expectedHash := hex.EncodeToString(hasher.Sum(nil))
		assert.Equal(t, expectedHash, key.KeyHash)
	}).Once()

	key, err := svc.Generate(ctx, accountID)
	assert.NoError(t, err)
	assert.NotEmpty(t, key.ID)
	assert.NotEmpty(t, key.PlainKey)
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
	expected := &apikey.APIKey{ID: id, KeyHash: "somehash"}
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
	repo.On("GetByID", ctx, id).Return(nil, ErrKeyNotFound).Once()
	err := svc.UpdateActive(ctx, id.String(), true)
	assert.ErrorIs(t, err, ErrKeyNotFound)
	repo.AssertExpectations(t)
}

func TestAPIKeyService_Delete_NotFound(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	id := uuid.New()
	repo.On("Delete", ctx, id).Return(ErrKeyNotFound).Once()
	err := svc.Delete(ctx, id.String())
	assert.ErrorIs(t, err, ErrKeyNotFound)
	repo.AssertExpectations(t)
}

func TestAPIKeyService_UpdateActive_RepoNoRows(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	id := uuid.New()
	key := &apikey.APIKey{ID: id, Active: true}
	repo.On("GetByID", ctx, id).Return(key, nil).Once()
	repo.On("Update", ctx, key).Return(postgres.ErrNoRowsAffected).Once()
	err := svc.UpdateActive(ctx, id.String(), false)
	assert.ErrorIs(t, err, ErrKeyNotFound)
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
	plainKey := "my-valid-plain-key"
	hasher := sha256.New()
	hasher.Write([]byte(plainKey))
	hash := hex.EncodeToString(hasher.Sum(nil))

	apiKeyObj := &apikey.APIKey{AccountID: accountID, KeyHash: hash, Active: true}

	repo.On("GetByKeyHash", ctx, hash).Return(apiKeyObj, nil).Once()
	gotAccountID, gotRole, err := svc.ValidateAPIKey(ctx, plainKey)
	assert.NoError(t, err)
	assert.Equal(t, accountID.String(), gotAccountID)
	assert.Equal(t, "", gotRole) // role is not fetched in this method

	// inactive key
	apiKeyObj.Active = false
	repo.On("GetByKeyHash", ctx, hash).Return(apiKeyObj, nil).Once()
	_, _, err = svc.ValidateAPIKey(ctx, plainKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inactive")
}

func TestAPIKeyService_ValidateAPIKey_NotFound(t *testing.T) {
	repo := new(MockAPIKeyRepository)
	svc := NewAPIKeyService(repo)
	ctx := context.Background()
	plainKey := "missing"
	hasher := sha256.New()
	hasher.Write([]byte(plainKey))
	hash := hex.EncodeToString(hasher.Sum(nil))

	repo.On("GetByKeyHash", ctx, hash).Return(nil, errors.New("not found")).Once()
	_, _, err := svc.ValidateAPIKey(ctx, plainKey)
	assert.Error(t, err)
	repo.AssertExpectations(t)
}
