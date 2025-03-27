package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/neo-vai/go-events/internal/repository/apikey/postgres"
)

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
	GetByKeyHash(ctx context.Context, keyHash string) (*apikey.APIKey, error)
	Update(ctx context.Context, key *apikey.APIKey) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListAll(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*apikey.APIKey, int64, error)
}

type APIKeyService struct {
	repo         APIKeyRepository
	apiKeyLength int
}

func NewAPIKeyService(repo APIKeyRepository, apiKeyLength int) *APIKeyService {
	return &APIKeyService{
		repo:         repo,
		apiKeyLength: apiKeyLength,
	}
}

func (s *APIKeyService) generateSecureKey() (plainKey, hash string, err error) {
	bytes := make([]byte, s.apiKeyLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	plainKey = base64.URLEncoding.EncodeToString(bytes)
	hasher := sha256.New()
	hasher.Write([]byte(plainKey))
	hash = hex.EncodeToString(hasher.Sum(nil))
	return plainKey, hash, nil
}

func (s *APIKeyService) Generate(ctx context.Context, accountIDStr string) (*apikey.APIKey, error) {
	accountID, err := uuid.Parse(accountIDStr)
	if err != nil {
		return nil, ErrInvalidAccountID
	}

	plainKey, hash, err := s.generateSecureKey()
	if err != nil {
		return nil, err
	}

	key := &apikey.APIKey{
		ID:        uuid.New(),
		AccountID: accountID,
		KeyHash:   hash,
		PlainKey:  plainKey,
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

// ValidateAPIKey verifies the plain API key and returns accountID, role, and apiKeyID.
func (s *APIKeyService) ValidateAPIKey(ctx context.Context, plainKey string) (accountID string, role string, apiKeyID string, err error) {
	hasher := sha256.New()
	hasher.Write([]byte(plainKey))
	hash := hex.EncodeToString(hasher.Sum(nil))

	apiKey, err := s.repo.GetByKeyHash(ctx, hash)
	if err != nil {
		return "", "", "", ErrKeyNotFound
	}
	if !apiKey.Active {
		return "", "", "", ErrInactiveKey
	}
	return apiKey.AccountID.String(), "", apiKey.ID.String(), nil
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
		"active":    "active",
	}
	if dbSort, ok := sortMap[sort]; ok {
		sort = dbSort
	} else {
		sort = "created_at"
	}
	return s.repo.ListAll(ctx, page, limit, sort, order, filters)
}
