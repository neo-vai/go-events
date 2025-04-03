package apikey

import (
	"github.com/neo-vai/go-events/internal/model/apikey"
)

// APIKeyResponse represents an API key in responses.
// The full key is only included upon creation.
type APIKeyResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Key       string `json:"key,omitempty" example:"dGhpcy1pcy1hLWZha2UtYXBpLWtleQ=="`
	KeyPrefix string `json:"keyPrefix,omitempty" example:"dGhpcy1p..."`
	Active    bool   `json:"active" example:"true"`
	CreatedAt string `json:"createdAt" example:"2023-01-01T12:00:00Z"`
}

// UpdateAPIKeyActiveRequest is used to change the active status of an API key.
type UpdateAPIKeyActiveRequest struct {
	Active *bool `json:"active" binding:"required" example:"true"`
}

func ToAPIKeyResponse(key *apikey.APIKey, includeFullKey bool) APIKeyResponse {
	resp := APIKeyResponse{
		ID:        key.ID.String(),
		Active:    key.Active,
		CreatedAt: key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if includeFullKey && key.PlainKey != "" {
		resp.Key = key.PlainKey
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
