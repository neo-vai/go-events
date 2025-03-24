package apikey

import (
	"context"

	"github.com/neo-vai/go-events/internal/model/apikey"
)

type APIKeyRepository interface {
	Create(ctx context.Context, apiKey *apikey.APIKey) error
	GetByID(ctx context.Context, id string) (*apikey.APIKey, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*apikey.APIKey, error)
	Update(ctx context.Context, apiKey *apikey.APIKey) error
	Delete(ctx context.Context, id string) error
}
