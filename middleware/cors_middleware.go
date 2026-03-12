package middleware

import "net/http"

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS) setup.
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Browser requirement: When credentials are included, Origin cannot be "*"
		// We use the request's Origin header to specify the allowed origin
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		// Allow credentials (cookies, authorization headers, etc.)
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Allow specific HTTP methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// Allow specific headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		// Preflight cache duration
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight (OPTIONS) requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Proceed to the next handler
		next.ServeHTTP(w, r)
	})
}
