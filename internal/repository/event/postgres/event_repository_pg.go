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

func (r *EventRepositoryPG) Create(ctx context.Context, ev *event.Event) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO events (id, account_id, username, api_key_id, name, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, ev.ID, ev.AccountID, ev.Username, ev.APIKeyID, ev.Name, ev.Payload, time.Now())
	return err
}

func (r *EventRepositoryPG) GetByID(ctx context.Context, id string) (*event.Event, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, account_id, username, api_key_id, name, payload, created_at
		FROM events WHERE id=$1
	`, id)

	ev := &event.Event{}
	if err := row.Scan(&ev.ID, &ev.AccountID, &ev.Username, &ev.APIKeyID, &ev.Name, &ev.Payload, &ev.CreatedAt); err != nil {
		return nil, err
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
		if err := rows.Scan(&ev.ID, &ev.AccountID, &ev.Username, &ev.APIKeyID, &ev.Name, &ev.Payload, &ev.CreatedAt); err != nil {
			return nil, err
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
		if err := rows.Scan(&ev.ID, &ev.AccountID, &ev.Username, &ev.APIKeyID, &ev.Name, &ev.Payload, &ev.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}

func (r *EventRepositoryPG) GetByAPIKeyID(ctx context.Context, apiKeyID string) ([]*event.Event, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, account_id, username, api_key_id, name, payload, created_at
		FROM events WHERE api_key_id=$1
	`, apiKeyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*event.Event
	for rows.Next() {
		ev := &event.Event{}
		if err := rows.Scan(&ev.ID, &ev.AccountID, &ev.Username, &ev.APIKeyID, &ev.Name, &ev.Payload, &ev.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, nil
}
