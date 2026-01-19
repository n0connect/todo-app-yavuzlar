package encryption

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"todo-app-backend/internal/cryptoengine"
	"todo-app-backend/internal/utils"
)

var cryptoLogger = utils.NewLogger("CRYPTO")

// EncryptWithKey encrypts data: first base64 encode, then AES encrypt
// Flow: raw data -> base64 encode -> AES encrypt -> base64 encode (for storage)
func EncryptWithKey(text string, key []byte) (string, error) {
	cryptoLogger.Debug("EncryptWithKey: starting encryption, input length: %d", len(text))

	if len(key) != cryptoengine.AES256KeySize {
		return "", fmt.Errorf("invalid key length")
	}

	// Step 1: Base64 encode the raw text first
	base64Encoded := base64.StdEncoding.EncodeToString([]byte(text))
	cryptoLogger.Debug("EncryptWithKey: base64 encoded, length: %d", len(base64Encoded))

	// Step 2: AES encrypt the base64 encoded string
	ciphertext, err := cryptoengine.EncryptAES256GCM(key, []byte(base64Encoded), nil)
	if err != nil {
		cryptoLogger.LogError("EncryptAES256GCM", err)
		return "", err
	}
	cryptoLogger.Debug("EncryptWithKey: AES encryption completed, ciphertext length: %d", len(ciphertext))

	// Step 3: Base64 encode the encrypted data for storage
	result := base64.StdEncoding.EncodeToString(ciphertext)
	cryptoLogger.Debug("EncryptWithKey: final base64 encoding completed, result length: %d", len(result))
	return result, nil
}

// DecryptWithKey decrypts data: AES decrypt, then base64 decode
// Flow: encrypted data -> base64 decode -> AES decrypt -> base64 decode -> raw data
func DecryptWithKey(encryptedText string, key []byte) (string, error) {
	cryptoLogger.Debug("DecryptWithKey: starting decryption, input length: %d", len(encryptedText))

	if len(key) != cryptoengine.AES256KeySize {
		return "", fmt.Errorf("invalid key length")
	}

	// Step 1: Base64 decode the stored encrypted data
	data, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		cryptoLogger.LogError("Base64 decode (step 1)", err)
		return "", err
	}
	cryptoLogger.Debug("DecryptWithKey: base64 decoded, data length: %d", len(data))

	// Step 2: AES decrypt
	base64Encoded, err := cryptoengine.DecryptAES256GCM(key, data, nil)
	if err != nil {
		cryptoLogger.LogError("DecryptAES256GCM", err)
		return "", err
	}
	cryptoLogger.Debug("DecryptWithKey: AES decryption completed, base64 string length: %d", len(base64Encoded))

	// Step 3: Base64 decode to get original text
	plaintext, err := base64.StdEncoding.DecodeString(string(base64Encoded))
	if err != nil {
		cryptoLogger.LogError("Base64 decode (step 2)", err)
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	cryptoLogger.Debug("DecryptWithKey: final base64 decode completed, plaintext length: %d", len(plaintext))

	return string(plaintext), nil
}

func EncryptWithMasterKey(text string) (string, error) {
	cryptoLogger.Debug("EncryptWithMasterKey: starting encryption with master key, input length: %d", len(text))
	result, err := EncryptWithKey(text, ActiveMasterKey())
	if err != nil {
		cryptoLogger.LogError("EncryptWithMasterKey", err)
		return "", err
	}
	cryptoLogger.Debug("EncryptWithMasterKey: successfully encrypted with master key, result length: %d", len(result))
	return result, nil
}

func DecryptWithMasterKey(encryptedText string, keyID string) (string, error) {
	cryptoLogger.Debug("DecryptWithMasterKey: starting decryption with master key, input length: %d", len(encryptedText))

	if keyID != "" {
		key, ok := MasterKeyByID(keyID)
		if !ok {
			return "", fmt.Errorf("unknown master key id")
		}
		result, err := DecryptWithKey(encryptedText, key)
		if err != nil {
			cryptoLogger.LogError("DecryptWithMasterKey", err)
			return "", err
		}
		cryptoLogger.Debug("DecryptWithMasterKey: successfully decrypted with key id: %s", keyID)
		return result, nil
	}

	result, err := DecryptWithKey(encryptedText, ActiveMasterKey())
	if err == nil {
		cryptoLogger.Debug("DecryptWithMasterKey: successfully decrypted with active key")
		return result, nil
	}

	if HasOldMasterKeys() {
		for oldID, key := range masterKeys.old {
			plaintext, tryErr := DecryptWithKey(encryptedText, key)
			if tryErr == nil {
				cryptoLogger.Warn("DecryptWithMasterKey: decrypted with old key id=%s", oldID)
				return plaintext, nil
			}
		}
	}

	cryptoLogger.LogError("DecryptWithMasterKey", err)
	return "", err
}

