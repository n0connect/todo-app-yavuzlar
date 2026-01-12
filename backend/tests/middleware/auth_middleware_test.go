package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"todo-app-backend/internal/auth"
	"todo-app-backend/internal/middleware"
	"todo-app-backend/tests/testutil"
)

func TestJWTMiddleware_ValidToken(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Skip: This test requires a database connection for userRepository.FindByUUID
	// For a full test, a test database would be needed (see tests/testutil/db.go)
	t.Skip("TestJWTMiddleware_ValidToken requires database connection (userRepository.FindByUUID)")

	// Create a valid JWT token
	userUUID := "test-user-uuid-123456789012"
	token, err := auth.SignToken(userUUID)
	if err != nil {
		t.Fatalf("SignToken() failed: %v", err)
	}

	// Create request with Authorization header
	req := httptest.NewRequest("GET", "/api/v1/todos", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Create a test handler
	handlerCalled := false
	testHandler := middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		userUUIDFromCtx, ok := auth.GetUserUUID(r.Context())
		if !ok {
			t.Error("GetUserUUID() should return true")
		}
		if userUUIDFromCtx != userUUID {
			t.Errorf("GetUserUUID() = %q, want %q", userUUIDFromCtx, userUUID)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Execute
	handler := http.HandlerFunc(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify
	if !handlerCalled {
		t.Error("Handler should have been called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestJWTMiddleware_MissingHeader(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	req := httptest.NewRequest("GET", "/api/v1/todos", nil)
	// No Authorization header

	rr := httptest.NewRecorder()

	handlerCalled := false
	testHandler := middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	handler := http.HandlerFunc(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify
	if handlerCalled {
		t.Error("Handler should not have been called")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_InvalidToken(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	req := httptest.NewRequest("GET", "/api/v1/todos", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")

	rr := httptest.NewRecorder()

	handlerCalled := false
	testHandler := middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	handler := http.HandlerFunc(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify
	if handlerCalled {
		t.Error("Handler should not have been called")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_ExpiredToken(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Note: Testing expired tokens requires time manipulation
	// For now, we test that invalid tokens are rejected
	req := httptest.NewRequest("GET", "/api/v1/todos", nil)
	req.Header.Set("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0IiwiZXhwIjoxfQ.invalid")

	rr := httptest.NewRecorder()

	handlerCalled := false
	testHandler := middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
	})

	handler := http.HandlerFunc(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify
	if handlerCalled {
		t.Error("Handler should not have been called")
	}
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestJWTMiddleware_InvalidHeaderFormat(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	tests := []struct {
		name   string
		header string
	}{
		{
			name:   "no Bearer prefix",
			header: "token-here",
		},
		{
			name:   "empty token",
			header: "Bearer ",
		},
		{
			name:   "missing space",
			header: "Bearertoken",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/todos", nil)
			req.Header.Set("Authorization", tt.header)

			rr := httptest.NewRecorder()

			handlerCalled := false
			testHandler := middleware.JWTMiddleware(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
			})

			handler := http.HandlerFunc(testHandler)
			handler.ServeHTTP(rr, req)

			// Verify
			if handlerCalled {
				t.Error("Handler should not have been called")
			}
			if rr.Code != http.StatusUnauthorized {
				t.Errorf("Status code = %d, want %d", rr.Code, http.StatusUnauthorized)
			}
		})
	}
}
