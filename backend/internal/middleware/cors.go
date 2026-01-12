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

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		corsLogger.Debug("CORSMiddleware: method=%s path=%s origin=%s", r.Method, r.URL.Path, origin)

		if origin != "" {
			if _, ok := allowedOrigins[origin]; ok {
				// Echo origin
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")

				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Expose-Headers", "Authorization")
				w.Header().Set("Access-Control-Max-Age", "3600")
			} else {
				// If Preflight then ret
				if r.Method == http.MethodOptions {
					http.Error(w, "CORS origin not allowed", http.StatusForbidden)
					return
				}
				// Dont send headers any other requests.
			}
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent) // 204
			return
		}

		next(w, r)
	}
}
