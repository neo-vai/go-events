package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/apikey"
)

type APIKeyRepository interface {
	Create(ctx context.Context, key *apikey.APIKey) error
	GetByID(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error)
	GetByAccountID(ctx context.Context, accountID uuid.UUID) ([]*apikey.APIKey, error)
	GetByKey(ctx context.Context, key string) (*apikey.APIKey, error)
	Update(ctx context.Context, key *apikey.APIKey) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type APIKeyService struct {
	repo APIKeyRepository
}

func NewAPIKeyService(repo APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// Generate создаёт новый ключ для accountID (string -> uuid.UUID)
func (s *APIKeyService) Generate(ctx context.Context, accountIDStr string) (*apikey.APIKey, error) {
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, errors.New("invalid account ID")
	}
	key := &apikey.APIKey{
		ID:        uuid.New(),
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

// GetByID возвращает ключ по ID (string -> uuid.UUID)
func (s *APIKeyService) GetByID(ctx context.Context, idStr string) (*apikey.APIKey, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, errors.New("invalid key ID")
	}
	return s.repo.GetByID(ctx, id)
}

// UpdateActive изменяет статус ключа
func (s *APIKeyService) UpdateActive(ctx context.Context, idStr string, active bool) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.New("invalid key ID")
	}
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	key.Active = active
	return s.repo.Update(ctx, key)
}

// Delete удаляет ключ
func (s *APIKeyService) Delete(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.New("invalid key ID")
	}
	return s.repo.Delete(ctx, id)
}

// ListByAccount возвращает все ключи аккаунта
func (s *APIKeyService) ListByAccount(ctx context.Context, accountIDStr string) ([]*apikey.APIKey, error) {
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, errors.New("invalid account ID")
	}
	return s.repo.GetByAccountID(ctx, accountID)
}

// ValidateAPIKey проверяет ключ и возвращает accountID как string
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, key string) (string, error) {
	apiKey, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return "", err
	}
	if !apiKey.Active {
		return "", errors.New("inactive API key")
	}
	return apiKey.AccountID.String(), nil
}
