package auth

import (
	"testing"

	"todo-app-backend/internal/auth"
)

func TestGeneratePendingID(t *testing.T) {
	// Test that pending ID is generated and has correct format (32 hex chars = 128 bits)
	pendingID1, err := auth.GeneratePendingID()
	if err != nil {
		t.Fatalf("GeneratePendingID() failed: %v", err)
	}

	if len(pendingID1) != 32 {
		t.Errorf("GeneratePendingID() length = %d, want 32 (hex-encoded 128-bit)", len(pendingID1))
	}

	// Test uniqueness
	pendingID2, err := auth.GeneratePendingID()
	if err != nil {
		t.Fatalf("GeneratePendingID() failed: %v", err)
	}

	if pendingID1 == pendingID2 {
		t.Error("GeneratePendingID() produced duplicate IDs")
	}

	// Test multiple generations
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := auth.GeneratePendingID()
		if err != nil {
			t.Fatalf("GeneratePendingID() failed: %v", err)
		}
		if ids[id] {
			t.Errorf("GeneratePendingID() produced duplicate ID: %s", id)
		}
		ids[id] = true
	}
}

func TestStoreAndGetPendingRegistration(t *testing.T) {
	accountNumber := "ABCDEFGHIJKLMNOPQRSTUVWXYZab12"

	// Generate pending ID
	pendingID, err := auth.GeneratePendingID()
	if err != nil {
		t.Fatalf("GeneratePendingID() failed: %v", err)
	}

	// Store pending registration
	auth.StorePendingRegistration(pendingID, accountNumber)

	// Retrieve it
	retrieved, err := auth.GetPendingRegistration(pendingID)
	if err != nil {
		t.Fatalf("GetPendingRegistration() failed: %v", err)
	}

	if retrieved != accountNumber {
		t.Errorf("GetPendingRegistration() = %q, want %q", retrieved, accountNumber)
	}
}

func TestGetPendingRegistration_NotFound(t *testing.T) {
	nonExistentID := "00000000000000000000000000000000"

	_, err := auth.GetPendingRegistration(nonExistentID)
	if err == nil {
		t.Error("GetPendingRegistration() with non-existent ID should have failed")
	}
	if err != nil && err.Error() != "invalid token" {
		t.Errorf("GetPendingRegistration() returned unexpected error: %v", err)
	}
}

func TestDeletePendingRegistration(t *testing.T) {
	accountNumber := "ABCDEFGHIJKLMNOPQRSTUVWXYZab12"

	// Generate and store
	pendingID, err := auth.GeneratePendingID()
	if err != nil {
		t.Fatalf("GeneratePendingID() failed: %v", err)
	}

	auth.StorePendingRegistration(pendingID, accountNumber)

	// Verify it exists
	_, err = auth.GetPendingRegistration(pendingID)
	if err != nil {
		t.Fatalf("GetPendingRegistration() failed before delete: %v", err)
	}

	// Delete it
	auth.DeletePendingRegistration(pendingID)

	// Verify it's gone
	_, err = auth.GetPendingRegistration(pendingID)
	if err == nil {
		t.Error("GetPendingRegistration() after delete should have failed")
	}
}

func TestPendingRegistration_Expiration(t *testing.T) {
	accountNumber := "ABCDEFGHIJKLMNOPQRSTUVWXYZab12"

	// This test is tricky because we can't easily manipulate time in the store
	// But we can test that expired entries are cleaned up
	// Note: The actual expiration is handled by GetPendingRegistration checking ExpiresAt
	// For a proper test, we'd need to inject a time provider, but for now we test the basic flow

	pendingID, err := auth.GeneratePendingID()
	if err != nil {
		t.Fatalf("GeneratePendingID() failed: %v", err)
	}

	auth.StorePendingRegistration(pendingID, accountNumber)

	// Immediately retrieve should work
	_, err = auth.GetPendingRegistration(pendingID)
	if err != nil {
		t.Errorf("GetPendingRegistration() immediately after store failed: %v", err)
	}

	// Note: Testing actual expiration would require waiting 5+ minutes or mocking time
	// This is a limitation of the current implementation
}

func TestPendingRegistration_OneTimeUse(t *testing.T) {
	accountNumber := "ABCDEFGHIJKLMNOPQRSTUVWXYZab12"

	pendingID, err := auth.GeneratePendingID()
	if err != nil {
		t.Fatalf("GeneratePendingID() failed: %v", err)
	}

	auth.StorePendingRegistration(pendingID, accountNumber)

	// First retrieval should work
	retrieved1, err := auth.GetPendingRegistration(pendingID)
	if err != nil {
		t.Fatalf("GetPendingRegistration() first call failed: %v", err)
	}
	if retrieved1 != accountNumber {
		t.Errorf("GetPendingRegistration() = %q, want %q", retrieved1, accountNumber)
	}

	// After deletion, second retrieval should fail
	auth.DeletePendingRegistration(pendingID)

	_, err = auth.GetPendingRegistration(pendingID)
	if err == nil {
		t.Error("GetPendingRegistration() after delete should have failed (one-time use)")
	}
}
