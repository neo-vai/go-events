package api_key

import "time"

type APIKey struct {
	ID        string
	Key       string
	Active    bool
	CreatedAt time.Time
}
