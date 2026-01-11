package handlers

import (
	"encoding/hex"
	"net/http"

	"todo-app-backend/internal/auth"
	"todo-app-backend/internal/encryption"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/store"
	"todo-app-backend/internal/utils"
)

var (
	authLogger     = utils.NewLogger("AUTH")
	userRepository = store.NewUserRepository()
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	authLogger.LogRequest(r.Method, r.URL.Path, "N/A")
	authLogger.Debug("Starting RegisterHandler")

	// Handle HEAD requests (used by browsers/tools for link previews, health checks, etc.)
	if r.Method == http.MethodHead {
		authLogger.Debug("Handling HEAD request for registration endpoint")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Method != http.MethodPost {
		authLogger.Warn("Invalid method for registration: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req models.RegisterRequest
	if err := utils.DecodeJSONRequest(r, &req); err != nil {
		// Empty body is OK - treat as generate UUID only (confirm=false)
		authLogger.Debug("Empty or invalid body, treating as UUID generation request")
		req.Confirm = false
	}

	// PHASE 1: Generate UUID only (confirm=false)
	// Returns a pending registration token that must be used to confirm
	if !req.Confirm {
		authLogger.Debug("Phase 1: Generating UUID preview (no account creation)")

		generatedUUID, err := utils.GenerateSecureUUID()
		if err != nil {
			authLogger.LogError("GenerateSecureUUID", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Check for collision - regenerate if needed
		exists, err := userRepository.ExistsByUUID(generatedUUID)
		if err != nil {
			authLogger.LogError("ExistsByUUID", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if exists {
			generatedUUID, err = utils.GenerateSecureUUID()
			if err != nil {
				authLogger.LogError("GenerateSecureUUID (retry)", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
		}

		// Generate pending registration token (contains the UUID, short-lived)
		pendingToken, err := auth.SignPendingRegistrationToken(generatedUUID)
		if err != nil {
			authLogger.LogError("SignPendingRegistrationToken", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		authLogger.Debug("Generated preview UUID: %s with pending token", generatedUUID)
		responseData := models.RegisterResponse{
			Success:      true,
			Message:      "UUID generated - click continue to create account",
			UUID:         generatedUUID,
			PendingToken: pendingToken,
			Confirmed:    false,
		}
		utils.EncodeJSONResponse(w, responseData, http.StatusOK)
		authLogger.LogResponse(http.StatusOK, "UUID preview generated with pending token")
		return
	}

	// PHASE 2: Create account (confirm=true)
	// ZERO-TRUST: Verify pending token and extract UUID from it
	authLogger.Debug("Phase 2: Confirming account creation")

	if req.PendingToken == "" {
		authLogger.Warn("No pending token provided for account confirmation")
		http.Error(w, "Pending token required", http.StatusBadRequest)
		return
	}

	// Verify pending token and extract the UUID from it
	pendingUUID, err := auth.VerifyPendingRegistrationToken(req.PendingToken)
	if err != nil {
		if err == auth.ErrExpiredToken {
			authLogger.Warn("Pending registration token expired")
			http.Error(w, "Registration expired, please try again", http.StatusUnauthorized)
			return
		}
		authLogger.Warn("Invalid pending registration token")
		http.Error(w, "Invalid registration token", http.StatusUnauthorized)
		return
	}

	// ZERO-TRUST: UUID comes from verified token, NOT from user input
	authLogger.Debug("Phase 2: Creating account with verified UUID: %s", pendingUUID)

	// Check if UUID already exists (race condition or replay attack)
	exists, err := userRepository.ExistsByUUID(pendingUUID)
	if err != nil {
		authLogger.LogError("ExistsByUUID", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if exists {
		authLogger.Warn("UUID already exists (possible replay): %s", pendingUUID)
		http.Error(w, "Account already created", http.StatusConflict)
		return
	}

	// Generate encryption key for user
	authLogger.Debug("Generating user AES key for UUID: %s", pendingUUID)
	userAESKey, err := encryption.GenerateUserAESKey()
	if err != nil {
		authLogger.LogError("GenerateUserAESKey", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	userKeyHex := hex.EncodeToString(userAESKey)
	encryptedUserKey, err := encryption.EncryptWithMasterKey(userKeyHex)
	if err != nil {
		authLogger.LogError("EncryptWithMasterKey", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	user := models.User{
		UUID:         pendingUUID,
		EncryptedKey: encryptedUserKey,
	}

	if err := userRepository.Create(&user); err != nil {
		authLogger.LogError("UserRepository.Create", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	authLogger.Info("Successfully registered new user: UUID=%s", pendingUUID)

	// Generate JWT token for immediate login
	token, err := auth.SignToken(pendingUUID)
	if err != nil {
		authLogger.LogError("SignToken", err)
		// Account created but token failed - user can still login manually
		responseData := models.RegisterResponse{
			Success:   true,
			Message:   "Account created successfully",
			UUID:      pendingUUID,
			Confirmed: true,
		}
		utils.EncodeJSONResponse(w, responseData, http.StatusCreated)
		return
	}

	responseData := models.RegisterResponse{
		Success:   true,
		Message:   "Account created successfully",
		UUID:      pendingUUID,
		Token:     token,
		Confirmed: true,
	}
	utils.EncodeJSONResponse(w, responseData, http.StatusCreated)
	authLogger.LogResponse(http.StatusCreated, "Account created successfully")
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	authLogger.LogRequest(r.Method, r.URL.Path, "N/A")
	authLogger.Debug("Starting LoginHandler")

	// Handle HEAD requests (used by browsers/tools for link previews, health checks, etc.)
	if r.Method == http.MethodHead {
		authLogger.Debug("Handling HEAD request for login endpoint")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Method != http.MethodPost {
		authLogger.Warn("Invalid method for login: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.LoginRequest
	if err := utils.DecodeJSONRequest(r, &req); err != nil {
		authLogger.LogError("DecodeJSONRequest", err)
		authLogger.Warn("Invalid JSON request for login")
		utils.EncodeJSONResponse(w, models.LoginResponse{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
		return
	}
	authLogger.Debug("Successfully decoded login request, UUID length: %d", len(req.UUID))

	if !utils.ValidateUUID(req.UUID) {
		authLogger.Warn("Invalid UUID format in login request: %s", req.UUID)
		utils.EncodeJSONResponse(w, models.LoginResponse{Success: false, Message: "Invalid UUID format"}, http.StatusBadRequest)
		return
	}
	authLogger.Debug("UUID format validated: %s", req.UUID)

	user, err := userRepository.FindByUUID(req.UUID)
	if err != nil {
		authLogger.LogError("UserRepository.FindByUUID", err)
		authLogger.Warn("Login failed: user not found for UUID: %s", req.UUID)
		utils.EncodeJSONResponse(w, models.LoginResponse{Success: false, Message: "Invalid credentials"}, http.StatusUnauthorized)
		return
	}
	authLogger.Info("Successful login for UUID: %s", req.UUID)

	// Generate JWT token
	authLogger.Debug("Generating JWT token for UUID: %s", user.UUID)
	token, err := auth.SignToken(user.UUID)
	if err != nil {
		authLogger.LogError("SignToken", err)
		authLogger.Error("Failed to generate JWT token for UUID: %s", user.UUID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	authLogger.Debug("Successfully generated JWT token for UUID: %s", user.UUID)

	responseData := models.LoginResponse{
		Success: true,
		Message: "Login successful",
		UUID:    user.UUID, // Backward compatibility
		Token:   token,
	}
	utils.EncodeJSONResponse(w, responseData, http.StatusOK)
	authLogger.LogResponse(http.StatusOK, "LoginHandler completed successfully")
}
