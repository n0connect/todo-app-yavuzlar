package auth

import (
	"os"
	"strings"
	"testing"

	"todo-app-backend/internal/auth"
	"todo-app-backend/tests/testutil"
)

func setupAuthTestEnv(t *testing.T) {
	t.Helper()
	testutil.SetupTestEnv(t)
	t.Cleanup(func() {
		testutil.TeardownTestEnv(t)
	})
}

// TestComputeAccountLookup_Deterministic tests that the same account number
// with the same pepper produces the same lookup value
func TestComputeAccountLookup_Deterministic(t *testing.T) {
	setupAuthTestEnv(t)

	// Set a test pepper
	originalPepper := os.Getenv("ACCOUNT_LOOKUP_PEPPER")
	testPepper := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef" // 64 hex chars = 32 bytes
	os.Setenv("ACCOUNT_LOOKUP_PEPPER", testPepper)
	defer func() {
		if originalPepper != "" {
			os.Setenv("ACCOUNT_LOOKUP_PEPPER", originalPepper)
		} else {
			os.Unsetenv("ACCOUNT_LOOKUP_PEPPER")
		}
	}()

	accountNumber := strings.Repeat("A", 43)

	// Compute lookup twice
	lookup1, err := auth.ComputeAccountLookup(accountNumber)
	if err != nil {
		t.Fatalf("ComputeAccountLookup() failed: %v", err)
	}

	lookup2, err := auth.ComputeAccountLookup(accountNumber)
	if err != nil {
		t.Fatalf("ComputeAccountLookup() failed: %v", err)
	}

	// Should be identical
	if lookup1 != lookup2 {
		t.Errorf("ComputeAccountLookup() is not deterministic: got %q and %q", lookup1, lookup2)
	}

	// Should be hex-encoded (64 hex chars for SHA256)
	if len(lookup1) != 64 {
		t.Errorf("ComputeAccountLookup() length = %d, want 64 (hex-encoded SHA256)", len(lookup1))
	}
}

// TestHashAndVerifyAccountNumber tests Argon2id hashing and verification
func TestHashAndVerifyAccountNumber(t *testing.T) {
	setupAuthTestEnv(t)

	accountNumber := strings.Repeat("A", 43)

	// Generate hash
	hash, err := auth.HashAccountNumber(accountNumber)
	if err != nil {
		t.Fatalf("HashAccountNumber() failed: %v", err)
	}

	// Hash should not be empty
	if hash == "" {
		t.Error("HashAccountNumber() returned empty hash")
	}

	// Hash should contain argon2id prefix
	if len(hash) < 20 {
		t.Errorf("HashAccountNumber() returned hash too short: %d", len(hash))
	}

	// Verify with correct account number
	err = auth.VerifyAccountNumberHash(hash, accountNumber)
	if err != nil {
		t.Errorf("VerifyAccountNumberHash() with correct account number failed: %v", err)
	}

	// Verify with wrong account number
	wrongAccountNumber := strings.Repeat("B", 43)
	err = auth.VerifyAccountNumberHash(hash, wrongAccountNumber)
	if err == nil {
		t.Error("VerifyAccountNumberHash() with wrong account number should have failed")
	}
}

// TestHashAndVerifyAccountNumber_DifferentSalts tests that same account number
// produces different hashes (due to random salt)
func TestHashAndVerifyAccountNumber_DifferentSalts(t *testing.T) {
	setupAuthTestEnv(t)

	accountNumber := strings.Repeat("A", 43)

	// Generate two hashes
	hash1, err := auth.HashAccountNumber(accountNumber)
	if err != nil {
		t.Fatalf("HashAccountNumber() failed: %v", err)
	}

	hash2, err := auth.HashAccountNumber(accountNumber)
	if err != nil {
		t.Fatalf("HashAccountNumber() failed: %v", err)
	}

	// Hashes should be different (different salts)
	if hash1 == hash2 {
		t.Error("HashAccountNumber() produced identical hashes (should have different salts)")
	}

	// But both should verify correctly
	err = auth.VerifyAccountNumberHash(hash1, accountNumber)
	if err != nil {
		t.Errorf("VerifyAccountNumberHash() with hash1 failed: %v", err)
	}

	err = auth.VerifyAccountNumberHash(hash2, accountNumber)
	if err != nil {
		t.Errorf("VerifyAccountNumberHash() with hash2 failed: %v", err)
	}
}

