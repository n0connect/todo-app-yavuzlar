package auth

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/cryptoengine"
	"todo-app-backend/internal/utils"
)

var accountLogger = utils.NewLogger("ACCOUNT")

type argon2Params struct {
	memoryKiB  uint32
	time       uint32
	parallel   uint32
	saltLength uint32
	hashLength uint32
}

var (
	dummyHashOnce sync.Once
	dummyHash     string
	dummyHashErr  error
)

func getArgon2Params() (argon2Params, error) {
	memory := config.GetArgon2MemoryKiB()
	timeCost := config.GetArgon2Time()
	parallel := config.GetArgon2Parallelism()
	saltLen := config.GetArgon2SaltLength()
	hashLen := config.GetArgon2HashLength()

	if memory <= 0 || timeCost <= 0 || parallel <= 0 || saltLen <= 0 || hashLen <= 0 {
		return argon2Params{}, fmt.Errorf("argon2 parameters not configured")
	}

	return argon2Params{
		memoryKiB:  uint32(memory),
		time:       uint32(timeCost),
		parallel:   uint32(parallel),
		saltLength: uint32(saltLen),
		hashLength: uint32(hashLen),
	}, nil
}

// ComputeAccountLookup computes HMAC-SHA256(pepper, accountNumber) for fast database lookup
// Returns hex-encoded string
func ComputeAccountLookup(accountNumber string) (string, error) {
	pepper := config.GetAccountLookupPepper()
	if len(pepper) == 0 {
		return "", errors.New("ACCOUNT_LOOKUP_PEPPER not configured")
	}
	if len(pepper) < 32 {
		return "", errors.New("ACCOUNT_LOOKUP_PEPPER too short")
	}
	mac, err := cryptoengine.HMACSHA256(pepper, []byte(accountNumber))
	if err != nil {
		return "", err
	}
	lookup := hex.EncodeToString(mac)

	accountLogger.Debug("ComputeAccountLookup: computed lookup for account number (masked)")
	return lookup, nil
}

// HashAccountNumber hashes account number using Argon2id
// Returns encoded hash string with parameters
func HashAccountNumber(accountNumber string) (string, error) {
	params, err := getArgon2Params()
	if err != nil {
		return "", err
	}

	// Generate random salt
	salt, err := cryptoengine.RandomBytes(int(params.saltLength))
	if err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// Hash with Argon2id
	hash, err := cryptoengine.Argon2idKey([]byte(accountNumber), salt, params.time, params.memoryKiB, params.parallel, params.hashLength)
	if err != nil {
		return "", fmt.Errorf("argon2id failed: %w", err)
	}

	// Encode: format = "argon2id$m=memory$t=time$p=parallelism$salt$hash"
	// Using hex encoding for both salt and hash
	encoded := fmt.Sprintf("argon2id$m=%d$t=%d$p=%d$%s$%s",
		params.memoryKiB,
		params.time,
		params.parallel,
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
	computedHash, err := cryptoengine.Argon2idKey([]byte(accountNumber), salt, time, memory, parallelism, uint32(len(expectedHash)))
	if err != nil {
		return fmt.Errorf("argon2id failed: %w", err)
	}

	// Constant-time comparison
	if !cryptoengine.ConstantTimeEqual(computedHash, expectedHash) {
		return errors.New("account number hash mismatch")
	}

	accountLogger.Debug("VerifyAccountNumberHash: verified account number hash")
	return nil
}

func getDummyHash() (string, error) {
	dummyHashOnce.Do(func() {
		params, err := getArgon2Params()
		if err != nil {
			dummyHashErr = err
			return
		}
		salt, err := cryptoengine.RandomBytes(int(params.saltLength))
		if err != nil {
			dummyHashErr = err
			return
		}
		hash, err := cryptoengine.Argon2idKey([]byte("dummy-account"), salt, params.time, params.memoryKiB, params.parallel, params.hashLength)
		if err != nil {
			dummyHashErr = err
			return
		}
		dummyHash = fmt.Sprintf("argon2id$m=%d$t=%d$p=%d$%s$%s",
			params.memoryKiB,
			params.time,
			params.parallel,
			hex.EncodeToString(salt),
			hex.EncodeToString(hash))
	})
	return dummyHash, dummyHashErr
}

// PerformDummyHashVerification performs a dummy Argon2id hash verification
// to prevent timing attacks by making all login attempts take similar time
// regardless of whether the user exists or not
// This is exported so handlers can use it for timing attack prevention
func PerformDummyHashVerification(accountNumber string) {
	dummyHash, err := getDummyHash()
	if err != nil {
		accountLogger.LogError("PerformDummyHashVerification", err)
		return
	}
	_ = VerifyAccountNumberHash(dummyHash, accountNumber)
	// Ignore error - this is just for timing protection
}
