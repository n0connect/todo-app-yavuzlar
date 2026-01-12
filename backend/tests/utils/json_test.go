package utils

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"

	"todo-app-backend/internal/models"
	"todo-app-backend/internal/utils"
)

func TestDecodeJSONRequest_ValidDepth(t *testing.T) {
	// Test with valid depth (depth 2 - TodoRequest with Tags array)
	jsonBody := `{
		"title": "Test todo",
		"completed": false,
		"tags": ["tag1", "tag2"],
		"priority": "high"
	}`

	req := httptest.NewRequest("POST", "/api/v1/todos", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	var todoReq models.TodoRequest
	err := utils.DecodeJSONRequest(req, &todoReq)
	if err != nil {
		t.Fatalf("DecodeJSONRequest() failed with valid depth: %v", err)
	}

	if todoReq.Title != "Test todo" {
		t.Errorf("Expected title 'Test todo', got '%s'", todoReq.Title)
	}
	if len(todoReq.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(todoReq.Tags))
	}
}

func TestDecodeJSONRequest_ExceedsMaxDepth(t *testing.T) {
	// Create deeply nested JSON (depth > MaxJSONDepth = 2)
	// Build a JSON with 5 levels of nesting (should be rejected)
	nestedJSON := strings.Repeat(`{"a":`, 5) + `"value"` + strings.Repeat(`}`, 5)

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewBufferString(nestedJSON))
	req.Header.Set("Content-Type", "application/json")

	var result map[string]interface{}
	err := utils.DecodeJSONRequest(req, &result)
	if err == nil {
		t.Error("DecodeJSONRequest() should have failed with deeply nested JSON")
	}

	if !strings.Contains(err.Error(), "nesting depth exceeds maximum") {
		t.Errorf("Expected depth limit error, got: %v", err)
	}
}

func TestDecodeJSONRequest_ValidShallowDepth(t *testing.T) {
	// Test with shallow depth (depth 1 - LoginRequest)
	jsonBody := `{
		"account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq"
	}`

	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	var loginReq models.LoginRequest
	err := utils.DecodeJSONRequest(req, &loginReq)
	if err != nil {
		t.Fatalf("DecodeJSONRequest() failed with shallow depth: %v", err)
	}

	if loginReq.AccountNumber != "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq" {
		t.Errorf("Expected account_number, got '%s'", loginReq.AccountNumber)
	}
}

func TestDecodeJSONRequest_ArrayDepth(t *testing.T) {
	// Test with array nesting (depth 2)
	jsonBody := `{
		"title": "Test",
		"tags": ["tag1", "tag2"]
	}`

	req := httptest.NewRequest("POST", "/api/v1/todos", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	var todoReq models.TodoRequest
	err := utils.DecodeJSONRequest(req, &todoReq)
	if err != nil {
		t.Fatalf("DecodeJSONRequest() failed with array depth: %v", err)
	}

	if len(todoReq.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(todoReq.Tags))
	}
}

func TestDecodeJSONRequest_UnknownFields(t *testing.T) {
	// Test that unknown fields are rejected
	jsonBody := `{
		"account_number": "x1_Op2u1bEokj79-HKY5V_EmOe1ATTDq",
		"malicious_field": "attack"
	}`

	req := httptest.NewRequest("POST", "/api/v1/login", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	var loginReq models.LoginRequest
	err := utils.DecodeJSONRequest(req, &loginReq)
	if err == nil {
		t.Error("DecodeJSONRequest() should have rejected unknown fields")
	}
}

func TestDecodeJSONRequest_SizeLimit(t *testing.T) {
	// Test that large request bodies are rejected
	largeBody := strings.Repeat("A", utils.MaxRequestBodySize+1)

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewBufferString(largeBody))
	req.Header.Set("Content-Type", "application/json")

	var result map[string]interface{}
	err := utils.DecodeJSONRequest(req, &result)
	if err == nil {
		t.Error("DecodeJSONRequest() should have rejected large request body")
	}

	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("Expected size limit error, got: %v", err)
	}
}

