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
	GetByAccountAndUser(ctx context.Context, accountID, username string) ([]*event.Event, error)
	GetByAPIKeyID(ctx context.Context, apiKeyID string) ([]*event.Event, error)
}

type EventService struct {
	repo EventRepository
}

func NewEventService(repo EventRepository) *EventService {
	return &EventService{repo: repo}
}

func (s *EventService) CreateEvent(ctx context.Context, ev *event.Event) error {
	ev.ID = uuid.New().String()
	ev.CreatedAt = time.Now()
	return s.repo.Create(ctx, ev)
}

func (s *EventService) GetByID(ctx context.Context, id string) (*event.Event, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *EventService) ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error) {
	switch {
	case accountID != "" && username != "" && apiKeyID != "":
		byUser, err := s.repo.GetByAccountAndUser(ctx, accountID, username)
		if err != nil {
			return nil, err
		}
		var result []*event.Event
		for _, e := range byUser {
			if e.APIKeyID == apiKeyID {
				result = append(result, e)
			}
		}
		return result, nil

	case accountID != "" && username != "":
		return s.repo.GetByAccountAndUser(ctx, accountID, username)

	case accountID != "" && apiKeyID != "":
		byAccount, err := s.repo.GetByAccountID(ctx, accountID)
		if err != nil {
			return nil, err
		}
		var result []*event.Event
		for _, e := range byAccount {
			if e.APIKeyID == apiKeyID {
				result = append(result, e)
			}
		}
		return result, nil

	case username != "" && apiKeyID != "":
		byKey, err := s.repo.GetByAPIKeyID(ctx, apiKeyID)
		if err != nil {
			return nil, err
		}
		var result []*event.Event
		for _, e := range byKey {
			if e.Username == username {
				result = append(result, e)
			}
		}
		return result, nil

	case accountID != "":
		return s.repo.GetByAccountID(ctx, accountID)

	case username != "":
		return []*event.Event{}, nil

	case apiKeyID != "":
		return s.repo.GetByAPIKeyID(ctx, apiKeyID)

	default:
		return []*event.Event{}, nil
	}
}
