// file: internal/repository/apikey/postgres/api_repository_pg.go

package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/model/apikey"
)

// ErrNoRowsAffected indicates that a DELETE/UPDATE affected zero rows.
var ErrNoRowsAffected = errors.New("no rows affected")

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
	tag, err := r.db.Exec(ctx, `
        UPDATE api_keys SET key=$1, active=$2 WHERE id=$3
    `, apiKey.Key, apiKey.Active, apiKey.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

func (r *APIKeyRepositoryPG) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM api_keys WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoRowsAffected
	}
	return nil
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

func (r *APIKeyRepositoryPG) ListAll(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*apikey.APIKey, int64, error) {
	offset := (page - 1) * limit

	baseQuery := `
		SELECT id, account_id, key, active, created_at
		FROM api_keys
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM api_keys WHERE 1=1`

	args := []interface{}{}
	argIdx := 1
	var whereClauses []string

	if q, ok := filters["q"].(string); ok && q != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("key ILIKE $%d", argIdx))
		args = append(args, "%"+q+"%")
		argIdx++
	}
	if accountID, ok := filters["account_id"].(string); ok && accountID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("account_id = $%d", argIdx))
		args = append(args, accountID)
		argIdx++
	}
	if active, ok := filters["active"].(bool); ok {
		whereClauses = append(whereClauses, fmt.Sprintf("active = $%d", argIdx))
		args = append(args, active)
		argIdx++
	}

	if len(whereClauses) > 0 {
		baseQuery += " AND " + strings.Join(whereClauses, " AND ")
		countQuery += " AND " + strings.Join(whereClauses, " AND ")
	}

	allowedSortFields := map[string]bool{"created_at": true, "key": true, "active": true}
	if sort != "" && allowedSortFields[sort] {
		orderDir := "ASC"
		if strings.ToUpper(order) == "DESC" {
			orderDir = "DESC"
		}
		baseQuery += fmt.Sprintf(" ORDER BY %s %s", sort, orderDir)
	} else {
		baseQuery += " ORDER BY created_at DESC"
	}

	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	var total int64
	err := r.db.QueryRow(ctx, countQuery, args[:argIdx-1]...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var keys []*apikey.APIKey
	for rows.Next() {
		key := &apikey.APIKey{}
		err := rows.Scan(&key.ID, &key.AccountID, &key.Key, &key.Active, &key.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		keys = append(keys, key)
	}
	return keys, total, nil
}