func TestDecodeJSONRequest_InvalidJSON(t *testing.T) {
	// Test with invalid JSON
	invalidJSON := `{invalid json}`

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewBufferString(invalidJSON))
	req.Header.Set("Content-Type", "application/json")

	var result map[string]interface{}
	err := utils.DecodeJSONRequest(req, &result)
	if err == nil {
		t.Error("DecodeJSONRequest() should have failed with invalid JSON")
	}
}

func TestDecodeJSONRequest_EmptyBody(t *testing.T) {
	// Test with empty body
	req := httptest.NewRequest("POST", "/api/v1/register", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")

	var registerReq models.RegisterRequest
	err := utils.DecodeJSONRequest(req, &registerReq)
	// Empty body should be handled gracefully (may succeed or fail depending on struct)
	// This test just ensures no panic occurs
	if err != nil {
		t.Logf("DecodeJSONRequest() with empty body returned error (expected): %v", err)
	}
}

func TestDecodeJSONRequest_MaxDepthBoundary(t *testing.T) {
	// Test with exactly MaxJSONDepth (2) levels - should succeed
	nestedJSON := strings.Repeat(`{"a":`, utils.MaxJSONDepth) + `"value"` + strings.Repeat(`}`, utils.MaxJSONDepth)

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewBufferString(nestedJSON))
	req.Header.Set("Content-Type", "application/json")

	var result map[string]interface{}
	err := utils.DecodeJSONRequest(req, &result)
	// Should succeed at exactly MaxJSONDepth (2)
	if err != nil {
		t.Errorf("DecodeJSONRequest() at max depth (2) should succeed, got error: %v", err)
	}
}

func TestDecodeJSONRequest_MaxDepthPlusOne(t *testing.T) {
	// Test with MaxJSONDepth + 1 levels (3 levels, should fail)
	nestedJSON := strings.Repeat(`{"a":`, utils.MaxJSONDepth+1) + `"value"` + strings.Repeat(`}`, utils.MaxJSONDepth+1)

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewBufferString(nestedJSON))
	req.Header.Set("Content-Type", "application/json")

	var result map[string]interface{}
	err := utils.DecodeJSONRequest(req, &result)
	if err == nil {
		t.Error("DecodeJSONRequest() should have failed with MaxJSONDepth+1 levels (3 levels)")
	}

	if !strings.Contains(err.Error(), "nesting depth exceeds maximum") {
		t.Errorf("Expected depth limit error, got: %v", err)
	}
}

func TestDecodeJSONRequest_Depth3_ShouldFail(t *testing.T) {
	// Test with depth 3 (should fail, max is 2)
	nestedJSON := `{"a": {"b": {"c": "value"}}}`

	req := httptest.NewRequest("POST", "/api/v1/test", bytes.NewBufferString(nestedJSON))
	req.Header.Set("Content-Type", "application/json")

	var result map[string]interface{}
	err := utils.DecodeJSONRequest(req, &result)
	if err == nil {
		t.Error("DecodeJSONRequest() should have failed with depth 3")
	}

	if !strings.Contains(err.Error(), "nesting depth exceeds maximum") {
		t.Errorf("Expected depth limit error, got: %v", err)
	}
}

func TestDecodeJSONRequest_Depth2_ShouldSucceed(t *testing.T) {
	// Test with depth 2 (should succeed, matches TodoRequest with Tags array)
	jsonBody := `{"tags": ["tag1", "tag2"]}`

	req := httptest.NewRequest("POST", "/api/v1/todos", bytes.NewBufferString(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	var todoReq models.TodoRequest
	err := utils.DecodeJSONRequest(req, &todoReq)
	if err != nil {
		t.Errorf("DecodeJSONRequest() should succeed with depth 2, got error: %v", err)
	}

	if len(todoReq.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(todoReq.Tags))
	}
}
