package middleware

import (
	"net/http"
	"strings"
)

// CORS adds Access-Control headers for allowed origins and short-circuits OPTIONS requests.
func CORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowAll := false
	normalized := make([]string, 0, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAll = true
			break
		}
		normalized = append(normalized, strings.ToLower(origin))
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if allowAll || containsOrigin(normalized, origin) {
				if allowAll {
					w.Header().Set("Access-Control-Allow-Origin", "*")
				} else {
					w.Header().Set("Access-Control-Allow-Origin", origin)
				}
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func containsOrigin(allowed []string, origin string) bool {
	origin = strings.ToLower(origin)
	for _, candidate := range allowed {
		if candidate == origin {
			return true
		}
	}
	return false
}
