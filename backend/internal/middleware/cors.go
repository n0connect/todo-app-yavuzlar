package middleware

import (
	"net/http"
	"strings"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/utils"
)

var allowedOrigins map[string]struct{}
var corsLogger = utils.NewLogger("CORS")

func init() {
	allowedOrigins = make(map[string]struct{})

	raw := config.GetEnv("ALLOWED_ORIGINS", "")
	if raw == "" {
		allowedOrigins["http://localhost"] = struct{}{}
		allowedOrigins["http://127.0.0.1"] = struct{}{}
		corsLogger.Info("CORS: default localhost whitelist enabled")
		return
	}

	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		allowedOrigins[o] = struct{}{}
	}
	corsLogger.Info("CORS: whitelist enabled from ALLOWED_ORIGINS, count=%d", len(allowedOrigins))
}

// isOriginAllowed checks if an origin is allowed, with flexible matching for localhost
func isOriginAllowed(origin string) bool {
	// Exact match first
	if _, ok := allowedOrigins[origin]; ok {
		return true
	}

	// Flexible matching for localhost: allow any port for localhost origins
	if strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") {
		// Check if base localhost is in allowed origins
		if _, ok := allowedOrigins["http://localhost"]; ok && strings.HasPrefix(origin, "http://localhost:") {
			return true
		}
		if _, ok := allowedOrigins["http://127.0.0.1"]; ok && strings.HasPrefix(origin, "http://127.0.0.1:") {
			return true
		}
	}

	return false
}

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		corsLogger.Debug("CORSMiddleware: method=%s path=%s origin=%s", r.Method, r.URL.Path, origin)

		if origin != "" {
			if isOriginAllowed(origin) {
				// Echo origin
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")

				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Expose-Headers", "Authorization")
				w.Header().Set("Access-Control-Max-Age", "3600")
			} else {
				// If Preflight then return error
				if r.Method == http.MethodOptions {
					http.Error(w, "CORS origin not allowed", http.StatusForbidden)
					return
				}
				// Don't send headers for any other requests.
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // 204
			return
		}

		next(w, r)
	}
}
