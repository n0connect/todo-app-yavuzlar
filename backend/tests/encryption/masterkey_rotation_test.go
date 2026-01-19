package encryption

import (
	"os"
	"testing"

	"todo-app-backend/internal/encryption"
	"todo-app-backend/tests/testutil"
)

func TestMasterKeyRotation_DecryptFallback(t *testing.T) {
	testutil.SetupTestEnv(t)
	defer testutil.TeardownTestEnv(t)

	oldKey := "1111111111111111111111111111111111111111111111111111111111111111"
	newKey := "2222222222222222222222222222222222222222222222222222222222222222"

	os.Setenv("MASTER_KEY_ACTIVE_ID", "old")
	os.Setenv("MASTER_KEY_ACTIVE", oldKey)
	os.Setenv("MASTER_KEY_OLD", "")
	encryption.ResetMasterKeysForTesting()

	ciphertext, err := encryption.EncryptWithMasterKey("rotation-test")
	if err != nil {
		t.Fatalf("EncryptWithMasterKey() failed: %v", err)
	}

	os.Setenv("MASTER_KEY_ACTIVE_ID", "new")
	os.Setenv("MASTER_KEY_ACTIVE", newKey)
	os.Setenv("MASTER_KEY_OLD", "old:"+oldKey)
	encryption.ResetMasterKeysForTesting()

	plaintext, err := encryption.DecryptWithMasterKey(ciphertext, "")
	if err != nil {
		t.Fatalf("DecryptWithMasterKey() failed: %v", err)
	}

	if plaintext != "rotation-test" {
		t.Errorf("DecryptWithMasterKey() = %q, want %q", plaintext, "rotation-test")
	}
}
