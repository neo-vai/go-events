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
	ListAll(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*event.Event, int64, error)
}

type EventService struct {
	repo       EventRepository
	apiKeyRepo apikeyRepo.APIKeyRepository
}

func NewEventService(repo EventRepository, apiKeyRepo apikeyRepo.APIKeyRepository) *EventService {
	return &EventService{
		repo:       repo,
		apiKeyRepo: apiKeyRepo,
	}
}

func (s *EventService) CreateEvent(ctx context.Context, ev *event.Event) error {
	// Validate APIKeyID if provided
	if ev.APIKeyID != "" {
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

	ev.ID = uuid.New().String()
	ev.CreatedAt = time.Now()
	err := s.repo.Create(ctx, ev)
	if err != nil {
		var pgErr *pgconn.PgError
		// If foreign key violation (account_id) – return original error,
		// handler will respond with 500. More detailed errors can be added if needed.
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			// foreign key violation (account_id) - should not happen if auth is correct
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

// ListEvents is a legacy method for backward compatibility.
// It returns all events matching the given filters without pagination.
// For new development, use ListEventsPaginated.
func (s *EventService) ListEvents(ctx context.Context, accountID, username, apiKeyID string) ([]*event.Event, error) {
	// Use paginated method with limit=0 to disable pagination.
	events, _, err := s.ListEventsPaginated(ctx, accountID, 1, 0, "", "", username, apiKeyID, "")
	return events, err
}

// ListEventsPaginated returns a paginated list of events with filtering and sorting capabilities.
// accountID is mandatory and enforced from the authenticated context.
// page and limit define pagination (page >= 1, limit > 0; if limit <= 0 pagination is disabled).
// sort and order define ordering (e.g., "createdAt", "DESC").
// username, apiKeyID, and searchQuery are optional filters.
// Returns the slice of events, total count (when paginated), and error.
func (s *EventService) ListEventsPaginated(
	ctx context.Context,
	accountID string,
	page, limit int,
	sort, order string,
	username, apiKeyID, searchQuery string,
) ([]*event.Event, int64, error) {
	if page < 1 {
		page = 1
	}
	// If limit <= 0, treat as "no pagination". We'll pass limit=0 to repository to skip LIMIT clause.
	paginationEnabled := limit > 0
	if !paginationEnabled {
		limit = 0
	}
	if limit > 1000 {
		limit = 1000 // protect against excessive requests
	}

	filters := make(map[string]interface{})
	// Mandatory filter: restrict to the authenticated account.
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

	// Map frontend sort fields to DB columns.
	sortMap := map[string]string{
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

	return s.repo.ListAll(ctx, page, limit, sort, order, filters)
}

// ListAllEvents is an administrative endpoint that returns a paginated list of all events
// with flexible filtering, sorting, and search capabilities.
// It is intended for use by admin handlers.
func (s *EventService) ListAllEvents(
	ctx context.Context,
	page, limit int,
	sort, order string,
	filters map[string]interface{},
) ([]*event.Event, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Map frontend sort fields to DB columns (same as used by admin UI).
	sortMap := map[string]string{
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

	// Delegate to repository's ListAll method.
	return s.repo.ListAll(ctx, page, limit, sort, order, filters)
}
