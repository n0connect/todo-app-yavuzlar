package middleware

import (
	"net/http/httptest"
	"testing"

	"todo-app-backend/internal/middleware"
	"todo-app-backend/tests/testutil"
)

func TestClientIP_UntrustedProxy_IgnoresForwardedHeaders(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	middleware.ResetTrustedProxiesForTesting()

	req := httptest.NewRequest("GET", "/api/v2/todos", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("X-Forwarded-For", "198.51.100.20")
	req.Header.Set("X-Real-IP", "198.51.100.30")

	ip := middleware.ClientIP(req)
	if ip != "203.0.113.10" {
		t.Errorf("ClientIP() = %q, want %q", ip, "203.0.113.10")
	}
}

func TestClientIP_TrustedProxy_UsesForwardedHeaders(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	t.Setenv("TRUSTED_PROXIES", "203.0.113.0/24")
	middleware.ResetTrustedProxiesForTesting()

	req := httptest.NewRequest("GET", "/api/v2/todos", nil)
	req.RemoteAddr = "203.0.113.10:12345"
	req.Header.Set("X-Forwarded-For", "198.51.100.20, 203.0.113.10")

	ip := middleware.ClientIP(req)
	if ip != "198.51.100.20" {
		t.Errorf("ClientIP() = %q, want %q", ip, "198.51.100.20")
	}
}
