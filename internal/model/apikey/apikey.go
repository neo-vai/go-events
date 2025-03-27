package apikey

import (
	"time"

	"github.com/google/uuid"
)

type APIKey struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	KeyHash   string
	PlainKey  string // only used when generating new key, not persisted
	Active    bool
	CreatedAt time.Time
}
