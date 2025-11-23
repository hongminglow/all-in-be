package server

import (
	"context"
	"net/http"
	"time"

	"github.com/hongminglow/all-in-be/internal/auth"
	"github.com/hongminglow/all-in-be/internal/config"
	"github.com/hongminglow/all-in-be/internal/http/handlers"
	"github.com/hongminglow/all-in-be/internal/middleware"
	"github.com/hongminglow/all-in-be/internal/storage"
)

// Server wraps an http.Server with configured routes.
type Server struct {
	inner *http.Server
}

// New wires up middleware, routes, and returns a ready server.
func New(cfg config.Config, store storage.UserStore) *Server {
	mux := http.NewServeMux()
	health := handlers.NewHealthHandler(time.Now())
	health.Register(mux)
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	auth := handlers.NewAuthHandler(store, tokenManager, &cfg)
	auth.Register(mux)

	handler := middleware.CORS(cfg.CORSOrigins, middleware.Logging(mux))

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddress(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	return &Server{inner: httpServer}
}

// Start begins serving HTTP traffic.
func (s *Server) Start() error {
	return s.inner.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.inner.Shutdown(ctx)
}
