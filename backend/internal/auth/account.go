package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/utils"

	"golang.org/x/crypto/argon2"
)

var accountLogger = utils.NewLogger("ACCOUNT")

const (
	// Argon2id parameters
	argon2Memory      = 64 * 1024 // 64 MB
	argon2Time        = 3
	argon2Parallelism = 2
	argon2SaltLength  = 16
	argon2KeyLength   = 32
)

// ComputeAccountLookup computes HMAC-SHA256(pepper, accountNumber) for fast database lookup
// Returns hex-encoded string
func ComputeAccountLookup(accountNumber string) (string, error) {
	pepper := config.GetAccountLookupPepper()
	if len(pepper) == 0 {
		return "", errors.New("ACCOUNT_LOOKUP_PEPPER not configured")
	}

	mac := hmac.New(sha256.New, pepper)
	mac.Write([]byte(accountNumber))
	lookup := hex.EncodeToString(mac.Sum(nil))

	accountLogger.Debug("ComputeAccountLookup: computed lookup for account number (masked)")
	return lookup, nil
}

// HashAccountNumber hashes account number using Argon2id
// Returns encoded hash string with parameters
func HashAccountNumber(accountNumber string) (string, error) {
	// Generate random salt
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash with Argon2id
	hash := argon2.IDKey([]byte(accountNumber), salt, argon2Time, argon2Memory, argon2Parallelism, argon2KeyLength)

	// Encode: format = "argon2id$m=memory$t=time$p=parallelism$salt$hash"
	// Using hex encoding for both salt and hash
	encoded := fmt.Sprintf("argon2id$m=%d$t=%d$p=%d$%s$%s",
		argon2Memory,
		argon2Time,
		argon2Parallelism,
		hex.EncodeToString(salt),
		hex.EncodeToString(hash))

	accountLogger.Debug("HashAccountNumber: hashed account number (masked)")
	return encoded, nil
}

// VerifyAccountNumberHash verifies account number against Argon2id hash
func VerifyAccountNumberHash(encodedHash, accountNumber string) error {
	// Parse encoded hash format: "argon2id$m=65536$t=3$p=2$<salt_hex>$<hash_hex>"
	var memory, time, parallelism uint32
	var saltHex, hashHex string

	// Split by $ to parse components
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[0] != "argon2id" {
		return fmt.Errorf("invalid hash format: expected 6 parts starting with 'argon2id'")
	}

	// Parse memory
	if _, err := fmt.Sscanf(parts[1], "m=%d", &memory); err != nil {
		return fmt.Errorf("invalid memory parameter: %w", err)
	}

	// Parse time
	if _, err := fmt.Sscanf(parts[2], "t=%d", &time); err != nil {
		return fmt.Errorf("invalid time parameter: %w", err)
	}

	// Parse parallelism
	if _, err := fmt.Sscanf(parts[3], "p=%d", &parallelism); err != nil {
		return fmt.Errorf("invalid parallelism parameter: %w", err)
	}

	saltHex = parts[4]
	hashHex = parts[5]

	// Validate salt and hash are not empty
	if saltHex == "" {
		return fmt.Errorf("invalid hash format: salt is empty")
	}
	if hashHex == "" {
		return fmt.Errorf("invalid hash format: hash is empty")
	}

	// Decode salt and hash
	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return fmt.Errorf("invalid salt encoding: %w", err)
	}

	expectedHash, err := hex.DecodeString(hashHex)
	if err != nil {
		return fmt.Errorf("invalid hash encoding: %w", err)
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(accountNumber), salt, time, memory, uint8(parallelism), uint32(len(expectedHash)))

	// Constant-time comparison
	if !hmac.Equal(computedHash, expectedHash) {
		return errors.New("account number hash mismatch")
	}

	accountLogger.Debug("VerifyAccountNumberHash: verified account number hash")
	return nil
}

// PerformDummyHashVerification performs a dummy Argon2id hash verification
// to prevent timing attacks by making all login attempts take similar time
// regardless of whether the user exists or not
// This is exported so handlers can use it for timing attack prevention
func PerformDummyHashVerification(accountNumber string) {
	// Use a dummy hash with same format as real hashes
	// This ensures constant-time operation similar to real verification
	dummyHash := "$argon2id$v=19$m=67108864,t=3,p=2$dummysalt123456$dummyhash123456789012345678901234567890"
	_ = VerifyAccountNumberHash(dummyHash, accountNumber)
	// Ignore error - this is just for timing protection
}
