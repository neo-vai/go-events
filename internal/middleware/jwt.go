package middleware

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	AccountID string `json:"account_id"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token using the provided secret and expiration hours.
func GenerateJWT(accountID, role, jwtSecret string, expiresHours int) (string, error) {
	claims := Claims{
		AccountID: accountID,
		Role:      role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresHours) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// ValidateJWT parses and validates the token using the provided secret.
func ValidateJWT(tokenString, jwtSecret string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}
	return claims, nil
}
