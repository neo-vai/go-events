package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/model/apikey"
	accRepo "github.com/neo-vai/go-events/internal/repository/account/postgres"
	"github.com/neo-vai/go-events/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestAccount(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	repo := accRepo.NewAccountRepositoryPG(pool)
	ctx := context.Background()
	id := uuid.New()
	acc := &account.Account{
		ID:           id,
		Name:         "Test",
		Email:        uuid.New().String() + "@example.com",
		Login:        uuid.New().String(),
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
	}
	err := repo.Create(ctx, acc)
	require.NoError(t, err)
	return id
}

func TestAPIKeyRepository_Create(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAPIKeyRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	key := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		Key:       uuid.New().String(),
		Active:    true,
		CreatedAt: time.Now(),
	}
	err := repo.Create(ctx, key)
	require.NoError(t, err)

	var exists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM api_keys WHERE id=$1)", key.ID).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists)

	saved, err := repo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	assert.Equal(t, key.Key, saved.Key)
	assert.Equal(t, key.AccountID, saved.AccountID)
	assert.True(t, saved.Active)
}

func TestAPIKeyRepository_Create_DuplicateKey(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAPIKeyRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	keyValue := "duplicate-key-value"
	key1 := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		Key:       keyValue,
		Active:    true,
	}
	require.NoError(t, repo.Create(ctx, key1))

	key2 := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		Key:       keyValue,
		Active:    true,
	}
	err := repo.Create(ctx, key2)
	require.Error(t, err)
	var pgErr *pgconn.PgError
	assert.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23505", pgErr.Code)
}

func TestAPIKeyRepository_GetByKey(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAPIKeyRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	key := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		Key:       "my-secret-key",
		Active:    true,
		CreatedAt: time.Now(),
	}
	err := repo.Create(ctx, key)
	require.NoError(t, err)

	found, err := repo.GetByKey(ctx, "my-secret-key")
	require.NoError(t, err)
	assert.Equal(t, key.ID, found.ID)
	assert.Equal(t, accountID, found.AccountID)

	_, err = repo.GetByKey(ctx, "nonexistent")
	assert.Error(t, err)
}

func TestAPIKeyRepository_Update(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAPIKeyRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	key := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		Key:       "original",
		Active:    true,
		CreatedAt: time.Now(),
	}
	err := repo.Create(ctx, key)
	require.NoError(t, err)

	key.Active = false
	err = repo.Update(ctx, key)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, key.ID)
	require.NoError(t, err)
	assert.False(t, updated.Active)
}

func TestAPIKeyRepository_Delete(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAPIKeyRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	key := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		Key:       "to-delete",
		Active:    true,
	}
	require.NoError(t, repo.Create(ctx, key))

	err := repo.Delete(ctx, key.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, key.ID)
	assert.Error(t, err)
}

func TestAPIKeyRepository_GetByAccountID(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAPIKeyRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	key1 := &apikey.APIKey{ID: uuid.New(), AccountID: accountID, Key: "k1", Active: true, CreatedAt: time.Now()}
	key2 := &apikey.APIKey{ID: uuid.New(), AccountID: accountID, Key: "k2", Active: false, CreatedAt: time.Now()}
	require.NoError(t, repo.Create(ctx, key1))
	require.NoError(t, repo.Create(ctx, key2))

	keys, err := repo.GetByAccountID(ctx, accountID)
	require.NoError(t, err)
	assert.Len(t, keys, 2)

	otherID := createTestAccount(t, pool)
	keys, err = repo.GetByAccountID(ctx, otherID)
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestAPIKeyRepository_Delete_NotFound(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAPIKeyRepositoryPG(pool)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoRowsAffected)
}

func TestAPIKeyRepository_Update_NotFound(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAPIKeyRepositoryPG(pool)
	ctx := context.Background()

	key := &apikey.APIKey{
		ID:     uuid.New(),
		Key:    "nonexistent",
		Active: false,
	}
	err := repo.Update(ctx, key)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoRowsAffected)
}