func GenerateUserAESKey() ([]byte, error) {
	cryptoLogger.Debug("GenerateUserAESKey: generating new 32-byte AES key")
	key, err := cryptoengine.RandomBytes(cryptoengine.AES256KeySize)
	if err != nil {
		cryptoLogger.LogError("GenerateUserAESKey", err)
		return nil, fmt.Errorf("failed to generate user AES key: %w", err)
	}
	cryptoLogger.Debug("GenerateUserAESKey: successfully generated 32-byte AES key")
	return key, nil
}

// DecryptUserKey decrypts an encrypted user key and returns the raw AES key
// This function is separated from database access for better modularity
func DecryptUserKey(encryptedKey string, keyID string) ([]byte, error) {
	cryptoLogger.Debug("DecryptUserKey: starting decryption, encryptedKey length: %d", len(encryptedKey))

	if encryptedKey == "" {
		cryptoLogger.Error("DecryptUserKey: encrypted key is empty")
		return nil, fmt.Errorf("encrypted key is empty")
	}

	cryptoLogger.Debug("DecryptUserKey: decrypting user key with master key")
	decryptedKeyHex, err := DecryptWithMasterKey(encryptedKey, keyID)
	if err != nil {
		cryptoLogger.LogError("DecryptUserKey", err)
		return nil, fmt.Errorf("failed to decrypt user key: %w", err)
	}
	cryptoLogger.Debug("DecryptUserKey: successfully decrypted user key, hex length: %d", len(decryptedKeyHex))

	userKey, err := hex.DecodeString(decryptedKeyHex)
	if err != nil {
		cryptoLogger.LogError("DecryptUserKey (hex decode)", err)
		return nil, fmt.Errorf("failed to decode user key: %w", err)
	}

	if len(userKey) != 32 {
		cryptoLogger.Error("DecryptUserKey: invalid user key length: %d (expected 32)", len(userKey))
		return nil, fmt.Errorf("invalid user key length")
	}
	cryptoLogger.Debug("DecryptUserKey: successfully decoded user key, length: %d", len(userKey))

	return userKey, nil
}

// EncryptWithAAD encrypts data with Additional Authenticated Data (AAD)
// AAD is used to prevent cross-user data swapping attacks
// Flow: raw data -> base64 encode -> AES-GCM encrypt (with AAD) -> base64 encode
func EncryptWithAAD(text string, key []byte, aad []byte) (string, error) {
	cryptoLogger.Debug("EncryptWithAAD: starting encryption, input length: %d, AAD length: %d", len(text), len(aad))
	if err := validateKeyAndAAD(key, aad, AADLength); err != nil {
		return "", err
	}

	ciphertext, err := cryptoengine.EncryptAES256GCM(key, []byte(text), aad)
	if err != nil {
		cryptoLogger.LogError("EncryptAES256GCM", err)
		return "", err
	}
	result := base64.StdEncoding.EncodeToString(ciphertext)
	cryptoLogger.Debug("EncryptWithAAD: encryption completed, result length: %d", len(result))
	return result, nil
}

// DecryptWithAAD decrypts data with Additional Authenticated Data (AAD)
// AAD must match the AAD used during encryption, otherwise decryption fails
// Flow: encrypted data -> base64 decode -> AES-GCM decrypt (with AAD) -> raw data
func DecryptWithAAD(encryptedText string, key []byte, aad []byte) (string, error) {
	cryptoLogger.Debug("DecryptWithAAD: starting decryption, input length: %d, AAD length: %d", len(encryptedText), len(aad))
	if err := validateKeyAndAAD(key, aad, AADLength); err != nil {
		return "", err
	}

	data, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		cryptoLogger.LogError("Base64 decode (step 1)", err)
		return "", err
	}

	plaintext, err := cryptoengine.DecryptAES256GCM(key, data, aad)
	if err != nil {
		cryptoLogger.LogError("DecryptAES256GCM", err)
		return "", fmt.Errorf("decryption failed: AAD mismatch or invalid ciphertext")
	}

	cryptoLogger.Debug("DecryptWithAAD: decryption completed, plaintext length: %d", len(plaintext))
	return string(plaintext), nil
}

func validateKeyAndAAD(key []byte, aad []byte, expectedAADLen int) error {
	if len(key) != cryptoengine.AES256KeySize {
		cryptoLogger.Error("Invalid key length: %d", len(key))
		return fmt.Errorf("invalid key length")
	}
	if len(aad) != expectedAADLen {
		cryptoLogger.Error("Invalid AAD length: %d (expected %d)", len(aad), expectedAADLen)
		return fmt.Errorf("invalid AAD length")
	}
	return nil
}
