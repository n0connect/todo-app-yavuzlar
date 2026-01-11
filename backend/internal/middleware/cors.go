package middleware

import (
	"net/http"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/utils"
)

var allowedOrigin string
var corsLogger = utils.NewLogger("CORS")

func init() {
	allowedOrigin = config.GetEnv("ALLOWED_ORIGIN", "*")
	corsLogger.Info("CORS middleware initialized with allowed origin: %s", allowedOrigin)
}

func CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		corsLogger.Debug("CORSMiddleware: processing request: method=%s path=%s origin=%s", r.Method, r.URL.Path, origin)

		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Expose-Headers", "Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")
		corsLogger.Debug("CORSMiddleware: CORS headers set: allowedOrigin=%s", allowedOrigin)

		if r.Method == http.MethodOptions {
			corsLogger.Debug("CORSMiddleware: handling OPTIONS preflight request for path: %s", r.URL.Path)
			w.WriteHeader(http.StatusOK)
			corsLogger.Debug("CORSMiddleware: OPTIONS preflight completed for path: %s", r.URL.Path)
			return
		}

		corsLogger.Debug("CORSMiddleware: CORS check passed, forwarding request: method=%s path=%s", r.Method, r.URL.Path)
		next(w, r)
	}
}
