package utils

import (
	"regexp"
	"testing"

	"todo-app-backend/internal/utils"
)

func TestGenerateAccountNumber_LengthAndCharset(t *testing.T) {
	// Test that generated account numbers are exactly 32 characters
	// and match base64url charset pattern
	pattern := regexp.MustCompile(`^[A-Za-z0-9_-]{32}$`)

	for i := 0; i < 100; i++ {
		accountNumber, err := utils.GenerateAccountNumber()
		if err != nil {
			t.Fatalf("GenerateAccountNumber() failed: %v", err)
		}

		// Check length
		if len(accountNumber) != 32 {
			t.Errorf("GenerateAccountNumber() length = %d, want 32", len(accountNumber))
		}

		// Check charset
		if !pattern.MatchString(accountNumber) {
			t.Errorf("GenerateAccountNumber() = %q, does not match base64url pattern", accountNumber)
		}
	}
}

func TestValidateAccountNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		comment string
	}{
		{
			name:    "valid base64url 32 chars",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZab12xx",
			want:    true,
			comment: "Valid 32-char base64url string",
		},
		{
			name:    "valid with numbers",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123xx",
			want:    true,
			comment: "Valid with numbers",
		},
		{
			name:    "valid with underscore",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZab_cdx",
			want:    true,
			comment: "Valid with underscore",
		},
		{
			name:    "valid with dash",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZab-cdx",
			want:    true,
			comment: "Valid with dash",
		},
		{
			name:    "valid mixed case",
			input:   "AbCdEfGhIjKlMnOpQrStUvWxYz123456",
			want:    true,
			comment: "Valid mixed case",
		},
		{
			name:    "too short",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZabc",
			want:    false,
			comment: "31 characters (too short)",
		},
		{
			name:    "too long",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZabcde",
			want:    false,
			comment: "33 characters (too long)",
		},
		{
			name:    "invalid char plus",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZab+c",
			want:    false,
			comment: "Contains '+' (invalid for base64url)",
		},
		{
			name:    "invalid char slash",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZab/c",
			want:    false,
			comment: "Contains '/' (invalid for base64url)",
		},
		{
			name:    "invalid char equals",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZab=c",
			want:    false,
			comment: "Contains '=' (invalid for base64url)",
		},
		{
			name:    "invalid char space",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZab c",
			want:    false,
			comment: "Contains space (invalid)",
		},
		{
			name:    "empty string",
			input:   "",
			want:    false,
			comment: "Empty string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.ValidateAccountNumber(tt.input)
			if got != tt.want {
				t.Errorf("ValidateAccountNumber(%q) = %v, want %v (%s)", tt.input, got, tt.want, tt.comment)
			}
		})
	}
}

func TestMaskAccountNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal 32 char",
			input:    "ABCDEFGHIJKLMNOPQRSTUVWXYZab12xx",
			expected: "****************************12xx",
		},
		{
			name:     "short string",
			input:    "ABC",
			expected: "****",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "****",
		},
		{
			name:     "exactly 4 chars",
			input:    "ABCD",
			expected: "****",
		},
		{
			name:     "5 chars",
			input:    "ABCDE",
			expected: "*BCDE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.MaskAccountNumber(tt.input)
			if got != tt.expected {
				t.Errorf("MaskAccountNumber(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
