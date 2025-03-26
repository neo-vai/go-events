package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/neo-vai/go-events/internal/repository/apikey/postgres"
)

// Domain errors
var (
	ErrInvalidAccountID = errors.New("invalid account ID")
	ErrInvalidKeyID     = errors.New("invalid key ID")
	ErrKeyNotFound      = errors.New("API key not found")
	ErrKeyAlreadyExists = errors.New("API key already exists")
	ErrInactiveKey      = errors.New("API key is inactive")
	ErrAccountNotFound  = errors.New("account not found")
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
		return nil, ErrInvalidAccountID
	}
	key := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		Key:       uuid.New().String(),
		Active:    true,
		CreatedAt: time.Now(),
	}
	err = s.repo.Create(ctx, key)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrKeyAlreadyExists
		}
		// Проверка на нарушение внешнего ключа (account_id не существует)
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return key, nil
}

// GetByID возвращает ключ по ID (string -> uuid.UUID)
func (s *APIKeyService) GetByID(ctx context.Context, idStr string) (*apikey.APIKey, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, ErrInvalidKeyID
	}
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrKeyNotFound
	}
	return key, nil
}

// UpdateActive изменяет статус ключа
func (s *APIKeyService) UpdateActive(ctx context.Context, idStr string, active bool) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ErrInvalidKeyID
	}
	key, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return ErrKeyNotFound
	}
	key.Active = active
	err = s.repo.Update(ctx, key)
	if errors.Is(err, postgres.ErrNoRowsAffected) {
		return ErrKeyNotFound
	}
	return err
}

// Delete удаляет ключ
func (s *APIKeyService) Delete(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ErrInvalidKeyID
	}
	err = s.repo.Delete(ctx, id)
	if errors.Is(err, postgres.ErrNoRowsAffected) {
		return ErrKeyNotFound
	}
	return err
}

// ListByAccount возвращает все ключи аккаунта
func (s *APIKeyService) ListByAccount(ctx context.Context, accountIDStr string) ([]*apikey.APIKey, error) {
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, ErrInvalidAccountID
	}
	keys, err := s.repo.GetByAccountID(ctx, accountID)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// ValidateAPIKey проверяет ключ и возвращает accountID как string
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, key string) (string, error) {
	apiKey, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return "", ErrKeyNotFound
	}
	if !apiKey.Active {
		return "", ErrInactiveKey
	}
	return apiKey.AccountID.String(), nil
}