// TestVerifyAccountNumberHash_InvalidFormat_ShouldFail tests that invalid hash formats
// return errors without panicking
func TestVerifyAccountNumberHash_InvalidFormat_ShouldFail(t *testing.T) {
	setupAuthTestEnv(t)

	accountNumber := strings.Repeat("A", 43)

	tests := []struct {
		name        string
		encodedHash string
		description string
	}{
		{
			name:        "empty string",
			encodedHash: "",
			description: "Empty hash string",
		},
		{
			name:        "missing parts",
			encodedHash: "argon2id$m=65536$t=3",
			description: "Missing parts (only 3 parts)",
		},
		{
			name:        "wrong prefix",
			encodedHash: "bcrypt$m=65536$t=3$p=2$salt$hash",
			description: "Wrong algorithm prefix",
		},
		{
			name:        "invalid memory format",
			encodedHash: "argon2id$invalid$t=3$p=2$salt$hash",
			description: "Invalid memory parameter format",
		},
		{
			name:        "invalid time format",
			encodedHash: "argon2id$m=65536$invalid$p=2$salt$hash",
			description: "Invalid time parameter format",
		},
		{
			name:        "invalid parallelism format",
			encodedHash: "argon2id$m=65536$t=3$invalid$salt$hash",
			description: "Invalid parallelism parameter format",
		},
		{
			name:        "empty salt",
			encodedHash: "argon2id$m=65536$t=3$p=2$$hash",
			description: "Empty salt hex",
		},
		{
			name:        "empty hash",
			encodedHash: "argon2id$m=65536$t=3$p=2$salt$",
			description: "Empty hash hex",
		},
		{
			name:        "non-hex salt",
			encodedHash: "argon2id$m=65536$t=3$p=2$nothex$0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			description: "Non-hex salt encoding",
		},
		{
			name:        "non-hex hash",
			encodedHash: "argon2id$m=65536$t=3$p=2$0123456789abcdef0123456789abcdef$nothex",
			description: "Non-hex hash encoding",
		},
		{
			name:        "too many parts",
			encodedHash: "argon2id$m=65536$t=3$p=2$salt$hash$extra",
			description: "Too many parts (7 instead of 6)",
		},
		{
			name:        "missing delimiter",
			encodedHash: "argon2idm=65536t=3p=2salthex",
			description: "Missing $ delimiters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic, should return an error
			err := auth.VerifyAccountNumberHash(tt.encodedHash, accountNumber)
			if err == nil {
				t.Errorf("VerifyAccountNumberHash() with %s should have failed, got nil error", tt.description)
			}
			// Verify error message is informative
			if err != nil && err.Error() == "" {
				t.Error("VerifyAccountNumberHash() returned empty error message")
			}
		})
	}
}

// TestVerifyAccountNumberHash_Empty_ShouldFail tests that empty inputs fail gracefully
func TestVerifyAccountNumberHash_Empty_ShouldFail(t *testing.T) {
	// Empty hash string
	err := auth.VerifyAccountNumberHash("", strings.Repeat("A", 43))
	if err == nil {
		t.Error("VerifyAccountNumberHash() with empty hash should have failed")
	}

	// Empty account number (should still parse hash but fail verification)
	validHash := "argon2id$m=65536$t=3$p=2$0123456789abcdef0123456789abcdef$0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	err = auth.VerifyAccountNumberHash(validHash, "")
	if err == nil {
		t.Error("VerifyAccountNumberHash() with empty account number should have failed")
	}
}
