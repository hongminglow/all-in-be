package respond

import (
	"encoding/json"
	"log"
	"net/http"
)

// Envelope is the standard API response wrapper used across handlers.
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// JSON writes a success or informational response using the common envelope.
func JSON(w http.ResponseWriter, status int, message string, data any) {
	write(w, status, Envelope{Code: status, Message: message, Data: data})
}

// Error writes an error response with the shared envelope structure.
func Error(w http.ResponseWriter, status int, message string) {
	write(w, status, Envelope{Code: status, Message: message})
}

func write(w http.ResponseWriter, status int, payload Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("respond: encode payload failed: %v", err)
	}
}
