package account

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
	"golang.org/x/crypto/bcrypt"
)

type AccountRepository interface {
	Create(ctx context.Context, account *account.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error)
	GetByLogin(ctx context.Context, login string) (*account.Account, error)
	Update(ctx context.Context, account *account.Account) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AccountService struct {
	repo AccountRepository
}

func NewAccountService(repo AccountRepository) *AccountService {
	return &AccountService{repo: repo}
}

func (s *AccountService) CreateAccount(ctx context.Context, acc *account.Account, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	acc.PasswordHash = string(hash)
	acc.CreatedAt = time.Now()
	return s.repo.Create(ctx, acc)
}

func (s *AccountService) VerifyPassword(ctx context.Context, login, password string) (bool, error) {
	acc, err := s.repo.GetByLogin(ctx, login)
	if err != nil {
		return false, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(acc.PasswordHash), []byte(password))
	if err != nil {
		return false, errors.New("invalid password")
	}
	return true, nil
}

func (s *AccountService) UpdateAccount(ctx context.Context, acc *account.Account) error {
	return s.repo.Update(ctx, acc)
}

func (s *AccountService) DeleteAccount(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return errors.New("invalid account ID")
	}
	return s.repo.Delete(ctx, id)
}

func (s *AccountService) GetByID(ctx context.Context, idStr string) (*account.Account, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, errors.New("invalid account ID")
	}
	return s.repo.GetByID(ctx, id)
}

func (s *AccountService) GetByLogin(ctx context.Context, login string) (*account.Account, error) {
	return s.repo.GetByLogin(ctx, login)
}
