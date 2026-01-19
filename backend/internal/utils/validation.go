package utils

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"todo-app-backend/internal/config"
	"todo-app-backend/internal/cryptoengine"
)

var validationLogger = NewLogger("VALIDATION")

// ========================================
// INPUT VALIDATION - OUTPUT ENCODING APPROACH
// ========================================
// Security Strategy: Accept user input as-is, encode on output
// XSS Prevention: Frontend uses textContent (auto-escapes HTML)
// Backend stores data as-is, encrypted with AES-256-GCM
// This allows emojis, special chars, any Unicode text

var (
	// AccountNumber: Base64URL (no padding) - length derived from byte size
	// Base64URL uses A-Z, a-z, 0-9, _, - (RFC 4648 Section 5)
	accountNumberBytes   = 32
	accountNumberLength  = base64.RawURLEncoding.EncodedLen(accountNumberBytes)
	accountNumberPattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

	// Base64: Only valid base64 characters
	base64Pattern = regexp.MustCompile(`^(?:[A-Za-z0-9+/]{4})*(?:[A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)?$`)
)

func internalIDLength() (int, error) {
	length := config.GetInternalIDLength()
	if length <= 0 {
		return 0, fmt.Errorf("INTERNAL_ID_LENGTH not configured")
	}
	return length, nil
}

func titleLimits() (int, int, error) {
	minLen := config.GetMinTitleLength()
	maxLen := config.GetMaxTitleLength()
	if minLen <= 0 || maxLen <= 0 || minLen > maxLen {
		return 0, 0, fmt.Errorf("invalid title length configuration")
	}
	return minLen, maxLen, nil
}

func maxTagLength() (int, error) {
	maxLen := config.GetMaxTagLength()
	if maxLen <= 0 {
		return 0, fmt.Errorf("MAX_TAG_LENGTH not configured")
	}
	return maxLen, nil
}

// GetMaxTagsPerTodo returns max tags per todo from config.
func GetMaxTagsPerTodo() (int, error) {
	maxTags := config.GetMaxTagsPerTodo()
	if maxTags <= 0 {
		return 0, fmt.Errorf("MAX_TAGS_PER_TODO not configured")
	}
	return maxTags, nil
}

func ValidateUUID(uuid string) bool {
	length, err := internalIDLength()
	if err != nil {
		validationLogger.LogError("ValidateUUID", err)
		return false
	}
	if len(uuid) != length {
		return false
	}
	for i := 0; i < len(uuid); i++ {
		if !isAlphaNumeric(uuid[i]) {
			return false
		}
	}
	return true
}

func GenerateInternalID() (string, error) {
	length, err := internalIDLength()
	if err != nil {
		return "", err
	}

	randomBytes, err := cryptoengine.RandomBytes(length)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// 62 characters: A-Z, a-z, 0-9
	alphanumeric := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		// Rejection sampling for uniform distribution
		// 256 % 62 = 8, reject values >= 248 to avoid bias
		b := randomBytes[i]
		for b >= 248 {
			extraByte, err := cryptoengine.RandomBytes(1)
			if err != nil {
				return "", fmt.Errorf("failed to generate random bytes: %w", err)
			}
			b = extraByte[0]
		}
		result[i] = alphanumeric[b%62]
	}

	validationLogger.Debug("GenerateInternalID: generated alphanumeric UUID with CSPRNG, length=%d", length)
	return string(result), nil
}

// GenerateAccountNumber generates a 256-bit (32 bytes) CSPRNG account number
// Returns base64url-encoded string (43 characters, no padding)
func GenerateAccountNumber() (string, error) {
	// Generate exactly 32 bytes (256 bits) of CSPRNG
	randomBytes, err := cryptoengine.RandomBytes(accountNumberBytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Base64URL encoding (RFC 4648 Section 5, no padding)
	encoded := base64.RawURLEncoding.EncodeToString(randomBytes)

	validationLogger.Debug("GenerateAccountNumber: generated 43-char base64url account number (256-bit CSPRNG)")
	return encoded, nil
}

// ValidateAccountNumber validates account number format
// Must be exactly 43 base64url characters (A-Z, a-z, 0-9, _, -)
func ValidateAccountNumber(accountNumber string) bool {
	if len(accountNumber) != accountNumberLength {
		return false
	}
	return accountNumberPattern.MatchString(accountNumber)
}

// MaskAccountNumber masks account number for logging (shows only last 4 characters)
// Example: "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567" -> "****************************567"
func MaskAccountNumber(accountNumber string) string {
	if len(accountNumber) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(accountNumber)-4) + accountNumber[len(accountNumber)-4:]
}

// ValidateTitle validates todo title with minimal restrictions
// Security: XSS is prevented by output encoding (textContent), not input filtering
// Allows: All Unicode characters including emojis, <, >, quotes, etc.
// Rejects: Only invalid UTF-8, null bytes, empty strings, and excessive length
func ValidateTitle(title string) (string, error) {
	validationLogger.Debug("ValidateTitle: validating input, length: %d", len(title))

	minLen, maxLen, err := titleLimits()
	if err != nil {
		return "", err
	}

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
	if runeCount < minLen {
		return "", fmt.Errorf("title too short")
	}
	if runeCount > maxLen {
		// Truncate to max length
		runes := []rune(title)
		title = string(runes[:maxLen])
		validationLogger.Debug("ValidateTitle: truncated to %d characters", maxLen)
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

// ValidateTag validates todo tag format
// Whitelist: Only alphanumeric characters (A-Z, a-z, 0-9), maximum 6 characters
// Rejects: Special characters, spaces, Unicode, and tags longer than 6 characters
func ValidateTag(tag string) (string, error) {
	validationLogger.Debug("ValidateTag: validating input, length: %d", len(tag))

	maxLen, err := maxTagLength()
	if err != nil {
		return "", err
	}

	// Step 1: Trim whitespace
	tag = strings.TrimSpace(tag)

	// Step 2: Check if empty
	if tag == "" {
		validationLogger.Warn("ValidateTag: empty tag rejected")
		return "", fmt.Errorf("tag cannot be empty")
	}

	// Step 3: Check length (in bytes, since we only allow ASCII)
	if len(tag) > maxLen {
		validationLogger.Warn("ValidateTag: tag too long, length: %d", len(tag))
		return "", fmt.Errorf("tag too long (maximum %d characters)", maxLen)
	}

	// Step 4: Validate format - only alphanumeric characters
	for i := 0; i < len(tag); i++ {
		if !isAlphaNumeric(tag[i]) {
			validationLogger.Warn("ValidateTag: invalid tag format")
			return "", fmt.Errorf("invalid tag format (only letters and numbers allowed)")
		}
	}
	if tag == "" {
		validationLogger.Warn("ValidateTag: invalid tag format")
		return "", fmt.Errorf("invalid tag format (only letters and numbers allowed)")
	}

	validationLogger.Debug("ValidateTag: validation passed, tag: %s", tag)
	return tag, nil
}

func isAlphaNumeric(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
}
