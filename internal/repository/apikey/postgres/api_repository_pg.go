package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/model/apikey"
)

type APIKeyRepositoryPG struct {
	db *pgxpool.Pool
}

func NewAPIKeyRepositoryPG(db *pgxpool.Pool) *APIKeyRepositoryPG {
	return &APIKeyRepositoryPG{db: db}
}

func (r *APIKeyRepositoryPG) Create(ctx context.Context, apiKey *apikey.APIKey) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO api_keys (id, account_id, key, active, created_at)
        VALUES ($1, $2, $3, $4, $5)
    `, apiKey.ID, apiKey.AccountID, apiKey.Key, apiKey.Active, time.Now())
	return err
}

func (r *APIKeyRepositoryPG) GetByID(ctx context.Context, id uuid.UUID) (*apikey.APIKey, error) {
	row := r.db.QueryRow(ctx, `
        SELECT id, account_id, key, active, created_at
        FROM api_keys WHERE id=$1
    `, id)

	key := &apikey.APIKey{}
	err := row.Scan(&key.ID, &key.AccountID, &key.Key, &key.Active, &key.CreatedAt)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (r *APIKeyRepositoryPG) GetByAccountID(ctx context.Context, accountID uuid.UUID) ([]*apikey.APIKey, error) {
	rows, err := r.db.Query(ctx, `
        SELECT id, account_id, key, active, created_at
        FROM api_keys WHERE account_id=$1
    `, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*apikey.APIKey
	for rows.Next() {
		key := &apikey.APIKey{}
		if err := rows.Scan(&key.ID, &key.AccountID, &key.Key, &key.Active, &key.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (r *APIKeyRepositoryPG) Update(ctx context.Context, apiKey *apikey.APIKey) error {
	_, err := r.db.Exec(ctx, `
        UPDATE api_keys SET key=$1, active=$2 WHERE id=$3
    `, apiKey.Key, apiKey.Active, apiKey.ID)
	return err
}

func (r *APIKeyRepositoryPG) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM api_keys WHERE id=$1`, id)
	return err
}

func (r *APIKeyRepositoryPG) GetByKey(ctx context.Context, key string) (*apikey.APIKey, error) {
	row := r.db.QueryRow(ctx, `
        SELECT id, account_id, key, active, created_at
        FROM api_keys WHERE key=$1
    `, key)

	ak := &apikey.APIKey{}
	err := row.Scan(&ak.ID, &ak.AccountID, &ak.Key, &ak.Active, &ak.CreatedAt)
	if err != nil {
		return nil, err
	}
	return ak, nil
}
