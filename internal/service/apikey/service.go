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
	ListAll(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*apikey.APIKey, int64, error)
}

type APIKeyService struct {
	repo APIKeyRepository
}

func NewAPIKeyService(repo APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

// Generate creates a new API key for the given accountID.
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
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return key, nil
}

// GetByID returns a key by its ID.
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

// UpdateActive toggles the active status of an API key.
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

// Delete removes an API key.
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

// ListByAccount returns all API keys for a given account.
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

// ValidateAPIKey validates the API key and returns the associated account ID and role.
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, key string) (accountID string, role string, err error) {
	apiKey, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return "", "", ErrKeyNotFound
	}
	if !apiKey.Active {
		return "", "", ErrInactiveKey
	}
	// Note: API key validation does not fetch the role directly.
	// We need to get the account role from the associated account.
	// For simplicity, we'll do a separate query, but ideally we'd join.
	// Since the repository doesn't have that, we'll implement a simple get in service.
	// (We'll assume AccountRepository is available; but to avoid circular deps, we'll pass a function or extend APIKeyRepository.)
	// As a pragmatic solution, we'll fetch the account in the middleware adapter.
	return apiKey.AccountID.String(), "", nil
}

func (s *APIKeyService) ListAllAPIKeys(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*apikey.APIKey, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	sortMap := map[string]string{
		"createdAt": "created_at",
		"key":       "key",
		"active":    "active",
	}
	if dbSort, ok := sortMap[sort]; ok {
		sort = dbSort
	} else {
		sort = "created_at"
	}
	return s.repo.ListAll(ctx, page, limit, sort, order, filters)
}
