package event

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/model/event"
	accountRepoPG "github.com/neo-vai/go-events/internal/repository/account/postgres"
	apikeyRepoPG "github.com/neo-vai/go-events/internal/repository/apikey/postgres"
	"github.com/neo-vai/go-events/internal/repository/cache"
	eventRepoPG "github.com/neo-vai/go-events/internal/repository/event/postgres"
	accountService "github.com/neo-vai/go-events/internal/service/account"
	apikeyService "github.com/neo-vai/go-events/internal/service/apikey"
	"github.com/neo-vai/go-events/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEventService(t *testing.T) (*EventService, *eventRepoPG.EventRepositoryPG, string, string, *testutil.TestDeps) {
	t.Helper()
	deps := testutil.SetupIntegrationTest(t)

	accountRepo := accountRepoPG.NewAccountRepositoryPG(deps.DB)
	apiKeyRepo := apikeyRepoPG.NewAPIKeyRepositoryPG(deps.DB)
	eventRepo := eventRepoPG.NewEventRepositoryPG(deps.DB)

	service := NewEventService(eventRepo, apiKeyRepo)

	hasher := accountService.NewBcryptHasher(10)
	accSvc := accountService.NewAccountService(accountRepo, hasher)
	acc := &account.Account{ID: uuid.New(), Email: "event-test@example.com"}
	err := accSvc.CreateAccount(context.Background(), acc, "Pass123")
	require.NoError(t, err)
	accountID := acc.ID.String()

	apiKeyCache := cache.NewAPIKeyCache(deps.RedisClient.Client())
	apiKeySvc := apikeyService.NewAPIKeyService(apiKeyRepo, apiKeyCache, 32)
	key, err := apiKeySvc.Generate(context.Background(), accountID)
	require.NoError(t, err)
	apiKeyID := key.ID.String()

	return service, eventRepo, accountID, apiKeyID, deps
}

func TestCreateEvent_Success(t *testing.T) {
	svc, repo, accountID, apiKeyID, _ := setupEventService(t)
	ctx := context.Background()

	ev := &event.Event{
		AccountID: accountID,
		Username:  "testuser",
		APIKeyID:  apiKeyID,
		Name:      "user_login",
		Payload:   `{"ip":"192.168.1.1","user_agent":"Mozilla/5.0"}`,
	}

	err := svc.CreateEvent(ctx, ev)
	require.NoError(t, err)

	assert.NotEmpty(t, ev.ID)
	assert.NotZero(t, ev.CreatedAt)

	retrieved, err := repo.GetByID(ctx, ev.ID)
	require.NoError(t, err)
	assert.Equal(t, ev.ID, retrieved.ID)
	assert.Equal(t, accountID, retrieved.AccountID)
	assert.Equal(t, "testuser", retrieved.Username)
	assert.Equal(t, apiKeyID, retrieved.APIKeyID)
	assert.Equal(t, "user_login", retrieved.Name)
	assert.JSONEq(t, `{"ip":"192.168.1.1","user_agent":"Mozilla/5.0"}`, retrieved.Payload)
}

func TestCreateEvent_WithoutAPIKey(t *testing.T) {
	svc, repo, accountID, _, _ := setupEventService(t)
	ctx := context.Background()

	ev := &event.Event{
		AccountID: accountID,
		Username:  "testuser",
		Name:      "page_view",
		Payload:   `{"page":"/home"}`,
	}

	err := svc.CreateEvent(ctx, ev)
	require.NoError(t, err)

	retrieved, err := repo.GetByID(ctx, ev.ID)
	require.NoError(t, err)
	assert.Empty(t, retrieved.APIKeyID)
}

func TestCreateEvent_InvalidAPIKey(t *testing.T) {
	svc, _, accountID, _, _ := setupEventService(t)
	ctx := context.Background()

	ev := &event.Event{
		AccountID: accountID,
		Username:  "testuser",
		APIKeyID:  uuid.New().String(),
		Name:      "test_event",
		Payload:   `{}`,
	}

	err := svc.CreateEvent(ctx, ev)
	assert.ErrorIs(t, err, ErrInvalidAPIKey)
}

