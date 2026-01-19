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

func TestRegisterV2Handler_EndToEnd(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	req1 := httptest.NewRequest("POST", "/api/v2/register", bytes.NewBuffer([]byte("{}")))
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

	if len(phase1Response.AccountNumber) != 43 {
		t.Errorf("Phase 1 Response.AccountNumber length = %d, want 43", len(phase1Response.AccountNumber))
	}

	if phase1Response.PendingToken == "" {
		t.Error("Phase 1 Response.PendingToken should not be empty")
	}

	if phase1Response.Confirmed {
		t.Error("Phase 1 Response.Confirmed should be false")
	}

	phase2Body := models.RegisterRequest{
		Confirm:      true,
		PendingToken: phase1Response.PendingToken,
	}

	bodyBytes, _ := json.Marshal(phase2Body)
	req2 := httptest.NewRequest("POST", "/api/v2/register", bytes.NewBuffer(bodyBytes))
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

	userRepo := store.NewUserRepository()
	lookup, _ := auth.ComputeAccountLookup(phase1Response.AccountNumber)
	user, err := userRepo.FindByAccountLookup(lookup)
	if err != nil {
		t.Fatalf("User should exist in database: %v", err)
	}
	if user == nil {
		t.Fatal("User should not be nil")
	}
	if user.MasterKeyID == "" {
		t.Error("User.MasterKeyID should not be empty")
	}

	testutil.CleanupTestUser(t, user.UUID)
}

func TestLoginV2Handler_EndToEnd(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	testutil.SetupTestDatabaseEnv()
	testutil.InitializeTestDatabase(t)
	defer testutil.CleanupTestData(t)

	req1 := httptest.NewRequest("POST", "/api/v2/register", bytes.NewBuffer([]byte("{}")))
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

	phase2Body := models.RegisterRequest{
		Confirm:      true,
		PendingToken: phase1Response.PendingToken,
	}
	bodyBytes, _ := json.Marshal(phase2Body)
	req2 := httptest.NewRequest("POST", "/api/v2/register", bytes.NewBuffer(bodyBytes))
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

	userRepo := store.NewUserRepository()
	lookup, _ := auth.ComputeAccountLookup(phase1Response.AccountNumber)
	user, err := userRepo.FindByAccountLookup(lookup)
	if err != nil {
		t.Fatalf("User should exist in database: %v", err)
	}
	if user == nil {
		t.Fatal("User should not be nil")
	}
	defer testutil.CleanupTestUser(t, user.UUID)

	loginBody := models.LoginRequest{
		AccountNumber: phase1Response.AccountNumber,
	}

	bodyBytes, _ = json.Marshal(loginBody)
	req := httptest.NewRequest("POST", "/api/v2/login", bytes.NewBuffer(bodyBytes))
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
