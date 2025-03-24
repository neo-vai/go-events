package account

import (
	"time"

	"github.com/neo-vai/go-events/internal/model/apikey"
)

type Account struct {
	ID           string
	Name         string
	Email        string
	Login        string
	PasswordHash string
	CreatedAt    time.Time
	APIKeys      []apikey.APIKey
}
