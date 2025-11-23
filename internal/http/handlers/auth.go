package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"github.com/hongminglow/all-in-be/internal/auth"
	"github.com/hongminglow/all-in-be/internal/config"
	"github.com/hongminglow/all-in-be/internal/http/respond"
	"github.com/hongminglow/all-in-be/internal/models"
	"github.com/hongminglow/all-in-be/internal/models/dto"
	"github.com/hongminglow/all-in-be/internal/storage"
)

// AuthHandler owns register/login endpoints backed by Neon Auth & Postgres.
type AuthHandler struct {
	store  storage.UserStore
	tokens *auth.TokenManager
	cfg    *config.Config
}

// NewAuthHandler constructs the handler.
func NewAuthHandler(store storage.UserStore, tokens *auth.TokenManager, cfg *config.Config) *AuthHandler {
	return &AuthHandler{store: store, tokens: tokens, cfg: cfg}
}

// Register attaches auth routes to the mux.
func (h *AuthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/register", h.handleRegister)
	mux.HandleFunc("/login", h.handleLogin)
}

func (h *AuthHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	phone := normalizePhone(req)
	if err := validateCredentials(req.Username, req.Email, phone, req.Password); err != nil {
		respond.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user := models.User{
		Username:     strings.TrimSpace(req.Username),
		Email:        strings.TrimSpace(req.Email),
		Phone:        phone,
		Role:         models.NormalUser,
		Balance:      h.cfg.InitBalance,
		PasswordHash: passwordHash,
	}
	created, err := h.store.CreateUser(r.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrAlreadyExists):
			respond.Error(w, http.StatusConflict, "user already exists")
		default:
			log.Printf("create user error: %v", err)
			respond.Error(w, http.StatusInternalServerError, "failed to create user")
		}
		return
	}

	respond.JSON(w, http.StatusOK, "User created successfully", created)
}

func (h *AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if strings.TrimSpace(req.Identifier) == "" || strings.TrimSpace(req.Password) == "" {
		respond.Error(w, http.StatusBadRequest, "identifier and password are required")
		return
	}
	user, err := h.store.FindByUsernameOrEmail(r.Context(), strings.TrimSpace(req.Identifier))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// Log the error even for not found to help debug if it's a join failure
			log.Printf("login failed: user not found or join failed for identifier %s: %v", req.Identifier, err)
			respond.Error(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		log.Printf("login failed: error fetching user %s: %v", req.Identifier, err)
		respond.Error(w, http.StatusInternalServerError, "failed to fetch user")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respond.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	token, err := h.tokens.Generate(user)
	if err != nil {
		respond.Error(w, http.StatusInternalServerError, "failed to generate token")
		return
	}
	respond.JSON(w, http.StatusOK, "login successful", dto.LoginResponse{Token: token, User: user})
}

func normalizePhone(req dto.RegisterRequest) string {
	if trimmed := strings.TrimSpace(req.Phone); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(req.PhoneNumber)
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
