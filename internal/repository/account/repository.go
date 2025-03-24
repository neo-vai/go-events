package account

import (
	"github.com/neo-vai/go-events/internal/model/account"
)

type AccountRepository interface {
	Create(account *account.Account) error
	GetByID(id string) (*account.Account, error)
	GetByLogin(login string) (*account.Account, error)
	Update(account *account.Account) error
	Delete(id string) error
}
