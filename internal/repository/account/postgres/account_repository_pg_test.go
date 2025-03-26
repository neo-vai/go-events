package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/neo-vai/go-events/internal/model/event"
	apikeyRepo "github.com/neo-vai/go-events/internal/repository/apikey/postgres"
	eventRepo "github.com/neo-vai/go-events/internal/repository/event/postgres"
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
		Role:         "user",
		Active:       true,
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
	assert.Equal(t, acc.Role, saved.Role)
	assert.Equal(t, acc.Active, saved.Active)
}

func TestAccountRepository_Create_DuplicateEmail(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	email := "duplicate@example.com"
	acc1 := &account.Account{
		ID:           uuid.New(),
		Name:         "First",
		Email:        email,
		Login:        "login1",
		PasswordHash: "hash",
		Role:         "user",
		Active:       true,
	}
	require.NoError(t, repo.Create(ctx, acc1))

	acc2 := &account.Account{
		ID:           uuid.New(),
		Name:         "Second",
		Email:        email,
		Login:        "login2",
		PasswordHash: "hash",
		Role:         "user",
		Active:       true,
	}
	err := repo.Create(ctx, acc2)
	require.Error(t, err)
	var pgErr *pgconn.PgError
	assert.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23505", pgErr.Code) // unique_violation
}

func TestAccountRepository_Create_DuplicateLogin(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	login := "duplicate_login"
	acc1 := &account.Account{
		ID:           uuid.New(),
		Name:         "First",
		Email:        "first@example.com",
		Login:        login,
		PasswordHash: "hash",
		Role:         "user",
		Active:       true,
	}
	require.NoError(t, repo.Create(ctx, acc1))

	acc2 := &account.Account{
		ID:           uuid.New(),
		Name:         "Second",
		Email:        "second@example.com",
		Login:        login,
		PasswordHash: "hash",
		Role:         "user",
		Active:       true,
	}
	err := repo.Create(ctx, acc2)
	require.Error(t, err)
	var pgErr *pgconn.PgError
	assert.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23505", pgErr.Code)
}

func TestAccountRepository_GetByID_NotFound(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	_, err := repo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
}

func TestAccountRepository_GetByLogin(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	acc := &account.Account{
		ID:           uuid.New(),
		Name:         "Login Test",
		Email:        "login_test@example.com",
		Login:        "findme",
		PasswordHash: "hash",
		Role:         "user",
		Active:       true,
		CreatedAt:    time.Now(),
	}
	require.NoError(t, repo.Create(ctx, acc))

	found, err := repo.GetByLogin(ctx, "findme")
	require.NoError(t, err)
	assert.Equal(t, acc.ID, found.ID)

	_, err = repo.GetByLogin(ctx, "notexist")
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
		Role:         "user",
		Active:       true,
		CreatedAt:    time.Now(),
	}
	require.NoError(t, repo.Create(ctx, acc))

	acc.Name = "New Name"
	err := repo.Update(ctx, acc)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, acc.ID)
	require.NoError(t, err)
	assert.Equal(t, "New Name", updated.Name)
}

func TestAccountRepository_Update_Conflict(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	acc1 := &account.Account{
		ID:           uuid.New(),
		Name:         "First",
		Email:        "first@example.com",
		Login:        "login_first",
		PasswordHash: "hash",
		Role:         "user",
		Active:       true,
	}
	acc2 := &account.Account{
		ID:           uuid.New(),
		Name:         "Second",
		Email:        "second@example.com",
		Login:        "login_second",
		PasswordHash: "hash",
		Role:         "user",
		Active:       true,
	}
	require.NoError(t, repo.Create(ctx, acc1))
	require.NoError(t, repo.Create(ctx, acc2))

	acc2.Email = acc1.Email
	err := repo.Update(ctx, acc2)
	require.Error(t, err)
	var pgErr *pgconn.PgError
	assert.ErrorAs(t, err, &pgErr)
	assert.Equal(t, "23505", pgErr.Code)
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
		Role:         "user",
		Active:       true,
		CreatedAt:    time.Now(),
	}
	require.NoError(t, repo.Create(ctx, acc))

	err := repo.Delete(ctx, acc.ID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, acc.ID)
	assert.Error(t, err)
}

func TestAccountRepository_CascadeDelete(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	accountRepo := NewAccountRepositoryPG(pool)
	apiKeyRepo := apikeyRepo.NewAPIKeyRepositoryPG(pool)
	eventRepo := eventRepo.NewEventRepositoryPG(pool)
	ctx := context.Background()

	acc := &account.Account{
		ID:           uuid.New(),
		Name:         "Cascade Test",
		Email:        "cascade@example.com",
		Login:        "cascade",
		PasswordHash: "hash",
		Role:         "user",
		Active:       true,
	}
	require.NoError(t, accountRepo.Create(ctx, acc))

	key := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: acc.ID,
		Key:       "key-for-cascade",
		Active:    true,
	}
	require.NoError(t, apiKeyRepo.Create(ctx, key))

	event := &event.Event{
		ID:        uuid.New().String(),
		AccountID: acc.ID.String(),
		Username:  "user",
		Name:      "test",
		Payload:   "{}",
	}
	require.NoError(t, eventRepo.Create(ctx, event))

	err := accountRepo.Delete(ctx, acc.ID)
	require.NoError(t, err)

	_, err = apiKeyRepo.GetByID(ctx, key.ID)
	assert.Error(t, err)

	_, err = eventRepo.GetByID(ctx, event.ID)
	assert.Error(t, err)
}

func TestAccountRepository_Delete_NotFound(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	err := repo.Delete(ctx, uuid.New())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoRowsAffected)
}

func TestAccountRepository_Update_NotFound(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewAccountRepositoryPG(pool)
	ctx := context.Background()

	acc := &account.Account{
		ID:    uuid.New(),
		Name:  "Ghost",
		Email: "ghost@example.com",
		Login: "ghost",
	}
	err := repo.Update(ctx, acc)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoRowsAffected)
}
