package apikey

import (
	"github.com/neo-vai/go-events/internal/model/api_key"
)

type APIKeyRepository interface {
	Create(apiKey *api_key.APIKey) error
	GetByID(id string) (*api_key.APIKey, error)
	GetByAccountID(accountID string) ([]*api_key.APIKey, error)
	Update(apiKey *api_key.APIKey) error
	Delete(id string) error
}
