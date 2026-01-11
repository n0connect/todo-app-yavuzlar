package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"todo-app-backend/internal/database"
	"todo-app-backend/internal/utils"
)

var cryptoLogger = utils.NewLogger("CRYPTO")

// EncryptWithKey encrypts data: first base64 encode, then AES encrypt
// Flow: raw data -> base64 encode -> AES encrypt -> base64 encode (for storage)
func EncryptWithKey(text string, key []byte) (string, error) {
	cryptoLogger.Debug("EncryptWithKey: starting encryption, input length: %d", len(text))

	// Step 1: Base64 encode the raw text first
	base64Encoded := base64.StdEncoding.EncodeToString([]byte(text))
	cryptoLogger.Debug("EncryptWithKey: base64 encoded, length: %d", len(base64Encoded))

	// Step 2: AES encrypt the base64 encoded string
	block, err := aes.NewCipher(key)
	if err != nil {
		cryptoLogger.LogError("AES NewCipher", err)
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		cryptoLogger.LogError("GCM NewGCM", err)
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		cryptoLogger.LogError("Random nonce generation", err)
		return "", err
	}
	cryptoLogger.Debug("EncryptWithKey: generated nonce, size: %d", len(nonce))

	ciphertext := gcm.Seal(nonce, nonce, []byte(base64Encoded), nil)
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

	// Step 1: Base64 decode the stored encrypted data
	data, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		cryptoLogger.LogError("Base64 decode (step 1)", err)
		return "", err
	}
	cryptoLogger.Debug("DecryptWithKey: base64 decoded, data length: %d", len(data))

	// Step 2: AES decrypt
	block, err := aes.NewCipher(key)
	if err != nil {
		cryptoLogger.LogError("AES NewCipher", err)
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		cryptoLogger.LogError("GCM NewGCM", err)
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		cryptoLogger.Error("DecryptWithKey: ciphertext too short: %d < %d", len(data), nonceSize)
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	cryptoLogger.Debug("DecryptWithKey: extracted nonce (size: %d) and ciphertext (size: %d)", len(nonce), len(ciphertext))

	base64Encoded, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		cryptoLogger.LogError("GCM Open (AES decrypt)", err)
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
	result, err := EncryptWithKey(text, database.EncryptionKey)
	if err != nil {
		cryptoLogger.LogError("EncryptWithMasterKey", err)
		return "", err
	}
	cryptoLogger.Debug("EncryptWithMasterKey: successfully encrypted with master key, result length: %d", len(result))
	return result, nil
}

func DecryptWithMasterKey(encryptedText string) (string, error) {
	cryptoLogger.Debug("DecryptWithMasterKey: starting decryption with master key, input length: %d", len(encryptedText))
	result, err := DecryptWithKey(encryptedText, database.EncryptionKey)
	if err != nil {
		cryptoLogger.LogError("DecryptWithMasterKey", err)
		return "", err
	}
	cryptoLogger.Debug("DecryptWithMasterKey: successfully decrypted with master key, result length: %d", len(result))
	return result, nil
}

func GenerateUserAESKey() ([]byte, error) {
	cryptoLogger.Debug("GenerateUserAESKey: generating new 32-byte AES key")
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		cryptoLogger.LogError("GenerateUserAESKey", err)
		return nil, fmt.Errorf("failed to generate user AES key: %w", err)
	}
	cryptoLogger.Debug("GenerateUserAESKey: successfully generated 32-byte AES key")
	return key, nil
}

// DecryptUserKey decrypts an encrypted user key and returns the raw AES key
// This function is separated from database access for better modularity
func DecryptUserKey(encryptedKey string) ([]byte, error) {
	cryptoLogger.Debug("DecryptUserKey: starting decryption, encryptedKey length: %d", len(encryptedKey))

	if encryptedKey == "" {
		cryptoLogger.Error("DecryptUserKey: encrypted key is empty")
		return nil, fmt.Errorf("encrypted key is empty")
	}

	cryptoLogger.Debug("DecryptUserKey: decrypting user key with master key")
	decryptedKeyHex, err := DecryptWithMasterKey(encryptedKey)
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

	// Step 1: Base64 encode the raw text first
	base64Encoded := base64.StdEncoding.EncodeToString([]byte(text))
	cryptoLogger.Debug("EncryptWithAAD: base64 encoded, length: %d", len(base64Encoded))

	// Step 2: AES-GCM encrypt with AAD
	block, err := aes.NewCipher(key)
	if err != nil {
		cryptoLogger.LogError("AES NewCipher", err)
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		cryptoLogger.LogError("GCM NewGCM", err)
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		cryptoLogger.LogError("Random nonce generation", err)
		return "", err
	}
	cryptoLogger.Debug("EncryptWithAAD: generated nonce, size: %d", len(nonce))

	// Use AAD in Seal operation
	ciphertext := gcm.Seal(nonce, nonce, []byte(base64Encoded), aad)
	cryptoLogger.Debug("EncryptWithAAD: AES-GCM encryption with AAD completed, ciphertext length: %d", len(ciphertext))

	// Step 3: Base64 encode the encrypted data for storage
	result := base64.StdEncoding.EncodeToString(ciphertext)
	cryptoLogger.Debug("EncryptWithAAD: final base64 encoding completed, result length: %d", len(result))
	return result, nil
}

// DecryptWithAAD decrypts data with Additional Authenticated Data (AAD)
// AAD must match the AAD used during encryption, otherwise decryption fails
// Flow: encrypted data -> base64 decode -> AES-GCM decrypt (with AAD) -> base64 decode -> raw data
func DecryptWithAAD(encryptedText string, key []byte, aad []byte) (string, error) {
	cryptoLogger.Debug("DecryptWithAAD: starting decryption, input length: %d, AAD length: %d", len(encryptedText), len(aad))

	// Step 1: Base64 decode the stored encrypted data
	data, err := base64.StdEncoding.DecodeString(encryptedText)
	if err != nil {
		cryptoLogger.LogError("Base64 decode (step 1)", err)
		return "", err
	}
	cryptoLogger.Debug("DecryptWithAAD: base64 decoded, data length: %d", len(data))

	// Step 2: AES-GCM decrypt with AAD
	block, err := aes.NewCipher(key)
	if err != nil {
		cryptoLogger.LogError("AES NewCipher", err)
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		cryptoLogger.LogError("GCM NewGCM", err)
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		cryptoLogger.Error("DecryptWithAAD: ciphertext too short: %d < %d", len(data), nonceSize)
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	cryptoLogger.Debug("DecryptWithAAD: extracted nonce (size: %d) and ciphertext (size: %d)", len(nonce), len(ciphertext))

	// Use AAD in Open operation - if AAD doesn't match, this will fail
	base64Encoded, err := gcm.Open(nil, nonce, ciphertext, aad)
	if err != nil {
		cryptoLogger.LogError("GCM Open (AES decrypt with AAD)", err)
		cryptoLogger.Error("DecryptWithAAD: AAD verification failed - potential security issue (wrong user/todo/field)")
		return "", fmt.Errorf("decryption failed: AAD mismatch or invalid ciphertext: %w", err)
	}
	cryptoLogger.Debug("DecryptWithAAD: AES-GCM decryption with AAD completed, base64 string length: %d", len(base64Encoded))

	// Step 3: Base64 decode to get original text
	plaintext, err := base64.StdEncoding.DecodeString(string(base64Encoded))
	if err != nil {
		cryptoLogger.LogError("Base64 decode (step 2)", err)
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	cryptoLogger.Debug("DecryptWithAAD: final base64 decode completed, plaintext length: %d", len(plaintext))

	return string(plaintext), nil
}
