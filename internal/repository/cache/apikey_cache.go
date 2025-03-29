package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	apiKeyCachePrefix = "apikey:hash:"
	apiKeyCacheTTL    = 5 * time.Minute
)

type CachedAPIKey struct {
	AccountID string `json:"account_id"`
	Active    bool   `json:"active"`
	APIKeyID  string `json:"api_key_id"`
}

type APIKeyCache interface {
	Get(ctx context.Context, keyHash string) (*CachedAPIKey, error)
	Set(ctx context.Context, keyHash string, data *CachedAPIKey) error
	Delete(ctx context.Context, keyHash string) error
}

type apiKeyCache struct {
	client *redis.Client
}

func NewAPIKeyCache(client *redis.Client) APIKeyCache {
	return &apiKeyCache{client: client}
}

func (c *apiKeyCache) key(keyHash string) string {
	return apiKeyCachePrefix + keyHash
}

func (c *apiKeyCache) Get(ctx context.Context, keyHash string) (*CachedAPIKey, error) {
	val, err := c.client.Get(ctx, c.key(keyHash)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("redis get error: %w", err)
	}

	var data CachedAPIKey
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}
	return &data, nil
}

func (c *apiKeyCache) Set(ctx context.Context, keyHash string, data *CachedAPIKey) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}
	return c.client.Set(ctx, c.key(keyHash), jsonData, apiKeyCacheTTL).Err()
}

func (c *apiKeyCache) Delete(ctx context.Context, keyHash string) error {
	return c.client.Del(ctx, c.key(keyHash)).Err()
}
