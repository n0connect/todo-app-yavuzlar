package middleware

import (
	"net/http"
	"strings"

	"todo-app-backend/internal/auth"
	"todo-app-backend/internal/store"
	"todo-app-backend/internal/utils"
)

var (
	authMiddlewareLogger = utils.NewLogger("AUTH_MIDDLEWARE")
	userRepository       = store.NewUserRepository()
)

// JWTMiddleware verifies JWT token and adds user UUID to request context
func JWTMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authMiddlewareLogger.Debug("JWTMiddleware: checking authentication for path: %s", r.URL.Path)

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			authMiddlewareLogger.Warn("JWTMiddleware: missing Authorization header for path: %s", r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Check Bearer prefix
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			authMiddlewareLogger.Warn("JWTMiddleware: invalid Authorization header format for path: %s", r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		authMiddlewareLogger.Debug("JWTMiddleware: extracted token from Authorization header")

		// Verify token
		userUUID, err := auth.VerifyToken(tokenString)
		if err != nil {
			if err == auth.ErrExpiredToken {
				authMiddlewareLogger.Warn("JWTMiddleware: expired token for path: %s", r.URL.Path)
			} else {
				authMiddlewareLogger.Warn("JWTMiddleware: invalid token for path: %s", r.URL.Path)
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		authMiddlewareLogger.Debug("JWTMiddleware: token verified successfully: UUID=%s path=%s", userUUID, r.URL.Path)

		// Verify user exists in database
		_, err = userRepository.FindByUUID(userUUID)
		if err != nil {
			authMiddlewareLogger.LogError("UserRepository.FindByUUID", err)
			authMiddlewareLogger.Warn("JWTMiddleware: user not found for UUID: %s path: %s", userUUID, r.URL.Path)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		authMiddlewareLogger.Debug("JWTMiddleware: user verified in database: UUID=%s path=%s", userUUID, r.URL.Path)

		// Add user UUID to context
		ctx := auth.WithUserUUID(r.Context(), userUUID)
		next(w, r.WithContext(ctx))
	}
}
