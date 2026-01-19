package handlers_test

import (
	"encoding/json"
	"testing"

	"todo-app-backend/internal/models"
)

func TestLoginResponse_NoAccountNumber(t *testing.T) {
	// This test verifies that LoginResponse does not include account_number field
	// Note: This is a unit test for the response structure, not a full integration test

	response := models.LoginResponse{
		Success: true,
		Message: "Login successful",
		Token:   "test-token",
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("json.Unmarshal() failed: %v", err)
	}

	// Verify account_number is NOT in response
	if _, exists := result["account_number"]; exists {
		t.Error("LoginResponse should not include account_number field")
	}

	// Verify required fields are present
	if result["success"] != true {
		t.Error("LoginResponse should include success field")
	}
	if result["message"] == nil {
		t.Error("LoginResponse should include message field")
	}
	if result["token"] == nil {
		t.Error("LoginResponse should include token field")
	}
}

func TestRegisterResponse_Structure(t *testing.T) {
	// Test RegisterResponse structure
	response := models.RegisterResponse{
		Success:       true,
		Message:       "Account created successfully",
		AccountNumber: "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefg",
		Token:         "test-token",
		Confirmed:     true,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("json.Marshal() failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("json.Unmarshal() failed: %v", err)
	}

	// Verify all expected fields
	requiredFields := []string{"success", "message", "account_number", "token", "confirmed"}
	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			t.Errorf("RegisterResponse missing required field: %s", field)
		}
	}
}
