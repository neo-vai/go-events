package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
	accountRepoPG "github.com/neo-vai/go-events/internal/repository/account/postgres"
	apikeyRepoPG "github.com/neo-vai/go-events/internal/repository/apikey/postgres"
	"github.com/neo-vai/go-events/internal/repository/cache"
	accountService "github.com/neo-vai/go-events/internal/service/account"
	"github.com/neo-vai/go-events/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupAPIKeyService initializes all dependencies for API key tests and returns:
// - the APIKeyService
// - the APIKeyRepository
// - the AccountRepository
// - the test account ID (already created and active)
func setupAPIKeyService(t *testing.T) (*APIKeyService, *apikeyRepoPG.APIKeyRepositoryPG, *accountRepoPG.AccountRepositoryPG, string) {
	t.Helper()
	deps := testutil.SetupIntegrationTest(t)

	accountRepo := accountRepoPG.NewAccountRepositoryPG(deps.DB)
	apiKeyRepo := apikeyRepoPG.NewAPIKeyRepositoryPG(deps.DB)
	apiKeyCache := cache.NewAPIKeyCache(deps.RedisClient.Client())
	service := NewAPIKeyService(apiKeyRepo, apiKeyCache, 32)

	// Create a test account for API key association
	hasher := accountService.NewBcryptHasher(10)
	accSvc := accountService.NewAccountService(accountRepo, hasher)
	acc := &account.Account{ID: uuid.New(), Email: "apikey-test@example.com"}
	err := accSvc.CreateAccount(context.Background(), acc, "Pass123")
	require.NoError(t, err)

	return service, apiKeyRepo, accountRepo, acc.ID.String()
}

func TestGenerateAPIKey_Success(t *testing.T) {
	svc, repo, _, accountID := setupAPIKeyService(t)
	ctx := context.Background()

	key, err := svc.Generate(ctx, accountID)
	require.NoError(t, err)

	// Check returned key has PlainKey and it's not empty
	assert.NotEmpty(t, key.PlainKey)
	assert.NotEmpty(t, key.KeyHash)
	assert.Equal(t, accountID, key.AccountID.String())
	assert.True(t, key.Active)

	// Verify hash matches PlainKey
	hasher := sha256.New()
	hasher.Write([]byte(key.PlainKey))
	expectedHash := hex.EncodeToString(hasher.Sum(nil))
	assert.Equal(t, expectedHash, key.KeyHash)

	// Verify in DB
	dbKey, err := repo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, key.KeyHash, dbKey.KeyHash)
	assert.Empty(t, dbKey.PlainKey, "PlainKey should not be persisted")
	assert.True(t, dbKey.Active)
}

func TestGenerateAPIKey_InvalidAccountID(t *testing.T) {
	svc, _, _, _ := setupAPIKeyService(t)
	ctx := context.Background()

	_, err := svc.Generate(ctx, "not-a-uuid")
	assert.ErrorIs(t, err, ErrInvalidAccountID)
}

func TestGenerateAPIKey_AccountNotFound(t *testing.T) {
	svc, _, _, _ := setupAPIKeyService(t)
	ctx := context.Background()

	_, err := svc.Generate(ctx, uuid.New().String())
	assert.ErrorIs(t, err, ErrAccountNotFound)
}

func TestValidateAPIKey_Success(t *testing.T) {
	svc, _, _, accountID := setupAPIKeyService(t)
	ctx := context.Background()

	// Generate a key for the test account
	key, err := svc.Generate(ctx, accountID)
	require.NoError(t, err)

	// Validate the plain key
	accID, role, keyID, err := svc.ValidateAPIKey(ctx, key.PlainKey)
	require.NoError(t, err)
	assert.Equal(t, accountID, accID)
	assert.Equal(t, "", role) // role is not returned from ValidateAPIKey
	assert.Equal(t, key.ID.String(), keyID)

	// Second validation should hit cache (no DB call)
	accID2, role2, keyID2, err := svc.ValidateAPIKey(ctx, key.PlainKey)
	require.NoError(t, err)
	assert.Equal(t, accID, accID2)
	assert.Equal(t, role, role2)
	assert.Equal(t, keyID, keyID2)
}

func TestValidateAPIKey_Inactive(t *testing.T) {
	svc, _, _, accountID := setupAPIKeyService(t)
	ctx := context.Background()

	key, err := svc.Generate(ctx, accountID)
	require.NoError(t, err)

	// Deactivate the key
	err = svc.UpdateActive(ctx, key.ID.String(), false)
	require.NoError(t, err)

	// Validation should fail
	_, _, _, err = svc.ValidateAPIKey(ctx, key.PlainKey)
	assert.ErrorIs(t, err, ErrInactiveKey)

	// Cache should also return error immediately (no DB)
	_, _, _, err = svc.ValidateAPIKey(ctx, key.PlainKey)
	assert.ErrorIs(t, err, ErrInactiveKey)
}

