package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/neo-vai/go-events/internal/model/event"
	accRepo "github.com/neo-vai/go-events/internal/repository/account/postgres"
	apikeyRepo "github.com/neo-vai/go-events/internal/repository/apikey/postgres"
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
		Email:        uuid.New().String() + "@example.com",
		PasswordHash: "hash",
		Role:         "user",
		Active:       true,
		CreatedAt:    time.Now(),
	}
	err := repo.Create(ctx, acc)
	require.NoError(t, err)
	return id
}

func createTestAPIKey(t *testing.T, pool *pgxpool.Pool, accountID uuid.UUID) string {
	repo := apikeyRepo.NewAPIKeyRepositoryPG(pool)
	ctx := context.Background()
	key := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		KeyHash:   uuid.New().String(),
		Active:    true,
		CreatedAt: time.Now(),
	}
	err := repo.Create(ctx, key)
	require.NoError(t, err)
	return key.ID.String()
}

func createTestEvent(t *testing.T, pool *pgxpool.Pool, accountID uuid.UUID, apiKeyID string) *event.Event {
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	ev := &event.Event{
		ID:        uuid.New().String(),
		AccountID: accountID.String(),
		Username:  "john",
		APIKeyID:  apiKeyID,
		Name:      "test_event",
		Payload:   `{"key":"value"}`,
		CreatedAt: time.Now(),
	}
	err := repo.Create(ctx, ev)
	require.NoError(t, err)
	return ev
}

func TestEventRepository_CreateAndGetByID(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	ev := &event.Event{
		ID:        uuid.New().String(),
		AccountID: accountID.String(),
		Username:  "alice",
		APIKeyID:  "",
		Name:      "login",
		Payload:   `{"ip":"127.0.0.1"}`,
	}
	err := repo.Create(ctx, ev)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, ev.ID)
	require.NoError(t, err)
	assert.Equal(t, ev.ID, found.ID)
	assert.Equal(t, ev.Username, found.Username)
	assert.Equal(t, ev.Name, found.Name)
	assert.JSONEq(t, `{"ip":"127.0.0.1"}`, found.Payload)
	assert.Equal(t, "", found.APIKeyID)
}

func TestEventRepository_CreateWithAPIKey(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)
	apiKeyID := createTestAPIKey(t, pool, accountID)

	ev := &event.Event{
		ID:        uuid.New().String(),
		AccountID: accountID.String(),
		Username:  "bob",
		APIKeyID:  apiKeyID,
		Name:      "api_call",
		Payload:   `{}`,
	}
	err := repo.Create(ctx, ev)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, ev.ID)
	require.NoError(t, err)
	assert.Equal(t, apiKeyID, found.APIKeyID)
	assert.JSONEq(t, `{}`, found.Payload)
}

func TestEventRepository_GetByAccountID(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	ev1 := createTestEvent(t, pool, accountID, "")
	ev2 := createTestEvent(t, pool, accountID, "")
	_ = ev1
	_ = ev2

	events, err := repo.GetByAccountID(ctx, accountID.String())
	require.NoError(t, err)
	assert.Len(t, events, 2)

	otherAccount := createTestAccount(t, pool)
	events, err = repo.GetByAccountID(ctx, otherAccount.String())
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestEventRepository_GetByAccountAndUser(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	ev1 := createTestEvent(t, pool, accountID, "")
	ev1.Username = "bob"
	_, err := pool.Exec(ctx, "UPDATE events SET username=$1 WHERE id=$2", "bob", ev1.ID)
	require.NoError(t, err)

	ev2 := createTestEvent(t, pool, accountID, "")
	ev2.Username = "alice"
	_, err = pool.Exec(ctx, "UPDATE events SET username=$1 WHERE id=$2", "alice", ev2.ID)
	require.NoError(t, err)

	events, err := repo.GetByAccountAndUser(ctx, accountID.String(), "bob")
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "bob", events[0].Username)
}

func TestEventRepository_GetByAPIKeyID(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	apiKeyID1 := createTestAPIKey(t, pool, accountID)
	ev1 := createTestEvent(t, pool, accountID, apiKeyID1)
	ev2 := createTestEvent(t, pool, accountID, "")
	_ = ev1
	_ = ev2

	events, err := repo.GetByAPIKeyID(ctx, apiKeyID1)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, apiKeyID1, events[0].APIKeyID)

	events, err = repo.GetByAPIKeyID(ctx, "")
	require.NoError(t, err)
	assert.Len(t, events, 1)
}

func TestEventRepository_NullAPIKeyID(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	ev := &event.Event{
		ID:        uuid.New().String(),
		AccountID: accountID.String(),
		Username:  "test",
		APIKeyID:  "",
		Name:      "null_key_test",
		Payload:   `{}`,
	}
	err := repo.Create(ctx, ev)
	require.NoError(t, err)

	found, err := repo.GetByID(ctx, ev.ID)
	require.NoError(t, err)
	assert.Equal(t, "", found.APIKeyID)
	assert.JSONEq(t, `{}`, found.Payload)
}