func TestCreateEvent_APIKeyNotOwnedByAccount(t *testing.T) {
	svc, _, accountID, _, deps := setupEventService(t)
	ctx := context.Background()

	accountRepo := accountRepoPG.NewAccountRepositoryPG(deps.DB)
	apiKeyRepo := apikeyRepoPG.NewAPIKeyRepositoryPG(deps.DB)

	hasher := accountService.NewBcryptHasher(10)
	accSvc := accountService.NewAccountService(accountRepo, hasher)
	otherAcc := &account.Account{ID: uuid.New(), Email: "other@example.com"}
	err := accSvc.CreateAccount(ctx, otherAcc, "Pass123")
	require.NoError(t, err)

	apiKeyCache := cache.NewAPIKeyCache(deps.RedisClient.Client())
	apiKeySvc := apikeyService.NewAPIKeyService(apiKeyRepo, apiKeyCache, 32)
	otherKey, err := apiKeySvc.Generate(ctx, otherAcc.ID.String())
	require.NoError(t, err)

	// Try to create event for accountID using otherKey (which belongs to otherAcc)
	ev := &event.Event{
		AccountID: accountID,
		Username:  "testuser",
		APIKeyID:  otherKey.ID.String(),
		Name:      "test_event",
		Payload:   `{}`,
	}

	err = svc.CreateEvent(ctx, ev)
	assert.ErrorIs(t, err, ErrAPIKeyNotOwned)
}

func TestCreateEvent_InactiveAPIKey(t *testing.T) {
	svc, _, accountID, _, deps := setupEventService(t)
	ctx := context.Background()

	apiKeyRepo := apikeyRepoPG.NewAPIKeyRepositoryPG(deps.DB)
	apiKeyCache := cache.NewAPIKeyCache(deps.RedisClient.Client())
	apiKeySvc := apikeyService.NewAPIKeyService(apiKeyRepo, apiKeyCache, 32)

	key, err := apiKeySvc.Generate(ctx, accountID)
	require.NoError(t, err)

	err = apiKeySvc.UpdateActive(ctx, key.ID.String(), false)
	require.NoError(t, err)

	ev := &event.Event{
		AccountID: accountID,
		Username:  "testuser",
		APIKeyID:  key.ID.String(),
		Name:      "test_event",
		Payload:   `{}`,
	}

	err = svc.CreateEvent(ctx, ev)
	assert.ErrorIs(t, err, ErrAPIKeyInactive)
}

func TestCreateEvent_InvalidAccount(t *testing.T) {
	svc, _, _, _, _ := setupEventService(t)
	ctx := context.Background()

	ev := &event.Event{
		AccountID: uuid.New().String(),
		Username:  "testuser",
		Name:      "test_event",
		Payload:   `{}`,
	}

	err := svc.CreateEvent(ctx, ev)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid account")
}

func TestGetByID_Success(t *testing.T) {
	svc, _, accountID, apiKeyID, _ := setupEventService(t)
	ctx := context.Background()

	ev := &event.Event{
		AccountID: accountID,
		Username:  "testuser",
		APIKeyID:  apiKeyID,
		Name:      "user_login",
		Payload:   `{"ip":"192.168.1.1"}`,
	}
	err := svc.CreateEvent(ctx, ev)
	require.NoError(t, err)

	retrieved, err := svc.GetByID(ctx, ev.ID)
	require.NoError(t, err)
	assert.Equal(t, ev.ID, retrieved.ID)
	assert.Equal(t, "testuser", retrieved.Username)
}

func TestGetByID_NotFound(t *testing.T) {
	svc, _, _, _, _ := setupEventService(t)
	ctx := context.Background()

	_, err := svc.GetByID(ctx, uuid.New().String())
	assert.ErrorIs(t, err, ErrEventNotFound)
}

