package event

import (
	"context"

	"github.com/neo-vai/go-events/internal/model/event"
)

type EventRepository interface {
	Create(ctx context.Context, event *event.Event) error
	GetByID(ctx context.Context, id string) (*event.Event, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*event.Event, error)
	GetByAccountAndUser(ctx context.Context, accountID, username string) ([]*event.Event, error)
	GetByAPIKeyID(ctx context.Context, apiKeyID string) ([]*event.Event, error)
}
