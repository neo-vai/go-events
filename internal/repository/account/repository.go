package account

import (
	"context"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/account"
)

type AccountRepository interface {
	Create(ctx context.Context, account *account.Account) error
	GetByID(ctx context.Context, id uuid.UUID) (*account.Account, error)
	GetByLogin(ctx context.Context, login string) (*account.Account, error)
	Update(ctx context.Context, account *account.Account) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int, sort, order string, filters map[string]interface{}) ([]*account.Account, int64, error)
}
