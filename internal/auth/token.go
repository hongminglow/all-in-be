package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/hongminglow/all-in-be/internal/models"
)

// TokenManager issues signed JWTs for authenticated users.
type TokenManager struct {
	secret []byte
	issuer string
	ttl    time.Duration
}

// NewTokenManager creates a manager with the provided secret, issuer, and lifetime.
func NewTokenManager(secret, issuer string, ttl time.Duration) *TokenManager {
	return &TokenManager{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    ttl,
	}
}

// Generate issues a signed JWT string for the provided user ID.
func (t *TokenManager) Generate(user models.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":      t.issuer,
		"sub":      fmt.Sprintf("%d", user.ID),
		"username": user.Username,
		"email":    user.Email,
		"iat":      now.Unix(),
		"nbf":      now.Unix(),
		"exp":      now.Add(t.ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(t.secret)
}
