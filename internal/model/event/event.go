package event

import "time"

type Event struct {
	ID        string
	AccountID string
	User      string
	APIKeyID  string
	Name      string
	Payload   string
	CreatedAt time.Time
}
