package encryption

import (
	"encoding/hex"
	"testing"

	"github.com/google/uuid"

	"todo-app-backend/internal/encryption"
	"todo-app-backend/tests/testutil"
)

func TestEncryptWithAAD_Basic(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Generate user key
	userKey, err := encryption.GenerateUserAESKey()
	if err != nil {
		t.Fatalf("GenerateUserAESKey() failed: %v", err)
	}

	if len(userKey) != 32 {
		t.Errorf("GenerateUserAESKey() length = %d, want 32", len(userKey))
	}

	// Test data
	plaintext := "Test todo title"
	userUUID := "test-user-uuid-123456789012"
	todoID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	field := encryption.FieldTitle
	purpose := encryption.PurposeStoredRecord

	// Build AAD
	aad := encryption.BuildAAD(userUUID, todoID, field, purpose)

	// Encrypt
	ciphertext, err := encryption.EncryptWithAAD(plaintext, userKey, aad)
	if err != nil {
		t.Fatalf("EncryptWithAAD() failed: %v", err)
	}

	if ciphertext == "" {
		t.Error("EncryptWithAAD() returned empty ciphertext")
	}

	// Decrypt
	decrypted, err := encryption.DecryptWithAAD(ciphertext, userKey, aad)
	if err != nil {
		t.Fatalf("DecryptWithAAD() failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("DecryptWithAAD() = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptWithAAD_CrossUserSwap_ShouldFail(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Generate two different user keys
	userKey1, err := encryption.GenerateUserAESKey()
	if err != nil {
		t.Fatalf("GenerateUserAESKey() failed: %v", err)
	}

	userKey2, err := encryption.GenerateUserAESKey()
	if err != nil {
		t.Fatalf("GenerateUserAESKey() failed: %v", err)
	}

	// Test data
	plaintext := "User1's secret todo"
	userUUID1 := "user1-uuid-123456789012"
	userUUID2 := "user2-uuid-987654321098"
	todoID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	field := encryption.FieldTitle
	purpose := encryption.PurposeStoredRecord

	// Build AAD for user1
	aad1 := encryption.BuildAAD(userUUID1, todoID, field, purpose)

	// Encrypt with user1's key and AAD
	ciphertext, err := encryption.EncryptWithAAD(plaintext, userKey1, aad1)
	if err != nil {
		t.Fatalf("EncryptWithAAD() failed: %v", err)
	}

	// Try to decrypt with user2's key (should fail)
	aad2 := encryption.BuildAAD(userUUID2, todoID, field, purpose)
	_, err = encryption.DecryptWithAAD(ciphertext, userKey2, aad2)
	if err == nil {
		t.Error("DecryptWithAAD() with different user's key should have failed (AAD mismatch)")
	}

	// Try to decrypt with user1's key but user2's AAD (should fail)
	_, err = encryption.DecryptWithAAD(ciphertext, userKey1, aad2)
	if err == nil {
		t.Error("DecryptWithAAD() with different AAD should have failed (cross-user data swap protection)")
	}

	// Try to decrypt with user2's key and user1's AAD (should fail)
	_, err = encryption.DecryptWithAAD(ciphertext, userKey2, aad1)
	if err == nil {
		t.Error("DecryptWithAAD() with wrong key should have failed")
	}
}

func TestEncryptWithAAD_DifferentFields_ShouldFail(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	userKey, err := encryption.GenerateUserAESKey()
	if err != nil {
		t.Fatalf("GenerateUserAESKey() failed: %v", err)
	}

	plaintext := "Test data"
	userUUID := "test-user-uuid-123456789012"
	todoID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	purpose := encryption.PurposeStoredRecord

	// Encrypt with FieldTitle
	aadTitle := encryption.BuildAAD(userUUID, todoID, encryption.FieldTitle, purpose)
	ciphertext, err := encryption.EncryptWithAAD(plaintext, userKey, aadTitle)
	if err != nil {
		t.Fatalf("EncryptWithAAD() failed: %v", err)
	}

	// Try to decrypt with FieldTags AAD (should fail)
	aadTags := encryption.BuildAAD(userUUID, todoID, encryption.FieldTags, purpose)
	_, err = encryption.DecryptWithAAD(ciphertext, userKey, aadTags)
	if err == nil {
		t.Error("DecryptWithAAD() with different field AAD should have failed")
	}
}

func TestEncryptWithAAD_DifferentTodoIDs_ShouldFail(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	userKey, err := encryption.GenerateUserAESKey()
	if err != nil {
		t.Fatalf("GenerateUserAESKey() failed: %v", err)
	}

	plaintext := "Test data"
	userUUID := "test-user-uuid-123456789012"
	todoID1, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	todoID2, _ := uuid.Parse("660e8400-e29b-41d4-a716-446655440001")
	field := encryption.FieldTitle
	purpose := encryption.PurposeStoredRecord

	// Encrypt with todoID1
	aad1 := encryption.BuildAAD(userUUID, todoID1, field, purpose)
	ciphertext, err := encryption.EncryptWithAAD(plaintext, userKey, aad1)
	if err != nil {
		t.Fatalf("EncryptWithAAD() failed: %v", err)
	}

	// Try to decrypt with todoID2 AAD (should fail)
	aad2 := encryption.BuildAAD(userUUID, todoID2, field, purpose)
	_, err = encryption.DecryptWithAAD(ciphertext, userKey, aad2)
	if err == nil {
		t.Error("DecryptWithAAD() with different todo ID AAD should have failed")
	}
}

func TestEncryptWithMasterKey(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	// Skip: This test requires database.EncryptionKey to be initialized
	// which requires database.Init() to be called
	// For full test, see integration tests with database setup
	t.Skip("TestEncryptWithMasterKey requires database.EncryptionKey initialization (database.Init())")

	// Convert to hex for storage
	userKeyHex := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	// Encrypt with master key
	encrypted, err := encryption.EncryptWithMasterKey(userKeyHex)
	if err != nil {
		t.Fatalf("EncryptWithMasterKey() failed: %v", err)
	}

	if encrypted == "" {
		t.Error("EncryptWithMasterKey() returned empty encrypted key")
	}

	// Decrypt
	decryptedBytes, err := encryption.DecryptUserKey(encrypted)
	if err != nil {
		t.Fatalf("DecryptUserKey() failed: %v", err)
	}

	// DecryptUserKey returns []byte (decoded from hex)
	// Convert back to hex string for comparison
	decryptedHex := hex.EncodeToString(decryptedBytes)
	if decryptedHex != userKeyHex {
		t.Errorf("DecryptUserKey() = %q, want %q", decryptedHex, userKeyHex)
	}
}

func TestEncryptWithAAD_EmptyPlaintext(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	userKey, err := encryption.GenerateUserAESKey()
	if err != nil {
		t.Fatalf("GenerateUserAESKey() failed: %v", err)
	}

	plaintext := ""
	userUUID := "test-user-uuid-123456789012"
	todoID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	aad := encryption.BuildAAD(userUUID, todoID, encryption.FieldTitle, encryption.PurposeStoredRecord)

	// Encrypt empty string
	ciphertext, err := encryption.EncryptWithAAD(plaintext, userKey, aad)
	if err != nil {
		t.Fatalf("EncryptWithAAD() with empty plaintext failed: %v", err)
	}

	// Decrypt
	decrypted, err := encryption.DecryptWithAAD(ciphertext, userKey, aad)
	if err != nil {
		t.Fatalf("DecryptWithAAD() failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("DecryptWithAAD() = %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptWithAAD_LongPlaintext(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	userKey, err := encryption.GenerateUserAESKey()
	if err != nil {
		t.Fatalf("GenerateUserAESKey() failed: %v", err)
	}

	// Create a long plaintext (1000 characters)
	plaintext := ""
	for i := 0; i < 1000; i++ {
		plaintext += "A"
	}

	userUUID := "test-user-uuid-123456789012"
	todoID, _ := uuid.Parse("550e8400-e29b-41d4-a716-446655440000")
	aad := encryption.BuildAAD(userUUID, todoID, encryption.FieldTitle, encryption.PurposeStoredRecord)

	// Encrypt
	ciphertext, err := encryption.EncryptWithAAD(plaintext, userKey, aad)
	if err != nil {
		t.Fatalf("EncryptWithAAD() with long plaintext failed: %v", err)
	}

	// Decrypt
	decrypted, err := encryption.DecryptWithAAD(ciphertext, userKey, aad)
	if err != nil {
		t.Fatalf("DecryptWithAAD() failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("DecryptWithAAD() length = %d, want %d", len(decrypted), len(plaintext))
	}
}
