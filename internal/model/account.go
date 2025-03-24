package model

import "time"

type Account struct {
	ID           string
	Name         string
	Email        string
	Login        string
	PasswordHash string
	CreatedAt    time.Time
	APIKeys      []APIKey
}
