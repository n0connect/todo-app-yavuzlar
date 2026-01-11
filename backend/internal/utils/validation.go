package utils

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var validationLogger = NewLogger("VALIDATION")

const (
	MaxTitleLength = 1000
	MinTitleLength = 1
)

// ========================================
// INPUT VALIDATION - OUTPUT ENCODING APPROACH
// ========================================
// Security Strategy: Accept user input as-is, encode on output
// XSS Prevention: Frontend uses textContent (auto-escapes HTML)
// Backend stores data as-is, encrypted with AES-256-GCM
// This allows emojis, special chars, any Unicode text

var (
	// UUID: Only alphanumeric, exactly 24 characters
	uuidPattern = regexp.MustCompile(`^[a-zA-Z0-9]{24}$`)

	// Base64: Only valid base64 characters
	base64Pattern = regexp.MustCompile(`^[A-Za-z0-9+/]*={0,2}$`)
)

func ValidateUUID(uuid string) bool {
	if len(uuid) != 24 {
		return false
	}
	return uuidPattern.MatchString(uuid)
}

func GenerateSecureUUID() (string, error) {
	// Use 24 bytes for better entropy distribution
	randomBytes := make([]byte, 24)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// 62 characters: A-Z, a-z, 0-9
	alphanumeric := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, 24)

	for i := 0; i < 24; i++ {
		// Rejection sampling for uniform distribution
		// 256 % 62 = 8, reject values >= 248 to avoid bias
		b := randomBytes[i]
		for b >= 248 {
			extraByte := make([]byte, 1)
			if _, err := rand.Read(extraByte); err != nil {
				return "", fmt.Errorf("failed to generate random bytes: %w", err)
			}
			b = extraByte[0]
		}
		result[i] = alphanumeric[b%62]
	}

	validationLogger.Debug("GenerateSecureUUID: generated 24-char alphanumeric UUID with CSPRNG")
	return string(result), nil
}

// ValidateTitle validates todo title with minimal restrictions
// Security: XSS is prevented by output encoding (textContent), not input filtering
// Allows: All Unicode characters including emojis, <, >, quotes, etc.
// Rejects: Only invalid UTF-8, null bytes, empty strings, and excessive length
func ValidateTitle(title string) (string, error) {
	validationLogger.Debug("ValidateTitle: validating input, length: %d", len(title))

	// Step 1: Check for null bytes (security)
	if strings.Contains(title, "\x00") {
		validationLogger.Warn("ValidateTitle: null byte detected")
		return "", fmt.Errorf("invalid characters")
	}

	// Step 2: Validate UTF-8 encoding
	if !utf8.ValidString(title) {
		validationLogger.Warn("ValidateTitle: invalid UTF-8 encoding")
		return "", fmt.Errorf("invalid encoding")
	}

	// Step 3: Trim whitespace
	title = strings.TrimSpace(title)

	if title == "" {
		validationLogger.Warn("ValidateTitle: empty title rejected")
		return "", fmt.Errorf("title cannot be empty")
	}

	// Step 4: Check length (in runes, not bytes - for Unicode support)
	runeCount := utf8.RuneCountInString(title)
	if runeCount < MinTitleLength {
		return "", fmt.Errorf("title too short")
	}
	if runeCount > MaxTitleLength {
		// Truncate to max length
		runes := []rune(title)
		title = string(runes[:MaxTitleLength])
		validationLogger.Debug("ValidateTitle: truncated to %d characters", MaxTitleLength)
	}

	validationLogger.Debug("ValidateTitle: validation passed, final length: %d runes", utf8.RuneCountInString(title))
	return title, nil
}

// ValidateBase64Data validates base64 input
func ValidateBase64Data(data string, maxLen int) error {
	if data == "" {
		return fmt.Errorf("empty data")
	}
	if len(data) > maxLen {
		return fmt.Errorf("data too large")
	}

	// Whitelist: Only valid base64 characters
	if !base64Pattern.MatchString(data) {
		return fmt.Errorf("invalid base64 format")
	}

	return nil
}
