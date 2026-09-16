package middleware

import (
	"net/http"
	"strings"
)

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS) with an explicit allowlist.
// It strictly forbids arbitrary origin reflection and wildcard credentials.
func CORSMiddleware(next http.Handler, allowedOriginsCSV string) http.Handler {
	allowedMap := make(map[string]bool)

	// Parse comma-separated allowed origins (from config/env)
	for _, o := range strings.Split(allowedOriginsCSV, ",") {
		if trimmed := strings.TrimSpace(o); trimmed != "" {
			allowedMap[trimmed] = true
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Only set CORS headers if origin is explicitly in the allowlist
		if origin != "" && allowedMap[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Internal-Service-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
