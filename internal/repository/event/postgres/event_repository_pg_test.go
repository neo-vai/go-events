package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/model/event"
	accRepo "github.com/neo-vai/go-events/internal/repository/account/postgres"
	"github.com/neo-vai/go-events/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestAccount возвращает UUID аккаунта
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

func createTestEvent(t *testing.T, pool *pgxpool.Pool, accountID uuid.UUID, apiKeyID string) *event.Event {
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	ev := &event.Event{
		ID:        uuid.New().String(),
		AccountID: accountID.String(), // преобразуем в string для модели Event
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

func TestEventRepository_Update(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	ev := createTestEvent(t, pool, accountID, "")
	ev.Username = "bob"
	err := repo.Update(ctx, ev)
	require.NoError(t, err)

	updated, err := repo.GetByID(ctx, ev.ID)
	require.NoError(t, err)
	assert.Equal(t, "bob", updated.Username)
}

func TestEventRepository_GetByAccountAndUser(t *testing.T) {
	pool := testutil.SetupTestDB(t)
	repo := NewEventRepositoryPG(pool)
	ctx := context.Background()
	accountID := createTestAccount(t, pool)

	ev1 := createTestEvent(t, pool, accountID, "")
	ev1.Username = "bob"
	err := repo.Update(ctx, ev1)
	require.NoError(t, err)

	ev2 := createTestEvent(t, pool, accountID, "")
	ev2.Username = "alice"
	err = repo.Update(ctx, ev2)
	require.NoError(t, err)

	// Передаём accountID как строку, так как метод репозитория ожидает string
	events, err := repo.GetByAccountAndUser(ctx, accountID.String(), "bob")
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "bob", events[0].Username)
}
