package account

import "github.com/neo-vai/go-events/internal/model/account"

type CreateAccountRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,strongpassword"`
}

type UpdateAccountRequest struct {
	Email string `json:"email" binding:"omitempty,email"`
}

type AccountResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
}

func ToAccountResponse(acc *account.Account) AccountResponse {
	return AccountResponse{
		ID:        acc.ID.String(),
		Email:     acc.Email,
		Role:      acc.Role,
		CreatedAt: acc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
