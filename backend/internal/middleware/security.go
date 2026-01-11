package middleware

import (
	"net/http"

	"todo-app-backend/internal/utils"
)

var securityLogger = utils.NewLogger("SECURITY_MIDDLEWARE")

// SecurityHeadersMiddleware adds security headers including CSP
func SecurityHeadersMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		securityLogger.Debug("SecurityHeadersMiddleware: processing request: method=%s path=%s", r.Method, r.URL.Path)

		// Content Security Policy - strict mode
		csp := "default-src 'self'; " +
			"script-src 'self'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"style-src-elem 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"img-src 'self' data:; " +
			"font-src 'self' https://fonts.gstatic.com; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'; " +
			"object-src 'none'; " +
			"upgrade-insecure-requests;"

		w.Header().Set("Content-Security-Policy", csp)
		securityLogger.Debug("SecurityHeadersMiddleware: set Content-Security-Policy header")

		// X-Content-Type-Options: Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// X-Frame-Options: Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// X-XSS-Protection: Legacy XSS protection (modern browsers ignore but good for older ones)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer-Policy: Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy: Restrict browser features
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		securityLogger.Debug("SecurityHeadersMiddleware: all security headers set for path: %s", r.URL.Path)

		next(w, r)
	}
}
