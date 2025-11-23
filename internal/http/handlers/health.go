package handlers

import (
	"net/http"
	"time"

	"github.com/hongminglow/all-in-be/internal/http/respond"
)

// HealthHandler returns uptime and basic status.
type HealthHandler struct {
	startedAt time.Time
}

// NewHealthHandler creates a health endpoint handler.
func NewHealthHandler(startedAt time.Time) *HealthHandler {
	return &HealthHandler{startedAt: startedAt}
}

// Register wires the handler into a ServeMux.
func (h *HealthHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.handle)
}

func (h *HealthHandler) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	respond.JSON(w, http.StatusOK, "service healthy", map[string]string{
		"status": "ok",
		"uptime": time.Since(h.startedAt).Truncate(time.Second).String(),
	})
}
