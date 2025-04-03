package account

import (
	"time"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/apikey"
)

type Account struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Role         string
	Active       bool
	CreatedAt    time.Time
	APIKeys      []apikey.APIKey
}
