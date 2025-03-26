package account

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/repository/account/postgres"
	"golang.org/x/crypto/bcrypt"
)

// Domain errors
var (
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrLoginAlreadyExists = errors.New("login already exists")
	ErrAccountNotFound    = errors.New("account not found")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidAccountID   = errors.New("invalid account ID")
	ErrAccountInactive    = errors.New("account is inactive")
	ErrInvalidRole        = errors.New("invalid role")
)

// PasswordHasher abstracts password hashing operations.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hashedPassword, password string) error
}

// BcryptHasher implements PasswordHasher using bcrypt.
type BcryptHasher struct {
	cost int
}

// NewBcryptHasher creates a new BcryptHasher with the given cost.
// If cost is 0, bcrypt.DefaultCost is used.
func NewBcryptHasher(cost int) *BcryptHasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return &BcryptHasher{cost: cost}
}

// Hash hashes the password using bcrypt.
func (h *BcryptHasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Compare compares a hashed password with a plain password.
func (h *BcryptHasher) Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// AccountRepository defines methods for account persistence.
type AccountRepository interface {
	Create(ctx context.Context, account *account.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error)
	GetByLogin(ctx context.Context, login string) (*account.Account, error)
	Update(ctx context.Context, account *account.Account) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*account.Account, int64, error)
}

// AccountService handles business logic for accounts.
type AccountService struct {
	repo   AccountRepository
	hasher PasswordHasher
}

// NewAccountService creates a new AccountService.
func NewAccountService(repo AccountRepository, hasher PasswordHasher) *AccountService {
	return &AccountService{
		repo:   repo,
		hasher: hasher,
	}
}

// CreateAccount creates a new account after hashing the password.
// Returns domain-specific errors on conflict.
func (s *AccountService) CreateAccount(ctx context.Context, acc *account.Account, password string) error {
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}
	acc.PasswordHash = hash
	acc.CreatedAt = time.Now()
	acc.Role = "user" // default role
	acc.Active = true // new accounts are active

	err = s.repo.Create(ctx, acc)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "accounts_email_key" {
				return ErrEmailAlreadyExists
			}
			if pgErr.ConstraintName == "accounts_login_key" {
				return ErrLoginAlreadyExists
			}
		}
		return err
	}
	return nil
}

// VerifyPassword checks if the provided password matches the stored hash.
// Also checks if account is active.
func (s *AccountService) VerifyPassword(ctx context.Context, login, password string) (bool, error) {
	acc, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return false, ErrAccountNotFound
	}
	if !acc.Active {
		return false, ErrAccountInactive
	}
	if err := s.hasher.Compare(acc.PasswordHash, password); err != nil {
		return false, ErrInvalidPassword
	}
	return true, nil
}

// UpdateAccount updates account details (non-admin fields).
func (s *AccountService) UpdateAccount(ctx context.Context, acc *account.Account) error {
	// Preserve role and active status when updating via user endpoint
	existing, err := s.repo.GetByID(ctx, acc.ID)
	if err != nil {
		return ErrAccountNotFound
	}
	acc.Role = existing.Role
	acc.Active = existing.Active

	err = s.repo.Update(ctx, acc)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "accounts_email_key" {
				return ErrEmailAlreadyExists
			}
			if pgErr.ConstraintName == "accounts_login_key" {
				return ErrLoginAlreadyExists
			}
		}
		if errors.Is(err, postgres.ErrNoRowsAffected) {
			return ErrAccountNotFound
		}
		return err
	}
	return nil
}

// DeleteAccount deletes an account by its string ID.
func (s *AccountService) DeleteAccount(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return ErrInvalidAccountID
	}
	err = s.repo.Delete(ctx, id)
	if errors.Is(err, postgres.ErrNoRowsAffected) {
		return ErrAccountNotFound
	}
	return err
}

// GetByID retrieves an account by its string ID.
func (s *AccountService) GetByID(ctx context.Context, idStr string) (*account.Account, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, ErrInvalidAccountID
	}
	acc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrAccountNotFound
	}
	return acc, nil
}

// GetByLogin retrieves an account by login.
func (s *AccountService) GetByLogin(ctx context.Context, login string) (*account.Account, error) {
	acc, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return nil, ErrAccountNotFound
	}
	return acc, nil
}

// Admin methods

// ListAccounts returns a paginated list of accounts with filters.
func (s *AccountService) ListAccounts(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*account.Account, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	return s.repo.List(ctx, page, limit, sort, order, filters)
}

// UpdateAccountAdmin updates any account fields (including role and active) by admin.
func (s *AccountService) UpdateAccountAdmin(ctx context.Context, idStr string, updates map[string]interface{}) (*account.Account, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, ErrInvalidAccountID
	}
	acc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	// Apply allowed updates
	if name, ok := updates["name"].(string); ok {
		acc.Name = name
	}
	if email, ok := updates["email"].(string); ok {
		acc.Email = email
	}
	if login, ok := updates["login"].(string); ok {
		acc.Login = login
	}
	if role, ok := updates["role"].(string); ok {
		if role != "user" && role != "admin" {
			return nil, ErrInvalidRole
		}
		acc.Role = role
	}
	if active, ok := updates["active"].(bool); ok {
		acc.Active = active
	}
	// Note: password update would be a separate flow

	err = s.repo.Update(ctx, acc)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			if pgErr.ConstraintName == "accounts_email_key" {
				return nil, ErrEmailAlreadyExists
			}
			if pgErr.ConstraintName == "accounts_login_key" {
				return nil, ErrLoginAlreadyExists
			}
		}
		if errors.Is(err, postgres.ErrNoRowsAffected) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}
	return acc, nil
}
