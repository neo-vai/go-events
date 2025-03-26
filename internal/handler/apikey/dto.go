package apikey

import (
	"github.com/neo-vai/go-events/internal/model/apikey"
)

type APIKeyResponse struct {
	ID        string `json:"id"`
	AccountID string `json:"accountID"`
	Key       string `json:"key"`
	Active    bool   `json:"active"`
	CreatedAt string `json:"createdAt"`
}

type UpdateAPIKeyActiveRequest struct {
	Active *bool `json:"active" binding:"required"`
}

func ToAPIKeyResponse(key *apikey.APIKey) APIKeyResponse {
	return APIKeyResponse{
		ID:        key.ID.String(),
		AccountID: key.AccountID.String(),
		Key:       key.Key,
		Active:    key.Active,
		CreatedAt: key.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
