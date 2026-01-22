package handlers

import (
	"encoding/hex"
	"net/http"
	"strings"

	"todo-app-backend/internal/auth"
	"todo-app-backend/internal/encryption"
	"todo-app-backend/internal/middleware"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/pow"
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

	// Prevent caching of sensitive responses
	w.Header().Set("Cache-Control", "no-store")

	// Parse request body
	var req models.RegisterRequest
	if err := utils.DecodeJSONRequest(r, &req); err != nil {
		// Empty body is OK - treat as generate AccountNumber only (confirm=false)
		authLogger.Debug("Empty or invalid body, treating as AccountNumber generation request")
		req.Confirm = false
	}

	// For Phase 1 (generate AccountNumber), we need PoW validation
	if !req.Confirm && req.PoW == nil {
		authLogger.Warn("PoW solution missing for Phase 1 registration")
		http.Error(w, "Proof of work required", http.StatusPreconditionFailed)
		return
	}

	// Validate PoW solution if present
	if req.PoW != nil {
		if !pow.ValidateSolution(req.PoW) {
			authLogger.Warn("Invalid PoW solution provided")
			http.Error(w, "Invalid proof of work", http.StatusPreconditionFailed)
			return
		}
		authLogger.Debug("Valid PoW solution accepted")
	}

	// PHASE 1: Generate AccountNumber only (confirm=false)
	// Returns a pending registration token that must be used to confirm
	if !req.Confirm {
		authLogger.Debug("Phase 1: Generating AccountNumber preview (no account creation)")

		// Generate AccountNumber (256-bit CSPRNG, base64url encoded)
		accountNumber, err := utils.GenerateAccountNumber()
		if err != nil {
			authLogger.LogError("GenerateAccountNumber", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Compute lookup for collision check (almost impossible)
		lookup, err := auth.ComputeAccountLookup(accountNumber)
		if err != nil {
			authLogger.LogError("ComputeAccountLookup", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Check for collision - regenerate if needed (very unlikely but safe)
		exists, err := userRepository.ExistsByAccountLookup(lookup)
		if err != nil {
			authLogger.LogError("ExistsByAccountLookup", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if exists {
			// thanks user for almost impossible situation
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Generate pending ID and store AccountNumber server-side
		pendingID, err := auth.GeneratePendingID()
		if err != nil {
			authLogger.LogError("GeneratePendingID", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Store AccountNumber server-side with pending ID
		if err := auth.StorePendingRegistration(pendingID, accountNumber); err != nil {
			authLogger.LogError("StorePendingRegistration", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Generate pending registration token (contains pending ID, not AccountNumber)
		pendingToken, err := auth.SignPendingRegistrationToken(pendingID)
		if err != nil {
			authLogger.LogError("SignPendingRegistrationToken", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		authLogger.Debug("Generated preview AccountNumber: %s with pending token", utils.MaskAccountNumber(accountNumber))
		responseData := models.RegisterResponse{
			Success:       true,
			Message:       "Account number generated - click continue to create account",
			AccountNumber: accountNumber,
			PendingToken:  pendingToken,
			Confirmed:     false,
		}
		utils.EncodeJSONResponse(w, responseData, http.StatusOK)
		authLogger.LogResponse(http.StatusOK, "AccountNumber preview generated with pending token")
		return
	}

	// PHASE 2: Create account (confirm=true)
	// ZERO-TRUST: Verify pending token and extract AccountNumber from it
	authLogger.Debug("Phase 2: Confirming account creation")

	if req.PendingToken == "" {
		authLogger.Warn("No pending token provided for account confirmation")
		http.Error(w, "Pending token required", http.StatusBadRequest)
		return
	}

	// Verify pending token and extract the AccountNumber from server-side store
	// Note: VerifyPendingRegistrationToken automatically deletes the pending registration (one-time use)
	pendingAccountNumber, err := auth.VerifyPendingRegistrationToken(req.PendingToken)
	if err != nil {
		// SECURITY: Generic error message to prevent information leakage
		// Don't distinguish between expired and invalid tokens
		authLogger.Warn("Invalid or expired pending registration token: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// ZERO-TRUST: AccountNumber comes from verified token, NOT from user input
	authLogger.Debug("Phase 2: Creating account with verified AccountNumber: %s", utils.MaskAccountNumber(pendingAccountNumber))

	// Compute lookup and hash
	lookup, err := auth.ComputeAccountLookup(pendingAccountNumber)
	if err != nil {
		authLogger.LogError("ComputeAccountLookup", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Check if account already exists (race condition or replay attack)
	exists, err := userRepository.ExistsByAccountLookup(lookup)
	if err != nil {
		authLogger.LogError("ExistsByAccountLookup", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if exists {
		// SECURITY: Generic error message to prevent enumeration
		// Don't reveal that account exists - use same message as invalid token
		authLogger.Warn("Account already exists (possible replay): %s", utils.MaskAccountNumber(pendingAccountNumber))
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Hash account number with Argon2id
	accountHash, err := auth.HashAccountNumber(pendingAccountNumber)
	if err != nil {
		authLogger.LogError("HashAccountNumber", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate internal user UUID (for backward compatibility and JWT subject)
	internalUUID, err := utils.GenerateInternalID()
	if err != nil {
		authLogger.LogError("GenerateInternalID", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Generate encryption key for user
	authLogger.Debug("Generating user AES key for AccountNumber: %s", utils.MaskAccountNumber(pendingAccountNumber))
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
	masterKeyID := encryption.ActiveMasterKeyID()

	user := models.User{
		UUID:          internalUUID,
		AccountLookup: lookup,
		AccountHash:   accountHash,
		EncryptedKey:  encryptedUserKey,
		MasterKeyID:   masterKeyID,
	}

	if err := userRepository.Create(&user); err != nil {
		authLogger.LogError("UserRepository.Create", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	authLogger.Info("Successfully registered new user: AccountNumber=%s", utils.MaskAccountNumber(pendingAccountNumber))

	// Generate JWT token for immediate login (using internal UUID as subject)
	token, err := auth.SignToken(internalUUID)
	if err != nil {
		authLogger.LogError("SignToken", err)
		// Account created but token failed - user can still login manually
		// SECURITY: Don't return AccountNumber in response
		responseData := models.RegisterResponse{
			Success:   true,
			Message:   "Account created successfully",
			Confirmed: true,
		}
		utils.EncodeJSONResponse(w, responseData, http.StatusCreated)
		return
	}

	// SECURITY: Don't return AccountNumber in response (already shown to user in Phase 1)
	// Prevents network sniffing and reduces information leakage
	responseData := models.RegisterResponse{
		Success:   true,
		Message:   "Account created successfully",
		Token:     token,
		Confirmed: true,
	}
	utils.EncodeJSONResponse(w, responseData, http.StatusCreated)
	authLogger.LogResponse(http.StatusCreated, "Account created successfully")
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP if multiple are present
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			// Remove port if present
			if colonIndex := strings.Index(ip, ":"); colonIndex != -1 {
				ip = ip[:colonIndex]
			}
			return ip
		}
	}

	// Check X-Real-IP header (another common proxy header)
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIndex := strings.Index(ip, ":"); colonIndex != -1 {
		ip = ip[:colonIndex]
	}
	return ip
}

func PoWChallengeHandler(w http.ResponseWriter, r *http.Request) {
	authLogger.LogRequest(r.Method, r.URL.Path, "N/A")
	authLogger.Debug("Starting PoWChallengeHandler")

	// Handle HEAD requests
	if r.Method == http.MethodHead {
		authLogger.Debug("Handling HEAD request for PoW challenge endpoint")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.Method != http.MethodGet {
		authLogger.Warn("Invalid method for PoW challenge: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Prevent caching of sensitive responses
	w.Header().Set("Cache-Control", "no-store")

	// Get client IP for rate limiting
	clientIP := getClientIP(r)

	// Generate a new PoW challenge
	challenge, err := pow.GenerateChallenge(clientIP)
	if err != nil {
		authLogger.LogError("GenerateChallenge", err)
		if strings.Contains(err.Error(), "rate limit exceeded") {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	authLogger.Debug("Generated PoW challenge with difficulty: %d", challenge.Difficulty)

	responseData := models.PoWChallengeResponse{
		Success:   true,
		Message:   "PoW challenge generated successfully",
		Challenge: challenge,
	}

	utils.EncodeJSONResponse(w, responseData, http.StatusOK)
	authLogger.LogResponse(http.StatusOK, "PoW challenge generated successfully")
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

	// Prevent caching of sensitive responses
	w.Header().Set("Cache-Control", "no-store")

	var req models.LoginRequest
	if err := utils.DecodeJSONRequest(r, &req); err != nil {
		authLogger.LogError("DecodeJSONRequest", err)
		authLogger.Warn("Invalid JSON request for login")
		utils.EncodeJSONResponse(w, models.LoginResponse{Success: false, Message: "Invalid request"}, http.StatusBadRequest)
		return
	}
	authLogger.Debug("Successfully decoded login request, AccountNumber length: %d", len(req.AccountNumber))

	// SECURITY: Validate AccountNumber format with strict regex pattern before any database operations
	// This prevents unnecessary database queries for invalid formats
	// Pattern: ^[A-Za-z0-9_-]{43}$ - exactly 43 base64url characters
	if !utils.ValidateAccountNumber(req.AccountNumber) {
		authLogger.Warn("Invalid AccountNumber format in login request: %s", utils.MaskAccountNumber(req.AccountNumber))
		utils.EncodeJSONResponse(w, models.LoginResponse{Success: false, Message: "Invalid credentials"}, http.StatusUnauthorized)
		return
	}
	authLogger.Debug("AccountNumber format validated: %s", utils.MaskAccountNumber(req.AccountNumber))

	// Compute lookup for database query (only reached if format is valid)
	lookup, err := auth.ComputeAccountLookup(req.AccountNumber)
	if err != nil {
		authLogger.LogError("ComputeAccountLookup", err)
		authLogger.Warn("Login failed: lookup computation error")
		utils.EncodeJSONResponse(w, models.LoginResponse{Success: false, Message: "Invalid credentials"}, http.StatusUnauthorized)
		return
	}

	// Apply per-account rate limit using lookup hash
	if !middleware.AllowAccountLookup(lookup) {
		authLogger.Warn("Login rate limit exceeded for account lookup")
		http.Error(w, "Too many requests", http.StatusTooManyRequests)
		return
	}

	// Find user by lookup
	user, err := userRepository.FindByAccountLookup(lookup)
	if err != nil {
		authLogger.LogError("UserRepository.FindByAccountLookup", err)
		authLogger.Warn("Login failed: user not found")
		// SECURITY: Perform dummy hash verification to prevent timing attack
		// This ensures user existence cannot be determined by response time
		auth.PerformDummyHashVerification(req.AccountNumber)
		utils.EncodeJSONResponse(w, models.LoginResponse{Success: false, Message: "Invalid credentials"}, http.StatusUnauthorized)
		return
	}
	// IDK - If openssl support constant time computation for argon2id hash
	// then why we prefer old-school and not secure dummy-hash verification ?
	// Removed old-school coded parts. Because attack-surface plane is much smaller.
	// Verify AccountNumber hash
	if err := auth.VerifyAccountNumberHash(user.AccountHash, req.AccountNumber); err != nil {
		authLogger.Warn("Login failed: AccountNumber hash verification failed")
		utils.EncodeJSONResponse(w, models.LoginResponse{Success: false, Message: "Invalid credentials"}, http.StatusUnauthorized)
		return
	}

	authLogger.Info("Successful login for AccountNumber: %s", utils.MaskAccountNumber(req.AccountNumber))

	// Generate JWT token (using internal UUID as subject)
	authLogger.Debug("Generating JWT token for user UUID: %s", user.UUID)
	token, err := auth.SignToken(user.UUID)
	if err != nil {
		authLogger.LogError("SignToken", err)
		authLogger.Error("Failed to generate JWT token for user UUID: %s", user.UUID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	authLogger.Debug("Successfully generated JWT token for user UUID: %s", user.UUID)

	responseData := models.LoginResponse{
		Success: true,
		Message: "Login successful",
		Token:   token,
	}
	utils.EncodeJSONResponse(w, responseData, http.StatusOK)
	authLogger.LogResponse(http.StatusOK, "LoginHandler completed successfully")
}
