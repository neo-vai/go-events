package apikey

import "time"

type APIKey struct {
	ID        string
	AccountID string
	Key       string
	Active    bool
	CreatedAt time.Time
}
