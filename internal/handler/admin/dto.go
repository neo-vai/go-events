package admin

import (
	"time"

	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/neo-vai/go-events/internal/model/event"
)

// AccountResponse is the admin view of an account.
type AccountResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string `json:"email" example:"user@example.com"`
	Role      string `json:"role" example:"admin"`
	Active    bool   `json:"active" example:"true"`
	CreatedAt string `json:"createdAt" example:"2023-01-01T12:00:00Z"`
}

func ToAccountResponse(acc *account.Account) AccountResponse {
	return AccountResponse{
		ID:        acc.ID.String(),
		Email:     acc.Email,
		Role:      acc.Role,
		Active:    acc.Active,
		CreatedAt: acc.CreatedAt.Format(time.RFC3339),
	}
}

// CreateAccountRequest is used by admins to create accounts.
type CreateAccountRequest struct {
	Email    string `json:"email" binding:"required,email" example:"newuser@example.com"`
	Password string `json:"password" binding:"required,strongpassword" example:"Passw0rd!"`
	Role     string `json:"role" binding:"required,oneof=user admin" example:"user"`
}

// UpdateAccountRequest allows admins to modify account fields.
type UpdateAccountRequest struct {
	Email  string `json:"email,omitempty" binding:"omitempty,email" example:"updated@example.com"`
	Role   string `json:"role,omitempty" binding:"omitempty,oneof=user admin" example:"admin"`
	Active *bool  `json:"active,omitempty" example:"false"`
}

// EventResponse is the admin view of an event.
type EventResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountID string `json:"accountId" example:"550e8400-e29b-41d4-a716-446655440001"`
	Username  string `json:"username" example:"john_doe"`
	APIKeyID  string `json:"apiKeyId" example:"550e8400-e29b-41d4-a716-446655440002"`
	Name      string `json:"name" example:"user_login"`
	Payload   string `json:"payload" example:"{\"ip\":\"192.168.1.1\"}"`
	CreatedAt string `json:"createdAt" example:"2023-01-01T12:00:00Z"`
}

func ToEventResponse(ev *event.Event) EventResponse {
	return EventResponse{
		ID:        ev.ID,
		AccountID: ev.AccountID,
		Username:  ev.Username,
		APIKeyID:  ev.APIKeyID,
		Name:      ev.Name,
		Payload:   ev.Payload,
		CreatedAt: ev.CreatedAt.Format(time.RFC3339),
	}
}

// APIKeyResponse is the admin view of an API key.
type APIKeyResponse struct {
	ID          string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	AccountID   string `json:"accountId" example:"550e8400-e29b-41d4-a716-446655440001"`
	AccountName string `json:"accountName,omitempty" example:"user@example.com"`
	KeyPrefix   string `json:"keyPrefix,omitempty" example:"dGhpcy1p..."`
	Active      bool   `json:"active" example:"true"`
	CreatedAt   string `json:"createdAt" example:"2023-01-01T12:00:00Z"`
}

func ToAPIKeyResponse(key *apikey.APIKey, accountName string) APIKeyResponse {
	resp := APIKeyResponse{
		ID:          key.ID.String(),
		AccountID:   key.AccountID.String(),
		AccountName: accountName,
		Active:      key.Active,
		CreatedAt:   key.CreatedAt.Format(time.RFC3339),
	}
	if key.PlainKey != "" {
		if len(key.PlainKey) >= 8 {
			resp.KeyPrefix = key.PlainKey[:8] + "..."
		} else {
			resp.KeyPrefix = key.PlainKey + "..."
		}
	}
	return resp
}

// UpdateAPIKeyRequest is used to change active status.
type UpdateAPIKeyRequest struct {
	Active *bool `json:"active" binding:"required" example:"false"`
}

// StatsResponse contains system statistics.
type StatsResponse struct {
	TotalAccounts  int64 `json:"totalAccounts" example:"42"`
	ActiveAccounts int64 `json:"activeAccounts" example:"38"`
	TotalEvents    int64 `json:"totalEvents" example:"15243"`
	EventsToday    int64 `json:"eventsToday" example:"512"`
	TotalAPIKeys   int64 `json:"totalApiKeys" example:"75"`
	ActiveAPIKeys  int64 `json:"activeApiKeys" example:"60"`
}
