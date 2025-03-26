package postgres

import (
	"context"
	"errors"
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

func (r *AccountRepositoryPG) Create(ctx context.Context, account *account.Account) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO accounts (id, name, email, login, password_hash, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, account.ID, account.Name, account.Email, account.Login, account.PasswordHash, time.Now())
	return err
}

func (r *AccountRepositoryPG) GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, email, login, password_hash, created_at
		FROM accounts WHERE id=$1
	`, id)

	acc := &account.Account{}
	if err := row.Scan(&acc.ID, &acc.Name, &acc.Email, &acc.Login, &acc.PasswordHash, &acc.CreatedAt); err != nil {
		return nil, err
	}
	return acc, nil
}

func (r *AccountRepositoryPG) GetByLogin(ctx context.Context, login string) (*account.Account, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, name, email, login, password_hash, created_at
		FROM accounts WHERE login=$1
	`, login)

	acc := &account.Account{}
	if err := row.Scan(&acc.ID, &acc.Name, &acc.Email, &acc.Login, &acc.PasswordHash, &acc.CreatedAt); err != nil {
		return nil, err
	}
	return acc, nil
}

func (r *AccountRepositoryPG) Update(ctx context.Context, account *account.Account) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE accounts SET name=$1, email=$2, login=$3, password_hash=$4 WHERE id=$5
	`, account.Name, account.Email, account.Login, account.PasswordHash, account.ID)
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