func TestValidateAPIKey_InvalidKey(t *testing.T) {
	svc, _, _, _ := setupAPIKeyService(t)
	ctx := context.Background()

	_, _, _, err := svc.ValidateAPIKey(ctx, "nonexistent-key")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestUpdateActive_Success(t *testing.T) {
	svc, repo, _, accountID := setupAPIKeyService(t)
	ctx := context.Background()

	key, err := svc.Generate(ctx, accountID)
	require.NoError(t, err)

	// Deactivate
	err = svc.UpdateActive(ctx, key.ID.String(), false)
	require.NoError(t, err)

	dbKey, err := repo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	assert.False(t, dbKey.Active)

	// Cache should be invalidated
	cached, err := svc.cache.Get(ctx, key.KeyHash)
	require.NoError(t, err)
	assert.Nil(t, cached, "cache should be empty after update")

	// Activate again
	err = svc.UpdateActive(ctx, key.ID.String(), true)
	require.NoError(t, err)
	dbKey, err = repo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	assert.True(t, dbKey.Active)
}

func TestUpdateActive_InvalidKeyID(t *testing.T) {
	svc, _, _, _ := setupAPIKeyService(t)
	ctx := context.Background()

	err := svc.UpdateActive(ctx, "invalid", true)
	assert.ErrorIs(t, err, ErrInvalidKeyID)
}

func TestUpdateActive_KeyNotFound(t *testing.T) {
	svc, _, _, _ := setupAPIKeyService(t)
	ctx := context.Background()

	err := svc.UpdateActive(ctx, uuid.New().String(), true)
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestDeleteAPIKey_Success(t *testing.T) {
	svc, repo, _, accountID := setupAPIKeyService(t)
	ctx := context.Background()

	key, err := svc.Generate(ctx, accountID)
	require.NoError(t, err)

	err = svc.Delete(ctx, key.ID.String())
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, key.ID)
	assert.Error(t, err)

	// Cache should be invalidated
	cached, err := svc.cache.Get(ctx, key.KeyHash)
	require.NoError(t, err)
	assert.Nil(t, cached)
}

func TestDeleteAPIKey_InvalidKeyID(t *testing.T) {
	svc, _, _, _ := setupAPIKeyService(t)
	ctx := context.Background()

	err := svc.Delete(ctx, "invalid")
	assert.ErrorIs(t, err, ErrInvalidKeyID)
}

func TestDeleteAPIKey_NotFound(t *testing.T) {
	svc, _, _, _ := setupAPIKeyService(t)
	ctx := context.Background()

	err := svc.Delete(ctx, uuid.New().String())
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestListByAccount_Success(t *testing.T) {
	svc, _, _, accountID := setupAPIKeyService(t)
	ctx := context.Background()

	// Generate two keys
	key1, err := svc.Generate(ctx, accountID)
	require.NoError(t, err)
	key2, err := svc.Generate(ctx, accountID)
	require.NoError(t, err)

	keys, err := svc.ListByAccount(ctx, accountID)
	require.NoError(t, err)
	assert.Len(t, keys, 2)

	// Keys should not contain PlainKey when fetched from DB
	for _, k := range keys {
		assert.Empty(t, k.PlainKey)
		assert.Contains(t, []string{key1.ID.String(), key2.ID.String()}, k.ID.String())
	}
}

func TestListByAccount_InvalidAccountID(t *testing.T) {
	svc, _, _, _ := setupAPIKeyService(t)
	ctx := context.Background()

	_, err := svc.ListByAccount(ctx, "invalid")
	assert.ErrorIs(t, err, ErrInvalidAccountID)
}

func TestListAllAPIKeys_Pagination(t *testing.T) {
	svc, _, _, accountID := setupAPIKeyService(t)
	ctx := context.Background()

	// Create multiple keys for the same account
	for i := 0; i < 5; i++ {
		_, err := svc.Generate(ctx, accountID)
		require.NoError(t, err)
	}

	// List with limit
	keys, total, err := svc.ListAllAPIKeys(ctx, 0, 3, "createdAt", "DESC", nil)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, keys, 3)

	// Filter by active
	filters := map[string]interface{}{"active": true}
	keys, total, err = svc.ListAllAPIKeys(ctx, 0, 10, "createdAt", "DESC", filters)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, keys, 5)
}
