package apikey

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	Key       string
	Active    bool
	CreatedAt time.Time
}
