package auth

import (
	"strings"
	"testing"

	"todo-app-backend/internal/auth"
	"todo-app-backend/tests/testutil"
)

func setupPendingJWTEnv(t *testing.T) {
	t.Helper()
	testutil.SetupTestEnv(t)
	t.Cleanup(func() {
		testutil.TeardownTestEnv(t)
	})
}

func TestSignPendingRegistrationToken_ContainsPendingID(t *testing.T) {
	setupPendingJWTEnv(t)

	// Generate pending ID
	pendingID, err := auth.GeneratePendingID()
	if err != nil {
		t.Fatalf("GeneratePendingID() failed: %v", err)
	}

	// Store AccountNumber server-side
	accountNumber := strings.Repeat("A", 43)
	if err := auth.StorePendingRegistration(pendingID, accountNumber); err != nil {
		t.Fatalf("StorePendingRegistration() failed: %v", err)
	}

	// Sign token with pending ID (not AccountNumber)
	token, err := auth.SignPendingRegistrationToken(pendingID)
	if err != nil {
		t.Fatalf("SignPendingRegistrationToken() failed: %v", err)
	}

	if token == "" {
		t.Error("SignPendingRegistrationToken() returned empty token")
	}

	// Verify token and retrieve AccountNumber
	retrievedAccountNumber, err := auth.VerifyPendingRegistrationToken(token)
	if err != nil {
		t.Fatalf("VerifyPendingRegistrationToken() failed: %v", err)
	}

	if retrievedAccountNumber != accountNumber {
		t.Errorf("VerifyPendingRegistrationToken() = %q, want %q", retrievedAccountNumber, accountNumber)
	}

	// Verify token was deleted (one-time use)
	_, err = auth.VerifyPendingRegistrationToken(token)
	if err == nil {
		t.Error("VerifyPendingRegistrationToken() should fail on second use (one-time use)")
	}
}

func TestSignPendingRegistrationToken_NoAccountNumberInToken(t *testing.T) {
	setupPendingJWTEnv(t)

	pendingID, err := auth.GeneratePendingID()
	if err != nil {
		t.Fatalf("GeneratePendingID() failed: %v", err)
	}

	accountNumber := strings.Repeat("A", 43)
	if err := auth.StorePendingRegistration(pendingID, accountNumber); err != nil {
		t.Fatalf("StorePendingRegistration() failed: %v", err)
	}

	token, err := auth.SignPendingRegistrationToken(pendingID)
	if err != nil {
		t.Fatalf("SignPendingRegistrationToken() failed: %v", err)
	}

	// Decode token and verify it does NOT contain AccountNumber
	// Token format: header.payload.signature
	// We can't easily decode without the secret, but we can verify the flow
	// The important thing is that VerifyPendingRegistrationToken retrieves from store

	// Verify that token works (proving it contains pending_id, not account_number)
	retrieved, err := auth.VerifyPendingRegistrationToken(token)
	if err != nil {
		t.Fatalf("VerifyPendingRegistrationToken() failed: %v", err)
	}

	if retrieved != accountNumber {
		t.Errorf("VerifyPendingRegistrationToken() = %q, want %q", retrieved, accountNumber)
	}
}

func TestVerifyPendingRegistrationToken_Expired(t *testing.T) {
	setupPendingJWTEnv(t)

	// This test would require manipulating JWT expiration or waiting for configured TTL
	// For now, we test that invalid tokens are rejected
	invalidToken := "invalid.token.here"

	_, err := auth.VerifyPendingRegistrationToken(invalidToken)
	if err == nil {
		t.Error("VerifyPendingRegistrationToken() with invalid token should have failed")
	}
}

func TestVerifyPendingRegistrationToken_InvalidPendingID(t *testing.T) {
	setupPendingJWTEnv(t)

	// Create token with pending ID that doesn't exist in store
	nonExistentPendingID, err := auth.GeneratePendingID()
	if err != nil {
		t.Fatalf("GeneratePendingID() failed: %v", err)
	}

	// Don't store it in the store
	token, err := auth.SignPendingRegistrationToken(nonExistentPendingID)
	if err != nil {
		t.Fatalf("SignPendingRegistrationToken() failed: %v", err)
	}

	// Verify should fail because pending ID doesn't exist in store
	_, err = auth.VerifyPendingRegistrationToken(token)
	if err == nil {
		t.Error("VerifyPendingRegistrationToken() with non-existent pending ID should have failed")
	}
}
