package handlers

import (
	"net/http"

	"todo-app-backend/internal/auth"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/utils"
)

var logoutLogger = utils.NewLogger("LOGOUT")

// LogoutHandler clears the authentication cookie
// SECURITY: Client-side logout (server-side JWT invalidation not implemented - stateless design)
// For full server-side logout, implement JWT blacklist (Redis-based)
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	logoutLogger.Debug("LogoutHandler: processing logout request from %s", r.RemoteAddr)

	// Clear the HttpOnly authentication cookie
	auth.ClearAuthCookie(w)
	logoutLogger.Debug("LogoutHandler: cleared authentication cookie")

	utils.EncodeJSONResponse(w, models.LoginResponse{
		Success: true,
		Message: "Logged out successfully",
	}, http.StatusOK)

	logoutLogger.LogResponse(http.StatusOK, "Logout successful")
}
