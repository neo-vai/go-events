package broker

import (
	"context"

	"github.com/neo-vai/go-events/internal/model/event"
)

// Publisher defines the interface for publishing events.
type Publisher interface {
	PublishEvent(ctx context.Context, ev *event.Event) error
}

// Subscriber defines the interface for subscribing to events.
type Subscriber interface {
	Subscribe(handler func(ev *event.Event) error) error
	Close() error
}
