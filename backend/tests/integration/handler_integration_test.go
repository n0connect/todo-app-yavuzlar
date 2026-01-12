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
	"todo-app-backend/tests/testutil"
)

// TestRegisterHandler_EndToEnd tests the complete registration flow with database
func TestRegisterHandler_EndToEnd(t *testing.T) {
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

	if rr1.Code != http.StatusOK {
		t.Fatalf("Phase 1 status code = %d, want %d. Body: %s", rr1.Code, http.StatusOK, rr1.Body.String())
	}

	var phase1Response models.RegisterResponse
	if err := json.NewDecoder(rr1.Body).Decode(&phase1Response); err != nil {
		t.Fatalf("Failed to decode Phase 1 response: %v", err)
	}

	if !phase1Response.Success {
		t.Error("Phase 1 Response.Success should be true")
	}

	if phase1Response.AccountNumber == "" {
		t.Error("Phase 1 Response.AccountNumber should not be empty")
	}

	if len(phase1Response.AccountNumber) != 32 {
		t.Errorf("Phase 1 Response.AccountNumber length = %d, want 32", len(phase1Response.AccountNumber))
	}

	if phase1Response.PendingToken == "" {
		t.Error("Phase 1 Response.PendingToken should not be empty")
	}

	if phase1Response.Confirmed {
		t.Error("Phase 1 Response.Confirmed should be false")
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

	if !phase2Response.Success {
		t.Error("Phase 2 Response.Success should be true")
	}

	if phase2Response.Token == "" {
		t.Error("Phase 2 Response.Token should not be empty")
	}

	if !phase2Response.Confirmed {
		t.Error("Phase 2 Response.Confirmed should be true")
	}

	// Verify user was created in database
	userRepo := store.NewUserRepository()
	lookup, _ := auth.ComputeAccountLookup(phase1Response.AccountNumber)
	user, err := userRepo.FindByAccountLookup(lookup)
	if err != nil {
		t.Fatalf("User should exist in database: %v", err)
	}

	if user == nil {
		t.Fatal("User should not be nil")
	}

	// Cleanup
	testutil.CleanupTestUser(t, user.UUID)
}

// TestLoginHandler_EndToEnd tests the complete login flow with database
// This test uses the REAL registration flow to create a user, then tests login
// This ensures we test the actual system behavior, not a manipulated state
func TestLoginHandler_EndToEnd(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database
	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	// First, register a user using the REAL registration flow (not manual creation)
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

	accountNumber := phase1Response.AccountNumber

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

	// Get user UUID for cleanup
	userRepo := store.NewUserRepository()
	lookup, _ := auth.ComputeAccountLookup(accountNumber)
	user, err := userRepo.FindByAccountLookup(lookup)
	if err != nil {
		t.Fatalf("User should exist in database: %v", err)
	}
	defer testutil.CleanupTestUser(t, user.UUID)

	// Now test login with the REAL registered user
	loginBody := models.LoginRequest{
		AccountNumber: accountNumber,
	}

	bodyBytes, _ = json.Marshal(loginBody)
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

	if rr.Code != http.StatusOK {
		t.Fatalf("Login status code = %d, want %d. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	var response models.LoginResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode login response: %v", err)
	}

	if !response.Success {
		t.Error("Login Response.Success should be true")
	}

	if response.Token == "" {
		t.Error("Login Response.Token should not be empty")
	}
}

// TestTodoHandlers_EndToEnd tests the complete todo CRUD flow with database
// This test uses the REAL registration flow to create a user, then tests todo operations
// This ensures we test the actual system behavior, not a manipulated state
func TestTodoHandlers_EndToEnd(t *testing.T) {
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

	// Test Create Todo
	createBody := models.TodoRequest{
		Title:    "Test Todo",
		Priority: "high",
		Tags:     []string{"test", "integration"},
	}

	createBodyBytes, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest("POST", "/api/v1/todos", bytes.NewBuffer(createBodyBytes))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRr := httptest.NewRecorder()

	// Create middleware chain with auth
	createHandler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.JWTMiddleware(
				handlers.CreateTodoHandler,
			),
		),
	)

	createHandler.ServeHTTP(createRr, createReq)

	if createRr.Code != http.StatusCreated {
		t.Fatalf("Create todo status code = %d, want %d. Body: %s", createRr.Code, http.StatusCreated, createRr.Body.String())
	}

	var createResponse map[string]interface{}
	if err := json.NewDecoder(createRr.Body).Decode(&createResponse); err != nil {
		t.Fatalf("Failed to decode create response: %v", err)
	}

	todoID, ok := createResponse["id"].(string)
	if !ok {
		t.Fatal("Create response should contain id")
	}

	// Test Get Todos
	getReq := httptest.NewRequest("GET", "/api/v1/todos", nil)
	getReq.Header.Set("Authorization", "Bearer "+token)
	getRr := httptest.NewRecorder()

	getHandler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.JWTMiddleware(
				handlers.GetTodosHandler,
			),
		),
	)

	getHandler.ServeHTTP(getRr, getReq)

	if getRr.Code != http.StatusOK {
		t.Fatalf("Get todos status code = %d, want %d. Body: %s", getRr.Code, http.StatusOK, getRr.Body.String())
	}

	var todos []map[string]interface{}
	if err := json.NewDecoder(getRr.Body).Decode(&todos); err != nil {
		t.Fatalf("Failed to decode todos response: %v", err)
	}

	if len(todos) != 1 {
		t.Errorf("Expected 1 todo, got %d", len(todos))
	}

	// Test Update Todo
	updateBody := models.TodoRequest{
		Title:    "Updated Todo",
		Completed: true,
	}

	bodyBytes, _ = json.Marshal(updateBody)
	req3 := httptest.NewRequest("PUT", "/api/v1/todos/"+todoID, bytes.NewBuffer(bodyBytes))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer "+token)
	rr3 := httptest.NewRecorder()

	updateHandler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.JWTMiddleware(
				handlers.UpdateTodoHandler,
			),
		),
	)

	updateHandler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusOK {
		t.Fatalf("Update todo status code = %d, want %d. Body: %s", rr3.Code, http.StatusOK, rr3.Body.String())
	}

	// Test Delete Todo
	req4 := httptest.NewRequest("DELETE", "/api/v1/todos/"+todoID, nil)
	req4.Header.Set("Authorization", "Bearer "+token)
	rr4 := httptest.NewRecorder()

	deleteHandler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.JWTMiddleware(
				handlers.DeleteTodoHandler,
			),
		),
	)

	deleteHandler.ServeHTTP(rr4, req4)

	if rr4.Code != http.StatusNoContent {
		t.Fatalf("Delete todo status code = %d, want %d", rr4.Code, http.StatusNoContent)
	}
}
