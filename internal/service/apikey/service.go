package apikey

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/apikey"
)

type APIKeyRepository interface {
	Create(ctx context.Context, key *apikey.APIKey) error
	GetByID(ctx context.Context, id string) (*apikey.APIKey, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*apikey.APIKey, error)
	Update(ctx context.Context, key *apikey.APIKey) error
	Delete(ctx context.Context, id string) error
}

type APIKeyService struct {
	repo APIKeyRepository
}

func NewAPIKeyService(repo APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// Generate new API Key for account
func (s *APIKeyService) Generate(ctx context.Context, accountID string) (*apikey.APIKey, error) {
	key := &apikey.APIKey{
		ID:        uuid.New().String(),
		AccountID: accountID,
		Key:       uuid.New().String(),
		Active:    true,
		CreatedAt: time.Now(),
	}
	if err := s.repo.Create(ctx, key); err != nil {
		return nil, err
	}
	return key, nil
}

// GetByID returns API key by ID
func (s *APIKeyService) GetByID(ctx context.Context, id string) (*apikey.APIKey, error) {
	return s.repo.GetByID(ctx, id)
}

// UpdateActive sets active status of API key
func (s *APIKeyService) UpdateActive(ctx context.Context, id string, active bool) error {
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	key.Active = active
	return s.repo.Update(ctx, key)
}

// Delete API Key
func (s *APIKeyService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// ListByAccount returns all keys for account
func (s *APIKeyService) ListByAccount(ctx context.Context, accountID string) ([]*apikey.APIKey, error) {
	return s.repo.GetByAccountID(ctx, accountID)
}
