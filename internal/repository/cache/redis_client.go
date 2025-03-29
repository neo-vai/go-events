package cache

import (
	"context"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr, password string, db int, maxRetries, poolSize, minIdleConns int,
	dialTimeout, readTimeout, writeTimeout, poolTimeout, idleTimeout time.Duration) (*RedisClient, error) {

	rdb := redis.NewClient(&redis.Options{
		Addr:            addr,
		Password:        password,
		DB:              db,
		MaxRetries:      maxRetries,
		PoolSize:        poolSize,
		MinIdleConns:    minIdleConns,
		DialTimeout:     dialTimeout,
		ReadTimeout:     readTimeout,
		WriteTimeout:    writeTimeout,
		PoolTimeout:     poolTimeout,
		ConnMaxIdleTime: idleTimeout,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("failed to connect to Redis", "error", err)
		return nil, err
	}

	slog.Info("connected to Redis", "addr", addr, "db", db)
	return &RedisClient{client: rdb}, nil
}

func (r *RedisClient) Client() *redis.Client {
	return r.client
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}
