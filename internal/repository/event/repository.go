package event

import (
	"github.com/neo-vai/go-events/internal/model/event"
)

type EventRepository interface {
	Create(event *event.Event) error
	GetByID(id string) (*event.Event, error)
	GetByAccountID(accountID string) ([]*event.Event, error)
	GetByAccountAndUser(accountID, user string) ([]*event.Event, error)
	GetByAPIKeyID(apiKeyID string) ([]*event.Event, error)
}
