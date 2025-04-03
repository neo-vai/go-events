package admin

import (
	"time"

	"github.com/neo-vai/go-events/internal/model/account"
	"github.com/neo-vai/go-events/internal/model/apikey"
	"github.com/neo-vai/go-events/internal/model/event"
)

// Account DTOs
type AccountResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
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

type CreateAccountRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,strongpassword"`
	Role     string `json:"role" binding:"required,oneof=user admin"`
}

type UpdateAccountRequest struct {
	Email  string `json:"email,omitempty" binding:"omitempty,email"`
	Role   string `json:"role,omitempty" binding:"omitempty,oneof=user admin"`
	Active *bool  `json:"active,omitempty"`
}

// Event DTOs
type EventResponse struct {
	ID        string `json:"id"`
	AccountID string `json:"accountId"`
	Username  string `json:"username"`
	APIKeyID  string `json:"apiKeyId"`
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
		CreatedAt: ev.CreatedAt.Format(time.RFC3339),
	}
}

// APIKey DTOs
type APIKeyResponse struct {
	ID          string `json:"id"`
	AccountID   string `json:"accountId"`
	AccountName string `json:"accountName,omitempty"`
	KeyPrefix   string `json:"keyPrefix,omitempty"` // first 8 chars
	Active      bool   `json:"active"`
	CreatedAt   string `json:"createdAt"`
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

type UpdateAPIKeyRequest struct {
	Active *bool `json:"active" binding:"required"`
}

// Stats DTO
type StatsResponse struct {
	TotalAccounts  int64 `json:"totalAccounts"`
	ActiveAccounts int64 `json:"activeAccounts"`
	TotalEvents    int64 `json:"totalEvents"`
	EventsToday    int64 `json:"eventsToday"`
	TotalAPIKeys   int64 `json:"totalApiKeys"`
	ActiveAPIKeys  int64 `json:"activeApiKeys"`
}
