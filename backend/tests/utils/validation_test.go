package utils

import (
	"regexp"
	"strings"
	"testing"

	"todo-app-backend/internal/utils"
)

func TestGenerateAccountNumber_LengthAndCharset(t *testing.T) {
	// Test that generated account numbers are exactly 43 characters
	// and match base64url charset pattern
	pattern := regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

	for i := 0; i < 100; i++ {
		accountNumber, err := utils.GenerateAccountNumber()
		if err != nil {
			t.Fatalf("GenerateAccountNumber() failed: %v", err)
		}

		// Check length
		if len(accountNumber) != 43 {
			t.Errorf("GenerateAccountNumber() length = %d, want 43", len(accountNumber))
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
			name:    "valid base64url 43 chars",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefg",
			want:    true,
			comment: "Valid 43-char base64url string",
		},
		{
			name:    "valid with numbers",
			input:   strings.Repeat("1", 43),
			want:    true,
			comment: "Valid with numbers",
		},
		{
			name:    "valid with underscore",
			input:   strings.Repeat("A", 42) + "_",
			want:    true,
			comment: "Valid with underscore",
		},
		{
			name:    "valid with dash",
			input:   strings.Repeat("B", 42) + "-",
			want:    true,
			comment: "Valid with dash",
		},
		{
			name:    "valid mixed case",
			input:   strings.Repeat("aB", 21) + "c",
			want:    true,
			comment: "Valid mixed case",
		},
		{
			name:    "too short",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdef",
			want:    false,
			comment: "Too short",
		},
		{
			name:    "too long",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghij",
			want:    false,
			comment: "Too long",
		},
		{
			name:    "invalid char plus",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abc+defg",
			want:    false,
			comment: "Contains '+' (invalid for base64url)",
		},
		{
			name:    "invalid char slash",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abc/defg",
			want:    false,
			comment: "Contains '/' (invalid for base64url)",
		},
		{
			name:    "invalid char equals",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abc=defg",
			want:    false,
			comment: "Contains '=' (invalid for base64url)",
		},
		{
			name:    "invalid char space",
			input:   "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abc defg",
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
			name:     "normal 43 char",
			input:    "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefg",
			expected: "***************************************defg",
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
