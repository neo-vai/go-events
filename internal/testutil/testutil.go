package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// SetupTestDB creates a new connection pool to a test database.
// It reads DATABASE_URL_TEST from environment, or falls back to a default.
// The database is truncated after each test.
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	dsn := os.Getenv("DATABASE_URL_TEST")
	if dsn == "" {
		// Если переменная не задана, используем значение по умолчанию (для удобства локальной разработки)
		// но рекомендуется всегда задавать её явно.
		dsn = "postgres://postgres:postgres@localhost:5433/eventtracker_test?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(func() {
		// Truncate all tables to keep tests isolated
		_, err := pool.Exec(ctx, "TRUNCATE TABLE events, api_keys, accounts RESTART IDENTITY CASCADE")
		require.NoError(t, err)
		pool.Close()
	})
	return pool
}
