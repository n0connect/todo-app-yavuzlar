package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func validateBase64(data string, maxLen int) ([]byte, error) {
	if maxLen <= 0 {
		return nil, fmt.Errorf("invalid base64 limit")
	}
	if data == "" {
		return nil, fmt.Errorf("empty base64 data")
	}
	if len(data) > maxLen {
		return nil, fmt.Errorf("base64 data too large")
	}

	decoded, err := base64.StdEncoding.Strict().DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("invalid base64")
	}
	return decoded, nil
}

// DecodeBase64Request decodes a base64-encoded JSON request body
// SECURITY: Limits request body size, nesting depth, and rejects unknown fields
func DecodeBase64Request(r *http.Request, v interface{}, maxLen int) error {
	// Limit request body size
	limitedReader := io.LimitReader(r.Body, MaxRequestBodySize)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	if len(bodyBytes) >= MaxRequestBodySize {
		return fmt.Errorf("request body too large (max %d bytes)", MaxRequestBodySize)
	}

	// Decode outer JSON (contains base64 data)
	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	decoder.DisallowUnknownFields() // Security: reject unknown fields

	var requestBody map[string]string
	if err := decoder.Decode(&requestBody); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}

	// Check depth of outer JSON
	if _, err := CheckJSONDepth(bodyBytes); err != nil {
		return fmt.Errorf("outer JSON depth check failed: %w", err)
	}

	dataBase64, ok := requestBody["data"]
	if !ok {
		return fmt.Errorf("missing data field")
	}

	// Decode base64 data
	dataJSON, err := validateBase64(dataBase64, maxLen)
	if err != nil {
		return fmt.Errorf("invalid base64: %w", err)
	}

	// Check depth of inner JSON (base64-decoded data)
	if _, err := CheckJSONDepth(dataJSON); err != nil {
		return fmt.Errorf("inner JSON depth check failed: %w", err)
	}

	// Decode inner JSON
	if err := json.Unmarshal(dataJSON, v); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}

	return nil
}

func EncodeBase64Response(w http.ResponseWriter, data interface{}, statusCode int) {
	responseJSON, _ := json.Marshal(data)
	responseBase64 := base64.StdEncoding.EncodeToString(responseJSON)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(fmt.Sprintf(`{"data":"%s"}`, responseBase64)))
}
