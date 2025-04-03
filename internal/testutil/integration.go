package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/broker"
	"github.com/neo-vai/go-events/internal/repository/cache"
	"github.com/stretchr/testify/require"
)

// TestDeps holds all dependencies for integration tests.
type TestDeps struct {
	DB          *pgxpool.Pool
	RedisClient *cache.RedisClient
	NATSClient  *broker.NATSClient
	Publisher   *broker.NATSPublisher
}

// SetupIntegrationTest initializes test database, redis, and nats.
// It truncates all tables before each test to ensure isolation.
func SetupIntegrationTest(t *testing.T) *TestDeps {
	t.Helper()

	ctx := context.Background()

	// PostgreSQL
	dbURL := os.Getenv("DATABASE_URL_TEST")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5433/eventtracker_test?sslmode=disable"
	}
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	db, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)

	// Truncate tables before test
	_, err = db.Exec(ctx, "TRUNCATE TABLE events, api_keys, accounts RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	// Redis
	redisAddr := os.Getenv("REDIS_URL_TEST")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD_TEST")
	if redisPassword == "" {
		redisPassword = "redispass"
	}
	redisClient, err := cache.NewRedisClient(
		redisAddr,
		redisPassword,
		0,
		3,
		10,
		5,
		5*time.Second,
		3*time.Second,
		3*time.Second,
		4*time.Second,
		300*time.Second,
	)
	require.NoError(t, err)

	// Flush Redis
	err = redisClient.Client().FlushDB(ctx).Err()
	require.NoError(t, err)

	// NATS
	natsURL := os.Getenv("BROKER_URL_TEST")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	natsClient, err := broker.NewNATSClient(natsURL, false) // JetStream disabled for simplicity in tests
	require.NoError(t, err)

	publisher := broker.NewNATSPublisher(natsClient, "events")

	t.Cleanup(func() {
		_, _ = db.Exec(ctx, "TRUNCATE TABLE events, api_keys, accounts RESTART IDENTITY CASCADE")
		db.Close()
		_ = redisClient.Close()
		natsClient.Close()
	})

	return &TestDeps{
		DB:          db,
		RedisClient: redisClient,
		NATSClient:  natsClient,
		Publisher:   publisher,
	}
}
