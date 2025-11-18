package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds runtime configuration sourced from env vars.
type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	JWTIssuer   string
	JWTTTL      time.Duration
	CORSOrigins []string
}

// Load reads configuration from the environment and performs minimal validation.
func Load() (Config, error) {
	cfg := Config{
		Port:        fallback(os.Getenv("PORT"), "8080"),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
		JWTSecret:   strings.TrimSpace(os.Getenv("JWT_SECRET")),
		JWTIssuer:   fallback(os.Getenv("JWT_ISSUER"), "all-in-backend"),
		CORSOrigins: parseCSV(fallback(os.Getenv("CORS_ALLOWED_ORIGINS"), "*")),
	}

	minutes := fallback(os.Getenv("JWT_TTL_MINUTES"), "60")
	if ttlMinutes, err := strconv.Atoi(minutes); err == nil && ttlMinutes > 0 {
		cfg.JWTTTL = time.Duration(ttlMinutes) * time.Minute
	} else {
		cfg.JWTTTL = 60 * time.Minute
	}

	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}

	return cfg, nil
}

// HTTPAddress returns the host:port pair for the HTTP server to bind to.
func (c Config) HTTPAddress() string {
	return fmt.Sprintf(":%s", c.Port)
}

func fallback(value, def string) string {
	if strings.TrimSpace(value) == "" {
		return def
	}
	return strings.TrimSpace(value)
}

func parseCSV(input string) []string {
	parts := strings.Split(input, ",")
	var out []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
