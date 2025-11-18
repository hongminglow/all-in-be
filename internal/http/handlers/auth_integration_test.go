package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"github.com/hongminglow/all-in-be/internal/auth"
	"github.com/hongminglow/all-in-be/internal/models"
	"github.com/hongminglow/all-in-be/internal/storage/postgres"
)

// TestAuthIntegration exercises the register/login endpoints against the live Neon DB.
func TestAuthIntegration(t *testing.T) {
	if os.Getenv("RUN_AUTH_INTEGRATION") != "true" {
		t.Skip("set RUN_AUTH_INTEGRATION=true to run this integration test")
	}

	loadDotEnv()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	store, err := postgres.NewUserStore(ctx, dbURL)
	if err != nil {
		t.Fatalf("init store: %v", err)
	}
	defer store.Close()

	secret := mustGetEnv(t, "JWT_SECRET")
	issuer := mustGetEnv(t, "JWT_ISSUER")
	ttl := mustGetTTL(t)
	tokens := auth.NewTokenManager(secret, issuer, ttl)

	mux := http.NewServeMux()
	authHandler := NewAuthHandler(store, tokens)
	authHandler.Register(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	username := fmt.Sprintf("apitest_%d", time.Now().UnixNano())
	email := fmt.Sprintf("%s@example.com", username)
	phone := fmt.Sprintf("+1555%07d", time.Now().UnixNano()%1_000_0000)
	password := fmt.Sprintf("Pass!%d", time.Now().UnixNano())

	registerBody := map[string]string{
		"username": username,
		"email":    email,
		"phone":    phone,
		"password": password,
	}
	user := requestRegister(t, ts.URL, registerBody)

	if user.Username != username || user.Email != email || user.Phone != phone {
		t.Fatalf("register mismatch: got %+v", user)
	}

	loggedIn := requestLogin(t, ts.URL, username, password)
	if loggedIn.User.ID != user.ID {
		t.Fatalf("login returned wrong user id: want %d got %d", user.ID, loggedIn.User.ID)
	}
	if strings.TrimSpace(loggedIn.Token) == "" {
		t.Fatal("login response missing token")
	}

	t.Logf("created user %s (id=%d) and successfully logged in via /login", username, user.ID)
}

type loginResponseBody struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

func requestRegister(t *testing.T, baseURL string, payload map[string]string) models.User {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal register payload: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/register", baseURL), bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build register request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("register request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status = %d", resp.StatusCode)
	}

	var out models.User
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	return out
}

func requestLogin(t *testing.T, baseURL, identifier, password string) loginResponseBody {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"identifier": identifier,
		"password":   password,
	})
	if err != nil {
		t.Fatalf("marshal login payload: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/login", baseURL), bytes.NewReader(body))
	if err != nil {
		t.Fatalf("build login request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("login request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", resp.StatusCode)
	}

	var out loginResponseBody
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	return out
}

func mustGetEnv(t *testing.T, key string) string {
	t.Helper()
	val := strings.TrimSpace(os.Getenv(key))
	if val == "" {
		t.Fatalf("%s is required", key)
	}
	return val
}

func mustGetTTL(t *testing.T) time.Duration {
	t.Helper()
	minutesStr := mustGetEnv(t, "JWT_TTL_MINUTES")
	minutes, err := strconv.Atoi(minutesStr)
	if err != nil || minutes <= 0 {
		t.Fatalf("invalid JWT_TTL_MINUTES value: %q", minutesStr)
	}
	return time.Duration(minutes) * time.Minute
}

func loadDotEnv() {
	paths := []string{
		".env",
		"../.env",
		"../../.env",
		"../../../.env",
		"../../../../.env",
	}
	for _, path := range paths {
		_ = godotenv.Overload(path)
	}
}
