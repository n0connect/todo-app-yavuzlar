package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"todo-app-backend/internal/config"
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

		if config.IsProductionMode() && IsRequestHTTPS(r) {
			maxAge := config.GetHSTSMaxAge()
			if maxAge > 0 {
				hsts := []string{"max-age=" + strconv.Itoa(maxAge)}
				if config.GetHSTSIncludeSubDomains() {
					hsts = append(hsts, "includeSubDomains")
				}
				if config.GetHSTSPreload() {
					hsts = append(hsts, "preload")
				}
				w.Header().Set("Strict-Transport-Security", strings.Join(hsts, "; "))
			}
		}

		if coop := config.GetSecurityHeaderCOOP(); coop != "" {
			w.Header().Set("Cross-Origin-Opener-Policy", coop)
		}
		if corp := config.GetSecurityHeaderCORP(); corp != "" {
			w.Header().Set("Cross-Origin-Resource-Policy", corp)
		}
		if coep := config.GetSecurityHeaderCOEP(); coep != "" {
			w.Header().Set("Cross-Origin-Embedder-Policy", coep)
		}

		next(w, r)
	}
}
