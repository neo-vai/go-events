package event

import "time"

type Event struct {
	ID        string
	AccountID string
	Username  string
	APIKeyID  string
	Name      string
	Payload   string
	CreatedAt time.Time
}
