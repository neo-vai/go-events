package event

import "github.com/neo-vai/go-events/internal/model/event"

// CreateEventRequest is the payload for creating a new event.
type CreateEventRequest struct {
	Username string `json:"username" binding:"required,alphanumdash" example:"john_doe"`
	Name     string `json:"name" binding:"required,alphanumdash" example:"user_login"`
	Payload  string `json:"payload" binding:"omitempty,json" example:"{\"ip\":\"192.168.1.1\"}"`
}

// EventResponse is the public representation of an event.
type EventResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Username  string `json:"username" example:"john_doe"`
	APIKeyID  string `json:"apiKeyId,omitempty" example:"550e8400-e29b-41d4-a716-446655440001"`
	Name      string `json:"name" example:"user_login"`
	Payload   string `json:"payload" example:"{\"ip\":\"192.168.1.1\"}"`
	CreatedAt string `json:"createdAt" example:"2023-01-01T12:00:00Z"`
}

func ToEventResponse(ev *event.Event) EventResponse {
	return EventResponse{
		ID:        ev.ID,
		Username:  ev.Username,
		APIKeyID:  ev.APIKeyID,
		Name:      ev.Name,
		Payload:   ev.Payload,
		CreatedAt: ev.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
