package dto

import "github.com/hongminglow/all-in-be/internal/models"

type RegisterRequest struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	PhoneNumber string `json:"phoneNumber"`
	Password    string `json:"password"`
}

type LoginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type LoginResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}
