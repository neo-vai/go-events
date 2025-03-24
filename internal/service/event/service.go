package event

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/neo-vai/go-events/internal/model/event"
)

type EventRepository interface {
	Create(ctx context.Context, event *event.Event) error
	GetByID(ctx context.Context, id string) (*event.Event, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*event.Event, error)
	GetByAccountAndUser(ctx context.Context, accountID, user string) ([]*event.Event, error)
	GetByAPIKeyID(ctx context.Context, apiKeyID string) ([]*event.Event, error)
}

type EventService struct {
	repo EventRepository
}

func NewEventService(repo EventRepository) *EventService {
	return &EventService{repo: repo}
}

// Create new event
func (s *EventService) CreateEvent(ctx context.Context, ev *event.Event) error {
	ev.ID = uuid.New().String()
	ev.CreatedAt = time.Now()
	return s.repo.Create(ctx, ev)
}

// Fetch events by account
func (s *EventService) GetByAccount(ctx context.Context, accountID string) ([]*event.Event, error) {
	return s.repo.GetByAccountID(ctx, accountID)
}

// Fetch events by account and user
func (s *EventService) GetByAccountAndUser(ctx context.Context, accountID, user string) ([]*event.Event, error) {
	return s.repo.GetByAccountAndUser(ctx, accountID, user)
}

// Fetch events by API Key
func (s *EventService) GetByAPIKey(ctx context.Context, apiKeyID string) ([]*event.Event, error) {
	return s.repo.GetByAPIKeyID(ctx, apiKeyID)
}