func TestListEventsPaginated_WithFilters(t *testing.T) {
	svc, _, accountID, apiKeyID, _ := setupEventService(t)
	ctx := context.Background()

	events := []*event.Event{
		{AccountID: accountID, Username: "alice", APIKeyID: apiKeyID, Name: "login", Payload: `{"browser":"chrome"}`},
		{AccountID: accountID, Username: "bob", APIKeyID: "", Name: "logout", Payload: `{"browser":"firefox"}`},
		{AccountID: accountID, Username: "alice", APIKeyID: apiKeyID, Name: "purchase", Payload: `{"amount":100}`},
		{AccountID: accountID, Username: "charlie", APIKeyID: apiKeyID, Name: "login", Payload: `{"browser":"safari"}`},
	}
	for _, ev := range events {
		err := svc.CreateEvent(ctx, ev)
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	t.Run("no filters", func(t *testing.T) {
		result, total, err := svc.ListEventsPaginated(ctx, accountID, 0, 10, "created_at", "ASC", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, result, 4)
	})

	t.Run("filter by username", func(t *testing.T) {
		result, total, err := svc.ListEventsPaginated(ctx, accountID, 0, 10, "created_at", "ASC", "alice", "", "")
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, result, 2)
		for _, ev := range result {
			assert.Equal(t, "alice", ev.Username)
		}
	})

	t.Run("filter by api_key_id", func(t *testing.T) {
		result, total, err := svc.ListEventsPaginated(ctx, accountID, 0, 10, "created_at", "ASC", "", apiKeyID, "")
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, result, 3)
		for _, ev := range result {
			assert.Equal(t, apiKeyID, ev.APIKeyID)
		}
	})

	t.Run("filter by search query", func(t *testing.T) {
		result, total, err := svc.ListEventsPaginated(ctx, accountID, 0, 10, "created_at", "ASC", "", "", "chrome")
		require.NoError(t, err)
		assert.Equal(t, int64(1), total)
		assert.Len(t, result, 1)
		assert.Contains(t, result[0].Payload, "chrome")
	})

	t.Run("pagination", func(t *testing.T) {
		result, total, err := svc.ListEventsPaginated(ctx, accountID, 0, 2, "created_at", "ASC", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, result, 2)

		result, total, err = svc.ListEventsPaginated(ctx, accountID, 2, 2, "created_at", "ASC", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, int64(4), total)
		assert.Len(t, result, 2)
	})

	t.Run("sorting", func(t *testing.T) {
		result, _, err := svc.ListEventsPaginated(ctx, accountID, 0, 10, "username", "ASC", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "alice", result[0].Username)
		assert.Equal(t, "alice", result[1].Username)
		assert.Equal(t, "bob", result[2].Username)
		assert.Equal(t, "charlie", result[3].Username)

		result, _, err = svc.ListEventsPaginated(ctx, accountID, 0, 10, "username", "DESC", "", "", "")
		require.NoError(t, err)
		assert.Equal(t, "charlie", result[0].Username)
	})
}

func TestListAllEvents_AdminView(t *testing.T) {
	svc, _, _, _, deps := setupEventService(t)
	ctx := context.Background()

	accountRepo := accountRepoPG.NewAccountRepositoryPG(deps.DB)
	hasher := accountService.NewBcryptHasher(10)
	accSvc := accountService.NewAccountService(accountRepo, hasher)

	acc1 := &account.Account{ID: uuid.New(), Email: "acc1@example.com"}
	acc2 := &account.Account{ID: uuid.New(), Email: "acc2@example.com"}
	err := accSvc.CreateAccount(ctx, acc1, "Pass123")
	require.NoError(t, err)
	err = accSvc.CreateAccount(ctx, acc2, "Pass123")
	require.NoError(t, err)

	events := []*event.Event{
		{AccountID: acc1.ID.String(), Username: "user1", Name: "event1", Payload: `{}`},
		{AccountID: acc1.ID.String(), Username: "user1", Name: "event2", Payload: `{}`},
		{AccountID: acc2.ID.String(), Username: "user2", Name: "event3", Payload: `{}`},
	}
	for _, ev := range events {
		err := svc.CreateEvent(ctx, ev)
		require.NoError(t, err)
	}

	t.Run("list all events", func(t *testing.T) {
		result, total, err := svc.ListAllEvents(ctx, 0, 10, "created_at", "ASC", nil)
		require.NoError(t, err)
		assert.Equal(t, int64(3), total)
		assert.Len(t, result, 3)
	})

	t.Run("filter by account_id", func(t *testing.T) {
		filters := map[string]interface{}{"account_id": acc1.ID.String()}
		result, total, err := svc.ListAllEvents(ctx, 0, 10, "created_at", "ASC", filters)
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		assert.Len(t, result, 2)
		for _, ev := range result {
			assert.Equal(t, acc1.ID.String(), ev.AccountID)
		}
	})
}

func TestListEvents_LegacyMethod(t *testing.T) {
	svc, _, accountID, apiKeyID, _ := setupEventService(t)
	ctx := context.Background()

	events := []*event.Event{
		{AccountID: accountID, Username: "alice", APIKeyID: apiKeyID, Name: "login", Payload: `{}`},
		{AccountID: accountID, Username: "bob", APIKeyID: "", Name: "logout", Payload: `{}`},
	}
	for _, ev := range events {
		err := svc.CreateEvent(ctx, ev)
		require.NoError(t, err)
	}

	result, err := svc.ListEvents(ctx, accountID, "", "")
	require.NoError(t, err)
	assert.Len(t, result, 2)

	result, err = svc.ListEvents(ctx, accountID, "alice", "")
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "alice", result[0].Username)

	result, err = svc.ListEvents(ctx, accountID, "", apiKeyID)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, apiKeyID, result[0].APIKeyID)
}
