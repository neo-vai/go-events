package event

import "github.com/neo-vai/go-events/internal/model/event"

type CreateEventRequest struct {
	Username string `json:"user" binding:"required"`
	APIKeyID string `json:"apiKeyID"`
	Name     string `json:"name" binding:"required"`
	Payload  string `json:"payload"`
}

type EventResponse struct {
	ID        string `json:"id"`
	AccountID string `json:"accountID"`
	Username  string `json:"user"`
	APIKeyID  string `json:"apiKeyID"`
	Name      string `json:"name"`
	Payload   string `json:"payload"`
	CreatedAt string `json:"createdAt"`
}

func ToEventResponse(ev *event.Event) EventResponse {
	return EventResponse{
		ID:        ev.ID,
		AccountID: ev.AccountID,
		Username:  ev.Username,
		APIKeyID:  ev.APIKeyID,
		Name:      ev.Name,
		Payload:   ev.Payload,
		CreatedAt: ev.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
