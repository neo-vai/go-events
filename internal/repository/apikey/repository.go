package apikey

import (
	"context"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/apikey"
)

type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *apikey.APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error)
	GetByAccountID(ctx context.Context, accountID uuid.UUID) ([]*apikey.APIKey, error)
	GetByKey(ctx context.Context, key string) (*apikey.APIKey, error)
	Update(ctx context.Context, apiKey *apikey.APIKey) error
	Delete(ctx context.Context, id uuid.UUID) error
}
