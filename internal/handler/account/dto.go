package account

import "github.com/neo-vai/go-events/internal/model/account"

type CreateAccountRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type UpdateAccountRequest struct {
	Name  string `json:"name"`
	Email string `json:"email" binding:"omitempty,email"`
	Login string `json:"login"`
}

type AccountResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Login     string `json:"login"`
	CreatedAt string `json:"createdAt"`
}

func ToAccountResponse(acc *account.Account) AccountResponse {
	return AccountResponse{
		ID:        acc.ID,
		Name:      acc.Name,
		Email:     acc.Email,
		Login:     acc.Login,
		CreatedAt: acc.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
