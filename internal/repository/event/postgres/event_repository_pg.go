package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/neo-vai/go-events/internal/model/event"
)

type EventRepositoryPG struct {
	db *pgxpool.Pool
}

func NewEventRepositoryPG(db *pgxpool.Pool) *EventRepositoryPG {
	return &EventRepositoryPG{db: db}
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func (r *EventRepositoryPG) Create(ctx context.Context, ev *event.Event) error {
	_, err := r.db.Exec(ctx, `
        INSERT INTO events (id, account_id, username, api_key_id, name, payload, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
    `, ev.ID, ev.AccountID, ev.Username, nullIfEmpty(ev.APIKeyID), ev.Name, ev.Payload, time.Now())
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
	rows, err := r.db.Query(ctx, `
        SELECT id, account_id, username, api_key_id, name, payload, created_at
        FROM events WHERE api_key_id=$1
    `, nullIfEmpty(apiKeyID))
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

func (r *EventRepositoryPG) Update(ctx context.Context, ev *event.Event) error {
	_, err := r.db.Exec(ctx, `
        UPDATE events SET account_id=$1, username=$2, api_key_id=$3, name=$4, payload=$5
        WHERE id=$6
    `, ev.AccountID, ev.Username, nullIfEmpty(ev.APIKeyID), ev.Name, ev.Payload, ev.ID)
	return err
}
