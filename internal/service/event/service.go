package event

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/neo-vai/go-events/internal/model/event"
	apikeyRepo "github.com/neo-vai/go-events/internal/repository/apikey"
)

// Domain errors
var (
	ErrEventNotFound  = errors.New("event not found")
	ErrInvalidEventID = errors.New("invalid event ID")
	ErrInvalidAPIKey  = errors.New("invalid API key")
	ErrAPIKeyNotOwned = errors.New("API key does not belong to account")
	ErrAPIKeyInactive = errors.New("API key is inactive")
)

type EventRepository interface {
	Create(ctx context.Context, event *event.Event) error
	GetByID(ctx context.Context, id string) (*event.Event, error)
	GetByAccountID(ctx context.Context, accountID string) ([]*event.Event, error)
	GetByAccountAndUser(ctx context.Context, accountID, username string) ([]*event.Event, error)
	GetByAPIKeyID(ctx context.Context, apiKeyID string) ([]*event.Event, error)
	ListAll(ctx context.Context, offset, limit int, sort, order string, filters map[string]interface{}) ([]*event.Event, int64, error)
}

type EventService struct {
	repo       EventRepository
	apiKeyRepo apikeyRepo.APIKeyRepository // optional, may be nil in worker context
}

func NewEventService(repo EventRepository, apiKeyRepo apikeyRepo.APIKeyRepository) *EventService {
	return &EventService{
		repo:       repo,
		apiKeyRepo: apiKeyRepo,
	}
}

func (s *EventService) CreateEvent(ctx context.Context, ev *event.Event) error {
	// Validate APIKeyID if provided and apiKeyRepo is available
	if ev.APIKeyID != "" && s.apiKeyRepo != nil {
		keyID, err := uuid.Parse(ev.APIKeyID)
		if err != nil {
			return ErrInvalidAPIKey
		}
		key, err := s.apiKeyRepo.GetByID(ctx, keyID)
		if err != nil {
			return ErrInvalidAPIKey
		}
		if key.AccountID.String() != ev.AccountID {
			return ErrAPIKeyNotOwned
		}
		if !key.Active {
			return ErrAPIKeyInactive
		}
	}

	// If ID and CreatedAt are not set, generate them (for worker consumption)
	if ev.ID == "" {
		ev.ID = uuid.New().String()
	}
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now()
	}

	err := s.repo.Create(ctx, ev)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return errors.New("invalid account")
		}
		return err
	}
	return nil
}

func (s *EventService) GetByID(ctx context.Context, id string) (*event.Event, error) {
	ev, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrEventNotFound
	}
	return ev, nil
}

// ListEvents is a legacy method for backward compatibility (used by worker? no, but keep for now).
// It returns up to 1000 most recent events for the account.
func (s *EventService) ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error) {
	events, _, err := s.ListEventsPaginated(ctx, accountID, 0, 1000, "created_at", "DESC", username, apiKeyID, "")
	return events, err
}

// ListEventsPaginated retrieves events for a specific account with pagination and filters.
func (s *EventService) ListEventsPaginated(
	ctx context.Context,
	accountID string,
	offset, limit int,
	sort, order string,
	username, apiKeyID, searchQuery string,
) ([]*event.Event, int64, error) {
	const defaultLimit = 20
	const maxLimit = 100

	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	filters := make(map[string]interface{})
	filters["account_id"] = accountID

	if username != "" {
		filters["username"] = username
	}
	if apiKeyID != "" {
		filters["api_key_id"] = apiKeyID
	}
	if searchQuery != "" {
		filters["q"] = searchQuery
	}

	// Map sort field names to database column names
	sortMap := map[string]string{
		"id":        "id",
		"createdAt": "created_at",
		"username":  "username",
		"name":      "name",
	}
	if dbSort, ok := sortMap[sort]; ok {
		sort = dbSort
	} else {
		sort = "created_at"
	}
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	return s.repo.ListAll(ctx, offset, limit, sort, order, filters)
}

// ListAllEvents returns all events (admin) with pagination and filters.
func (s *EventService) ListAllEvents(
	ctx context.Context,
	offset, limit int,
	sort, order string,
	filters map[string]interface{},
) ([]*event.Event, int64, error) {
	const defaultLimit = 20
	const maxLimit = 100

	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	sortMap := map[string]string{
		"id":        "id",
		"createdAt": "created_at",
		"username":  "username",
		"name":      "name",
	}
	if dbSort, ok := sortMap[sort]; ok {
		sort = dbSort
	} else {
		sort = "created_at"
	}
	if order != "ASC" && order != "DESC" {
		order = "DESC"
	}

	return s.repo.ListAll(ctx, offset, limit, sort, order, filters)
}
