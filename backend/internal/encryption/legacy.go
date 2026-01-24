package encryption

import (
	"encoding/base64"
	"fmt"

	"todo-app-backend/internal/cryptoengine"
)

// LegacyDecryptWithKey decrypts data encrypted with old double-base64 method
// Flow (OLD): encrypted data -> base64 decode -> AES decrypt -> base64 decode -> raw data
// This function provides backward compatibility for data encrypted before the fix
func LegacyDecryptWithKey(encryptedText string, key []byte) (string, error) {
	cryptoLogger.Debug("LegacyDecryptWithKey: starting legacy decryption, input length: %d", len(encryptedText))

	if len(key) != cryptoengine.AES256KeySize {
		return "", fmt.Errorf("invalid key length")
	}

	// Step 1: Base64 decode the stored encrypted data
	data, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		cryptoLogger.LogError("Base64 decode (step 1)", err)
		return "", err
	}
	cryptoLogger.Debug("LegacyDecryptWithKey: base64 decoded, data length: %d", len(data))

	// Step 2: AES decrypt
	base64Encoded, err := cryptoengine.DecryptAES256GCM(key, data, nil)
	if err != nil {
		cryptoLogger.LogError("DecryptAES256GCM", err)
		return "", err
	}
	cryptoLogger.Debug("LegacyDecryptWithKey: AES decryption completed, base64 string length: %d", len(base64Encoded))

	// Step 3: Base64 decode to get original text (OLD METHOD)
	plaintext, err := base64.StdEncoding.DecodeString(string(base64Encoded))
	if err != nil {
		cryptoLogger.LogError("Base64 decode (step 2)", err)
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	cryptoLogger.Debug("LegacyDecryptWithKey: final base64 decode completed, plaintext length: %d", len(plaintext))

	return string(plaintext), nil
}

// SmartDecryptWithKey attempts new format first, falls back to legacy if needed
// This provides seamless migration - no user action required
// SECURITY: This is safe because both methods use authenticated encryption (AES-GCM)
// Failed decryption with new method does not leak information
func SmartDecryptWithKey(encryptedText string, key []byte) (string, error) {
	cryptoLogger.Debug("SmartDecryptWithKey: attempting new format decryption first")

	// Try new format first (single base64)
	plaintext, err := DecryptWithKey(encryptedText, key)
	if err == nil {
		cryptoLogger.Debug("SmartDecryptWithKey: successfully decrypted with new format")
		return plaintext, nil
	}

	// New format failed - try legacy format
	cryptoLogger.Debug("SmartDecryptWithKey: new format failed, trying legacy format")
	plaintext, err = LegacyDecryptWithKey(encryptedText, key)
	if err == nil {
		cryptoLogger.Info("SmartDecryptWithKey: successfully decrypted with legacy format (consider re-encrypting)")
		return plaintext, nil
	}

	cryptoLogger.Warn("SmartDecryptWithKey: both new and legacy decryption failed")
	return "", fmt.Errorf("decryption failed with both formats")
}

// SmartDecryptWithMasterKey is convenience wrapper using master key
func SmartDecryptWithMasterKey(encryptedText string) (string, error) {
	result, err := SmartDecryptWithKey(encryptedText, ActiveMasterKey())
	if err != nil {
		// Try old master keys if active fails
		oldKeys := GetOldMasterKeys()
		for keyID, key := range oldKeys {
			cryptoLogger.Debug("SmartDecryptWithMasterKey: trying old master key: %s", keyID)
			result, err = SmartDecryptWithKey(encryptedText, key)
			if err == nil {
				cryptoLogger.Info("SmartDecryptWithMasterKey: decrypted with old key %s", keyID)
				return result, nil
			}
		}
		return "", err
	}
	return result, nil
}

// MigrateToNewFormat decrypts with legacy format and re-encrypts with new format
// Use this for batch migration scripts
func MigrateToNewFormat(legacyEncrypted string, key []byte) (string, error) {
	cryptoLogger.Info("MigrateToNewFormat: migrating data from legacy to new format")

	// Decrypt with legacy
	plaintext, err := LegacyDecryptWithKey(legacyEncrypted, key)
	if err != nil {
		return "", fmt.Errorf("legacy decryption failed: %w", err)
	}

	// Re-encrypt with new format
	newEncrypted, err := EncryptWithKey(plaintext, key)
	if err != nil {
		return "", fmt.Errorf("new encryption failed: %w", err)
	}

	cryptoLogger.Info("MigrateToNewFormat: migration successful")
	return newEncrypted, nil
}
