package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"todo-app-backend/internal/auth"
	"todo-app-backend/internal/handlers"
	"todo-app-backend/internal/middleware"
	"todo-app-backend/internal/models"
	"todo-app-backend/internal/store"
	"todo-app-backend/tests/testutil"
)

// TestRegisterHandler_ConcurrentPhase1 tests concurrent registration phase 1 requests
func TestRegisterHandler_ConcurrentPhase1(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database
	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	const numRequests = 10
	var wg sync.WaitGroup
	results := make([]*httptest.ResponseRecorder, numRequests)
	errors := make([]error, numRequests)

	handler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.RateLimitMiddleware(
				handlers.RegisterHandler,
			),
		),
	)

	// Make concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer([]byte("{}")))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			results[idx] = rr
			if rr.Code != http.StatusOK {
				errors[idx] = &testError{message: "Request failed", code: rr.Code}
			}
		}(i)
	}

	wg.Wait()

	// Verify all requests succeeded
	successCount := 0
	for i, rr := range results {
		if rr == nil {
			t.Errorf("Request %d: response is nil", i)
			continue
		}
		if rr.Code == http.StatusOK {
			successCount++

			var response models.RegisterResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Errorf("Request %d: failed to decode response: %v", i, err)
				continue
			}

			if !response.Success {
				t.Errorf("Request %d: response.Success should be true", i)
			}

			if response.AccountNumber == "" {
				t.Errorf("Request %d: AccountNumber should not be empty", i)
			}
		} else {
			t.Logf("Request %d: failed with status %d: %s", i, rr.Code, rr.Body.String())
		}
	}

	// All requests should succeed (rate limit allows 20 per minute)
	if successCount < numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, successCount)
	}
}

// TestRegisterHandler_ConcurrentPhase2 tests concurrent registration phase 2 (race condition test)
func TestRegisterHandler_ConcurrentPhase2(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Setup database
	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	// Rate limiter is automatically reset by InitializeTestDatabase

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
	if err := json.NewDecoder(rr1.Body).Decode(&phase1Response); err != nil {
		t.Fatalf("Failed to decode Phase 1 response: %v", err)
	}

	// Try to use the same token concurrently (should only one succeed)
	const numConcurrent = 5
	var wg sync.WaitGroup
	results := make([]int, numConcurrent)

	phase2Body := models.RegisterRequest{
		Confirm:      true,
		PendingToken: phase1Response.PendingToken,
	}

	bodyBytes, _ := json.Marshal(phase2Body)

	handler2 := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.RateLimitMiddleware(
				handlers.RegisterHandler,
			),
		),
	)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler2.ServeHTTP(rr, req)
			results[idx] = rr.Code
		}(i)
	}

	wg.Wait()

	// Only one should succeed (201), others should fail (401 - token already used)
	successCount := 0
	failCount := 0

	for i, code := range results {
		if code == http.StatusCreated {
			successCount++
		} else if code == http.StatusUnauthorized {
			failCount++
		} else {
			t.Errorf("Request %d: unexpected status code %d", i, code)
		}
	}

	// Exactly one should succeed
	if successCount != 1 {
		t.Errorf("Expected exactly 1 successful registration, got %d", successCount)
	}

	// Others should fail
	if failCount != numConcurrent-1 {
		t.Errorf("Expected %d failed registrations, got %d", numConcurrent-1, failCount)
	}
}

// TestTodoHandler_ConcurrentCreate tests concurrent todo creation
// This test uses the REAL registration flow to create a user, then tests concurrent todo operations
func TestTodoHandler_ConcurrentCreate(t *testing.T) {
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

	const numRequests = 10
	var wg sync.WaitGroup
	results := make([]int, numRequests)

	handler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.JWTMiddleware(
				handlers.CreateTodoHandler,
			),
		),
	)

	// Make concurrent create requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			createBody := models.TodoRequest{
				Title:    "Concurrent Todo",
				Priority: "medium",
			}

			bodyBytes, _ := json.Marshal(createBody)
			req := httptest.NewRequest("POST", "/api/v1/todos", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			results[idx] = rr.Code
		}(i)
	}

	wg.Wait()

	// All should succeed
	successCount := 0
	for i, code := range results {
		if code == http.StatusCreated {
			successCount++
		} else {
			t.Errorf("Request %d: expected status %d, got %d", i, http.StatusCreated, code)
		}
	}

	if successCount != numRequests {
		t.Errorf("Expected %d successful creates, got %d", numRequests, successCount)
	}
}

// TestTodoHandler_ConcurrentUpdate tests concurrent todo updates (race condition)
// This test uses the REAL registration flow to create a user, then tests concurrent todo operations
func TestTodoHandler_ConcurrentUpdate(t *testing.T) {
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

	// Create a todo first using the REAL handler (not service directly)
	createBody := models.TodoRequest{
		Title: "Test Todo",
	}
	createBodyBytes, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest("POST", "/api/v1/todos", bytes.NewBuffer(createBodyBytes))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer "+token)
	createRr := httptest.NewRecorder()

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

	const numRequests = 5
	var wg sync.WaitGroup
	results := make([]int, numRequests)

	handler := middleware.SecurityHeadersMiddleware(
		middleware.CORSMiddleware(
			middleware.JWTMiddleware(
				handlers.UpdateTodoHandler,
			),
		),
	)

	// Make concurrent update requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			updateBody := models.TodoRequest{
				Title:     "Updated Todo",
				Completed: true,
			}

			bodyBytes, _ := json.Marshal(updateBody)
			req := httptest.NewRequest("PUT", "/api/v1/todos/"+todoID, bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			results[idx] = rr.Code
		}(i)
	}

	wg.Wait()

	// All should succeed (last write wins)
	successCount := 0
	for i, code := range results {
		if code == http.StatusOK {
			successCount++
		} else {
			t.Logf("Request %d: status code %d", i, code)
		}
	}

	// At least some should succeed
	if successCount == 0 {
		t.Error("Expected at least one successful update")
	}
}

type testError struct {
	message string
	code    int
}

func (e *testError) Error() string {
	return e.message
}
