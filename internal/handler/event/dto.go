package event

import "github.com/neo-vai/go-events/internal/model/event"

type CreateEventRequest struct {
	Username string `json:"username" binding:"required,alphanumdash"`
	Name     string `json:"name" binding:"required,alphanumdash"`
	Payload  string `json:"payload" binding:"omitempty,json"`
}

type EventResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	APIKeyID  string `json:"apiKeyId,omitempty"`
	Name      string `json:"name"`
	Payload   string `json:"payload"`
	CreatedAt string `json:"createdAt"`
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
