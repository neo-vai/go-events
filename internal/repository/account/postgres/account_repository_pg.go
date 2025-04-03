package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/model/account"
)

// ErrNoRowsAffected indicates that a DELETE/UPDATE affected zero rows.
var ErrNoRowsAffected = errors.New("no rows affected")

type AccountRepositoryPG struct {
	db *pgxpool.Pool
}

func NewAccountRepositoryPG(db *pgxpool.Pool) *AccountRepositoryPG {
	return &AccountRepositoryPG{db: db}
}

func (r *AccountRepositoryPG) Create(ctx context.Context, acc *account.Account) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO accounts (id, email, password_hash, role, active, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, acc.ID, acc.Email, acc.PasswordHash, acc.Role, acc.Active, time.Now())
	return err
}

func (r *AccountRepositoryPG) GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, active, created_at
		FROM accounts WHERE id=$1
	`, id)

	acc := &account.Account{}
	err := row.Scan(&acc.ID, &acc.Email, &acc.PasswordHash, &acc.Role, &acc.Active, &acc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

func (r *AccountRepositoryPG) GetByEmail(ctx context.Context, email string) (*account.Account, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, role, active, created_at
		FROM accounts WHERE email=$1
	`, email)

	acc := &account.Account{}
	err := row.Scan(&acc.ID, &acc.Email, &acc.PasswordHash, &acc.Role, &acc.Active, &acc.CreatedAt)
	if err != nil {
		return nil, err
	}
	return acc, nil
}

func (r *AccountRepositoryPG) Update(ctx context.Context, acc *account.Account) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE accounts SET email=$1, password_hash=$2, role=$3, active=$4 WHERE id=$5
	`, acc.Email, acc.PasswordHash, acc.Role, acc.Active, acc.ID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

func (r *AccountRepositoryPG) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM accounts WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

// List retrieves accounts with offset, limit, sorting, and filtering.
// filters map supports: "q" (search in email), "role", "active".
func (r *AccountRepositoryPG) List(ctx context.Context, offset, limit int, sort, order string, filters map[string]interface{}) ([]*account.Account, int64, error) {
	// Base query
	baseQuery := `
		SELECT id, email, password_hash, role, active, created_at
		FROM accounts
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM accounts WHERE 1=1`

	args := []interface{}{}
	argIdx := 1
	var whereClauses []string

	// Apply filters
	if q, ok := filters["q"].(string); ok && q != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("email ILIKE $%d", argIdx))
		pattern := "%" + q + "%"
		args = append(args, pattern)
		argIdx++
	}
	if role, ok := filters["role"].(string); ok && role != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("role = $%d", argIdx))
		args = append(args, role)
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

	// Sorting
	allowedSortFields := map[string]bool{
		"id": true, "email": true, "role": true, "active": true, "created_at": true,
	}
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

	// Execute count query
	var total int64
	countArgs := args[:argIdx-1]
	err := r.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Execute list query
	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var accounts []*account.Account
	for rows.Next() {
		acc := &account.Account{}
		err := rows.Scan(&acc.ID, &acc.Email, &acc.PasswordHash, &acc.Role, &acc.Active, &acc.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		accounts = append(accounts, acc)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, err
	}
	return accounts, total, nil
}
