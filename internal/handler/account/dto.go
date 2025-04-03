package account

import "github.com/neo-vai/go-events/internal/model/account"

// CreateAccountRequest represents the payload for creating a new account.
type CreateAccountRequest struct {
	Email    string `json:"email" binding:"required,email" example:"user@example.com"`
	Password string `json:"password" binding:"required,strongpassword" example:"Passw0rd!"`
}

// UpdateAccountRequest represents the payload for updating the current account.
type UpdateAccountRequest struct {
	Email string `json:"email" binding:"omitempty,email" example:"newemail@example.com"`
}

// AccountResponse is the public representation of an account.
type AccountResponse struct {
	ID        string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email     string `json:"email" example:"user@example.com"`
	Role      string `json:"role" example:"user"`
	CreatedAt string `json:"createdAt" example:"2023-01-01T12:00:00Z"`
}

func ToAccountResponse(acc *account.Account) AccountResponse {
	return AccountResponse{
		ID:        acc.ID.String(),
		Email:     acc.Email,
		Role:      acc.Role,
		CreatedAt: acc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
