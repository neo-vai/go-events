package account

import (
	"time"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/apikey"
)

type Account struct {
	ID           uuid.UUID
	Name         string
	Email        string
	Login        string
	PasswordHash string
	CreatedAt    time.Time
	APIKeys      []apikey.APIKey
}
