package apikey

import (
	"github.com/neo-vai/go-events/internal/model/apikey"
)

type APIKeyResponse struct {
	ID        string `json:"id"`
	Key       string `json:"key,omitempty"`       // only present on creation
	KeyPrefix string `json:"keyPrefix,omitempty"` // first 8 chars, always present
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
}

type UpdateAPIKeyActiveRequest struct {
	Active *bool `json:"active" binding:"required"`
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
