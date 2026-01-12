package middleware

import (
	"net/http"

	"todo-app-backend/internal/utils"
)

var securityLogger = utils.NewLogger("SECURITY_MIDDLEWARE")

// SecurityHeadersMiddleware adds security headers not including CSP
func SecurityHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		next(w, r)
	}
}
