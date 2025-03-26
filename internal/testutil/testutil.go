package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// SetupTestDB creates a new connection pool to a test database.
// It truncates all tables before each test to ensure isolation.
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/eventtracker_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	// Truncate tables before test to start clean
	_, err = pool.Exec(ctx, "TRUNCATE TABLE events, api_keys, accounts RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "TRUNCATE TABLE events, api_keys, accounts RESTART IDENTITY CASCADE")
		pool.Close()
	})
	return pool
}
