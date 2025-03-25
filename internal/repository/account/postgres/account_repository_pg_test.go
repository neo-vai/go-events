package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountRepository_Create(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	acc := &account.Account{
		ID:           uuid.New(),
		Name:         "Test User",
		Email:        "test@example.com",
		Login:        "testuser",
		PasswordHash: "hashed",
		CreatedAt:    time.Now(),
	}
	err := repo.Create(ctx, acc)
	require.NoError(t, err)

	saved, err := repo.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, acc.Name, saved.Name)
	assert.Equal(t, acc.Email, saved.Email)
	assert.Equal(t, acc.Login, saved.Login)
	assert.Equal(t, acc.PasswordHash, saved.PasswordHash)
}

func TestAccountRepository_GetByID_NotFound(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
}

func TestAccountRepository_Update(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	acc := &account.Account{
		ID:           uuid.New(),
		Name:         "Old Name",
		Email:        "old@example.com",
		Login:        "oldlogin",
		PasswordHash: "oldhash",
		CreatedAt:    time.Now(),
	}
	err := repo.Create(ctx, acc)
	require.NoError(t, err)

	acc.Name = "New Name"
	err = repo.Update(ctx, acc)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
}

func TestAccountRepository_Delete(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	acc := &account.Account{
		ID:           uuid.New(),
		Name:         "To Delete",
		Email:        "delete@example.com",
		Login:        "delete",
		PasswordHash: "hash",
		CreatedAt:    time.Now(),
	}
	err := repo.Create(ctx, acc)
	require.NoError(t, err)

	err = repo.Delete(ctx, acc.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, acc.ID)
	assert.Error(t, err)
}
