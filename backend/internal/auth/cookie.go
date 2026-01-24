package auth

import (
	"net/http"
	"time"

	"todo-app-backend/internal/config"
)

const (
	AuthCookieName = "__Host-Session"
	// __Host- prefix requires: Secure flag, no Domain, Path=/
	// This prevents subdomain cookie attacks and MITM
)

// SetAuthCookie sets a secure, HttpOnly authentication cookie
// SECURITY: HttpOnly prevents JavaScript access (XSS protection)
// SECURITY: Secure flag requires HTTPS
// SECURITY: SameSite=Strict prevents CSRF
// SECURITY: __Host- prefix enforces security requirements
func SetAuthCookie(w http.ResponseWriter, token string) {
	expirationMinutes := config.GetJWTExpiration()

	cookie := &http.Cookie{
		Name:     AuthCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,  // JavaScript cannot access (XSS protection)
		Secure:   config.IsProductionMode(), // HTTPS only in production
		SameSite: http.SameSiteStrictMode,   // CSRF protection
		MaxAge:   int(expirationMinutes.Seconds()),
	}

	http.SetCookie(w, cookie)
}

// GetAuthCookie extracts JWT token from cookie
// Returns empty string if cookie not found or invalid
func GetAuthCookie(r *http.Request) string {
	cookie, err := r.Cookie(AuthCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

// ClearAuthCookie clears the authentication cookie
func ClearAuthCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     AuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   config.IsProductionMode(),
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1, // Delete immediately
		Expires:  time.Unix(0, 0),
	}

	http.SetCookie(w, cookie)
}
