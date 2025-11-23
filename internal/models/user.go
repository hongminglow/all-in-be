package models

import "time"

// User captures application-facing fields for an authenticated identity.
type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	Phone        string    `json:"phone"`
	Role         string    `json:"role"`
	Permissions  []string  `json:"permissions"`
	Balance      float64   `json:"balance"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}
