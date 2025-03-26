package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/model/event"
)

type EventRepositoryPG struct {
	db *pgxpool.Pool
}

func NewEventRepositoryPG(db *pgxpool.Pool) *EventRepositoryPG {
	return &EventRepositoryPG{db: db}
}

// toNullableUUID преобразует строку в uuid.UUID или nil, если строка пуста.
func toNullableUUID(s string) interface{} {
	if s == "" {
		return nil
	}
	uid, err := uuid.Parse(s)
	if err != nil {
		return s // fallback to string if parsing fails (should not happen with valid data)
	}
	return uid
}

func (r *EventRepositoryPG) Create(ctx context.Context, ev *event.Event) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO events (id, account_id, username, api_key_id, name, payload, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, ev.ID, ev.AccountID, ev.Username, toNullableUUID(ev.APIKeyID), ev.Name, ev.Payload, time.Now())
	return err
}

func (r *EventRepositoryPG) GetByID(ctx context.Context, id string) (*event.Event, error) {
	row := r.db.QueryRow(ctx, `
        SELECT id, account_id, username, api_key_id, name, payload, created_at
        FROM events WHERE id=$1
    `, id)

	ev := &event.Event{}
	var apiKeyID *string
	err := row.Scan(&ev.ID, &ev.AccountID, &ev.Username, &apiKeyID, &ev.Name, &ev.Payload, &ev.CreatedAt)
	if err != nil {
		return nil, err
	}
	if apiKeyID != nil {
		ev.APIKeyID = *apiKeyID
	} else {
		ev.APIKeyID = ""
	}
	return ev, nil
}

func (r *EventRepositoryPG) GetByAccountID(ctx context.Context, accountID string) ([]*event.Event, error) {
	rows, err := r.db.Query(ctx, `
        SELECT id, account_id, username, api_key_id, name, payload, created_at
        FROM events WHERE account_id=$1
    `, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*event.Event
	for rows.Next() {
		ev := &event.Event{}
		var apiKeyID *string
		if err := rows.Scan(&ev.ID, &ev.AccountID, &ev.Username, &apiKeyID, &ev.Name, &ev.Payload, &ev.CreatedAt); err != nil {
			return nil, err
		}
		if apiKeyID != nil {
			ev.APIKeyID = *apiKeyID
		}
		events = append(events, ev)
	}
	return events, nil
}

func (r *EventRepositoryPG) GetByAccountAndUser(ctx context.Context, accountID, username string) ([]*event.Event, error) {
	rows, err := r.db.Query(ctx, `
        SELECT id, account_id, username, api_key_id, name, payload, created_at
        FROM events WHERE account_id=$1 AND username=$2
    `, accountID, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*event.Event
	for rows.Next() {
		ev := &event.Event{}
		var apiKeyID *string
		if err := rows.Scan(&ev.ID, &ev.AccountID, &ev.Username, &apiKeyID, &ev.Name, &ev.Payload, &ev.CreatedAt); err != nil {
			return nil, err
		}
		if apiKeyID != nil {
			ev.APIKeyID = *apiKeyID
		}
		events = append(events, ev)
	}
	return events, nil
}

func (r *EventRepositoryPG) GetByAPIKeyID(ctx context.Context, apiKeyID string) ([]*event.Event, error) {
	if apiKeyID == "" {
		rows, err := r.db.Query(ctx, `
            SELECT id, account_id, username, api_key_id, name, payload, created_at
            FROM events WHERE api_key_id IS NULL
        `)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		var events []*event.Event
		for rows.Next() {
			ev := &event.Event{}
			var apiKeyIDPtr *string
			if err := rows.Scan(&ev.ID, &ev.AccountID, &ev.Username, &apiKeyIDPtr, &ev.Name, &ev.Payload, &ev.CreatedAt); err != nil {
				return nil, err
			}
			if apiKeyIDPtr != nil {
				ev.APIKeyID = *apiKeyIDPtr
			}
			events = append(events, ev)
		}
		return events, nil
	}

	uid, err := uuid.Parse(apiKeyID)
	if err != nil {
		return nil, err
	}
	rows, err := r.db.Query(ctx, `
        SELECT id, account_id, username, api_key_id, name, payload, created_at
        FROM events WHERE api_key_id = $1
    `, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*event.Event
	for rows.Next() {
		ev := &event.Event{}
		var apiKeyIDPtr *string
		if err := rows.Scan(&ev.ID, &ev.AccountID, &ev.Username, &apiKeyIDPtr, &ev.Name, &ev.Payload, &ev.CreatedAt); err != nil {
			return nil, err
		}
		if apiKeyIDPtr != nil {
			ev.APIKeyID = *apiKeyIDPtr
		}
		events = append(events, ev)
	}
	return events, nil
}

// ListAll retrieves events with pagination, sorting, and filtering.
func (r *EventRepositoryPG) ListAll(ctx context.Context, page, limit int, sort, order string, filters map[string]interface{}) ([]*event.Event, int64, error) {
	offset := (page - 1) * limit

	baseQuery := `
		SELECT id, account_id, username, api_key_id, name, payload, created_at
		FROM events
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM events WHERE 1=1`

	args := []interface{}{}
	argIdx := 1
	var whereClauses []string

	if q, ok := filters["q"].(string); ok && q != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(username ILIKE $%d OR name ILIKE $%d OR payload::text ILIKE $%d)", argIdx, argIdx+1, argIdx+2))
		pattern := "%" + q + "%"
		args = append(args, pattern, pattern, pattern)
		argIdx += 3
	}
	if accountID, ok := filters["account_id"].(string); ok && accountID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("account_id = $%d", argIdx))
		args = append(args, accountID)
		argIdx++
	}
	if apiKeyID, ok := filters["api_key_id"].(string); ok && apiKeyID != "" {
		if apiKeyID == "null" {
			whereClauses = append(whereClauses, "api_key_id IS NULL")
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("api_key_id = $%d", argIdx))
			args = append(args, apiKeyID)
			argIdx++
		}
	}

	if len(whereClauses) > 0 {
		baseQuery += " AND " + strings.Join(whereClauses, " AND ")
		countQuery += " AND " + strings.Join(whereClauses, " AND ")
	}

	allowedSortFields := map[string]bool{"created_at": true, "username": true, "name": true}
	if sort != "" && allowedSortFields[sort] {
		orderDir := "ASC"
		if strings.ToUpper(order) == "DESC" {
			orderDir = "DESC"
		}
		baseQuery += fmt.Sprintf(" ORDER BY %s %s", sort, orderDir)
	} else {
		baseQuery += " ORDER BY created_at DESC"
	}

	baseQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	var total int64
	err := r.db.QueryRow(ctx, countQuery, args[:argIdx-1]...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, baseQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*event.Event
	for rows.Next() {
		ev := &event.Event{}
		var apiKeyID *string
		err := rows.Scan(&ev.ID, &ev.AccountID, &ev.Username, &apiKeyID, &ev.Name, &ev.Payload, &ev.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		if apiKeyID != nil {
			ev.APIKeyID = *apiKeyID
		}
		events = append(events, ev)
	}
	return events, total, nil
}
