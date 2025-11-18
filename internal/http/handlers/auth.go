package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"github.com/hongminglow/all-in-be/internal/auth"
	"github.com/hongminglow/all-in-be/internal/models"
	"github.com/hongminglow/all-in-be/internal/storage"
)

// AuthHandler owns register/login endpoints backed by Neon Auth & Postgres.
type AuthHandler struct {
	store  storage.UserStore
	tokens *auth.TokenManager
}

// NewAuthHandler constructs the handler.
func NewAuthHandler(store storage.UserStore, tokens *auth.TokenManager) *AuthHandler {
	return &AuthHandler{store: store, tokens: tokens}
}

// Register attaches auth routes to the mux.
func (h *AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/register", h.handleRegister)
	mux.HandleFunc("/login", h.handleLogin)
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if err := validateCredentials(req.Username, req.Email, req.Phone, req.Password); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := models.User{
		Username:     strings.TrimSpace(req.Username),
		Email:        strings.TrimSpace(req.Email),
		Phone:        strings.TrimSpace(req.Phone),
		PasswordHash: passwordHash,
	}
	created, err := h.store.CreateUser(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrAlreadyExists):
			respondError(w, http.StatusConflict, "user already exists")
		default:
			respondError(w, http.StatusInternalServerError, "failed to create user")
		}
		return
	}

	respondJSON(w, http.StatusCreated, created)
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(req.Identifier) == "" || strings.TrimSpace(req.Password) == "" {
		respondError(w, http.StatusBadRequest, "identifier and password are required")
		return
	}
	user, err := h.store.FindByUsernameOrEmail(r.Context(), strings.TrimSpace(req.Identifier))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := h.tokens.Generate(user)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	respondJSON(w, http.StatusOK, loginResponse{Token: token, User: user})
}

func validateCredentials(username, email, phone, password string) error {
	if strings.TrimSpace(username) == "" || strings.TrimSpace(email) == "" || strings.TrimSpace(phone) == "" {
		return errors.New("username, email, and phone are required")
	}
	if len(strings.TrimSpace(password)) < 8 || !utf8.ValidString(password) {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
