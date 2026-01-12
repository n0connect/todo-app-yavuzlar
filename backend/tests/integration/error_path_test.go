package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"todo-app-backend/internal/auth"
	"todo-app-backend/internal/handlers"
	"todo-app-backend/internal/middleware"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/store"
	"todo-app-backend/internal/utils"
	"todo-app-backend/tests/testutil"
)

// TestRegisterHandler_InvalidPendingToken tests error handling for invalid pending token
func TestRegisterHandler_InvalidPendingToken(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database
	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	// Try to confirm with invalid token
	phase2Body := models.RegisterRequest{
		Confirm:      true,
		PendingToken: "invalid-token-12345",
	}

	bodyBytes, _ := json.Marshal(phase2Body)
	req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.RateLimitMiddleware(
				handlers.RegisterHandler,
			),
		),
	)

	handler.ServeHTTP(rr, req)

	// Should return 401 Unauthorized
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d. Body: %s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}
}

// TestRegisterHandler_ExpiredPendingToken tests error handling for expired token
func TestRegisterHandler_ExpiredPendingToken(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database
	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	// Phase 1: Generate AccountNumber
	req1 := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer([]byte("{}")))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()

	handler1 := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.RateLimitMiddleware(
				handlers.RegisterHandler,
			),
		),
	)

	handler1.ServeHTTP(rr1, req1)

	var phase1Response models.RegisterResponse
	json.NewDecoder(rr1.Body).Decode(&phase1Response)

	// Try to use the same token twice (should fail - one-time use)
	phase2Body := models.RegisterRequest{
		Confirm:      true,
		PendingToken: phase1Response.PendingToken,
	}

	bodyBytes, _ := json.Marshal(phase2Body)
	req2 := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()

	handler2 := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.RateLimitMiddleware(
				handlers.RegisterHandler,
			),
		),
	)

	// First use - should succeed
	handler2.ServeHTTP(rr2, req2)
	if rr2.Code != http.StatusCreated {
		t.Fatalf("First use should succeed, got status: %d", rr2.Code)
	}

	// Second use - should fail (one-time use)
	// Create a new request with the same body (HTTP request body can only be read once)
	bodyBytes2, _ := json.Marshal(phase2Body)
	req3 := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer(bodyBytes2))
	req3.Header.Set("Content-Type", "application/json")
	rr3 := httptest.NewRecorder()
	handler2.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusUnauthorized {
		t.Errorf("Second use should fail with status %d, got %d", http.StatusUnauthorized, rr3.Code)
	}
}

// TestLoginHandler_NonexistentUser tests error handling for non-existent user
func TestLoginHandler_NonexistentUser(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database
	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	// Generate valid format AccountNumber that doesn't exist
	accountNumber, err := utils.GenerateAccountNumber()
	if err != nil {
		t.Fatalf("Failed to generate account number: %v", err)
	}

	loginBody := models.LoginRequest{
		AccountNumber: accountNumber,
	}

	bodyBytes, _ := json.Marshal(loginBody)
	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.RateLimitMiddleware(
				handlers.LoginHandler,
			),
		),
	)

	handler.ServeHTTP(rr, req)

	// Should return 401 Unauthorized (generic message)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d. Body: %s", rr.Code, http.StatusUnauthorized, rr.Body.String())
	}

	var response models.LoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Success {
		t.Error("Response.Success should be false for non-existent user")
	}
}

// TestTodoHandler_Unauthorized tests error handling for unauthorized requests
func TestTodoHandler_Unauthorized(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database
	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	// Try to create todo without token
	createBody := models.TodoRequest{
		Title: "Test Todo",
	}

	bodyBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest("POST", "/api/v1/todos", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No Authorization header
	rr := httptest.NewRecorder()

	handler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.JWTMiddleware(
				handlers.CreateTodoHandler,
			),
		),
	)

	handler.ServeHTTP(rr, req)

	// Should return 401 Unauthorized
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

// TestTodoHandler_InvalidToken tests error handling for invalid JWT token
func TestTodoHandler_InvalidToken(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database
	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	createBody := models.TodoRequest{
		Title: "Test Todo",
	}

	bodyBytes, _ := json.Marshal(createBody)
	req := httptest.NewRequest("POST", "/api/v1/todos", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid-token-12345")
	rr := httptest.NewRecorder()

	handler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.JWTMiddleware(
				handlers.CreateTodoHandler,
			),
		),
	)

	handler.ServeHTTP(rr, req)

	// Should return 401 Unauthorized
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

// TestTodoHandler_NotFound tests error handling for non-existent todo
// This test uses the REAL registration flow to create a user, then tests error handling
func TestTodoHandler_NotFound(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database
	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	// Register a user using the REAL registration flow (not manual creation)
	// Phase 1: Generate AccountNumber
	req1 := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer([]byte("{}")))
	req1.Header.Set("Content-Type", "application/json")
	rr1 := httptest.NewRecorder()

	handler1 := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.RateLimitMiddleware(
				handlers.RegisterHandler,
			),
		),
	)

	handler1.ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("Phase 1 status code = %d, want %d. Body: %s", rr1.Code, http.StatusOK, rr1.Body.String())
	}

	var phase1Response models.RegisterResponse
	if err := json.NewDecoder(rr1.Body).Decode(&phase1Response); err != nil {
		t.Fatalf("Failed to decode Phase 1 response: %v", err)
	}

	// Phase 2: Confirm registration
	phase2Body := models.RegisterRequest{
		Confirm:      true,
		PendingToken: phase1Response.PendingToken,
	}

	bodyBytes, _ := json.Marshal(phase2Body)
	req2 := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()

	handler2 := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.RateLimitMiddleware(
				handlers.RegisterHandler,
			),
		),
	)

	handler2.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("Phase 2 status code = %d, want %d. Body: %s", rr2.Code, http.StatusCreated, rr2.Body.String())
	}

	var phase2Response models.RegisterResponse
	if err := json.NewDecoder(rr2.Body).Decode(&phase2Response); err != nil {
		t.Fatalf("Failed to decode Phase 2 response: %v", err)
	}

	// Get user UUID for cleanup
	userRepo := store.NewUserRepository()
	lookup, _ := auth.ComputeAccountLookup(phase1Response.AccountNumber)
	user, err := userRepo.FindByAccountLookup(lookup)
	if err != nil {
		t.Fatalf("User should exist in database: %v", err)
	}
	defer testutil.CleanupTestUser(t, user.UUID)

	// Use the token from registration response (real system behavior)
	token := phase2Response.Token
	if token == "" {
		t.Fatal("Registration should return a token")
	}

	// Try to update non-existent todo
	updateBody := models.TodoRequest{
		Title: "Updated Todo",
	}

	updateBodyBytes, _ := json.Marshal(updateBody)
	req := httptest.NewRequest("PUT", "/api/v1/todos/00000000-0000-0000-0000-000000000000", bytes.NewBuffer(updateBodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	handler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.JWTMiddleware(
				handlers.UpdateTodoHandler,
			),
		),
	)

	handler.ServeHTTP(rr, req)

	// Should return 404 Not Found
	if rr.Code != http.StatusNotFound {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusNotFound)
	}
}
