package account

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
	"golang.org/x/crypto/bcrypt"
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
func (s *AccountService) CreateAccount(ctx context.Context, acc *account.Account, password string) error {
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return err
	}
	acc.PasswordHash = hash
	acc.CreatedAt = time.Now()
	return s.repo.Create(ctx, acc)
}

// VerifyPassword checks if the provided password matches the stored hash.
func (s *AccountService) VerifyPassword(ctx context.Context, login, password string) (bool, error) {
	acc, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return false, err
	}
	if err := s.hasher.Compare(acc.PasswordHash, password); err != nil {
		return false, errors.New("invalid password")
	}
	return true, nil
}

// UpdateAccount updates account details.
func (s *AccountService) UpdateAccount(ctx context.Context, acc *account.Account) error {
	return s.repo.Update(ctx, acc)
}

// DeleteAccount deletes an account by its string ID.
func (s *AccountService) DeleteAccount(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.New("invalid account ID")
	}
	return s.repo.Delete(ctx, id)
}

// GetByID retrieves an account by its string ID.
func (s *AccountService) GetByID(ctx context.Context, idStr string) (*account.Account, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, errors.New("invalid account ID")
	}
	return s.repo.GetByID(ctx, id)
}

// GetByLogin retrieves an account by login.
func (s *AccountService) GetByLogin(ctx context.Context, login string) (*account.Account, error) {
	return s.repo.GetByLogin(ctx, login)
}
