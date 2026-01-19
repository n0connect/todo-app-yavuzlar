package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"todo-app-backend/internal/middleware"
	"todo-app-backend/tests/testutil"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	req := httptest.NewRequest("GET", "/api/v2/todos", nil)

	rr := httptest.NewRecorder()

	handlerCalled := false
	testHandler := middleware.SecurityHeadersMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := http.HandlerFunc(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify security headers
	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range headers {
		actualValue := rr.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Header %s = %q, want %q", header, actualValue, expectedValue)
		}
	}

	// Verify handler was called
	if !handlerCalled {
		t.Error("Handler should have been called")
	}
}

func TestSecurityHeadersMiddleware_PermissionsPolicy(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	req := httptest.NewRequest("GET", "/api/v2/todos", nil)

	rr := httptest.NewRecorder()

	testHandler := middleware.SecurityHeadersMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := http.HandlerFunc(testHandler)
	handler.ServeHTTP(rr, req)

	// Verify Permissions-Policy header
	permissionsPolicy := rr.Header().Get("Permissions-Policy")
	if permissionsPolicy == "" {
		t.Error("Permissions-Policy header should be set")
	}

	// Should contain geolocation=(), microphone=(), camera=()
	if permissionsPolicy != "geolocation=(), microphone=(), camera=()" {
		t.Errorf("Permissions-Policy = %q, want %q", permissionsPolicy, "geolocation=(), microphone=(), camera=()")
	}
}
