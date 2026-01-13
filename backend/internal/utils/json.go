package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// MaxRequestBodySize limits request body size to prevent DoS attacks
	// 1 MB should be sufficient for all API endpoints
	MaxRequestBodySize = 1 << 20 // 1 MB

	// MaxJSONDepth limits the maximum nesting depth of JSON objects/arrays
	// Analysis of current endpoints:
	//   - LoginRequest: depth 1
	//   - RegisterRequest: depth 1
	//   - TodoRequest: depth 2 (Tags array)
	// Maximum actual depth: 2
	// Security limit: 2 (matches actual usage, strict security - no buffer for unnecessary nesting)
	MaxJSONDepth = 2
)

// CheckJSONDepth checks the maximum nesting depth of a JSON document
// Returns the maximum depth and any error encountered
func CheckJSONDepth(data []byte) (int, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	depth := 0
	maxDepth := 0

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("invalid JSON: %w", err)
		}

		// Track depth based on delimiter tokens
		switch token {
		case json.Delim('{'), json.Delim('['):
			depth++
			if depth > maxDepth {
				maxDepth = depth
			}
			if depth > MaxJSONDepth {
				return maxDepth, fmt.Errorf("JSON nesting depth exceeds maximum (%d levels)", MaxJSONDepth)
			}
		case json.Delim('}'), json.Delim(']'):
			depth--
			if depth < 0 {
				return 0, fmt.Errorf("invalid JSON: unmatched closing delimiter")
			}
		}
	}

	return maxDepth, nil
}

// DecodeJSONRequest decodes a JSON request body into the provided struct
// SECURITY: Limits request body size and nesting depth to prevent DoS attacks
func DecodeJSONRequest(r *http.Request, v interface{}) error {
	// Limit request body size to prevent memory exhaustion DoS
	limitedReader := io.LimitReader(r.Body, MaxRequestBodySize)

	// Read entire body to check depth (we need to read it anyway for decoding)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Check if body exceeds size limit
	if len(bodyBytes) >= MaxRequestBodySize {
		return fmt.Errorf("request body too large (max %d bytes)", MaxRequestBodySize)
	}

	// Check JSON depth before decoding
	if _, err := CheckJSONDepth(bodyBytes); err != nil {
		return err
	}

	// Decode JSON into struct
	decoder := json.NewDecoder(bytes.NewReader(bodyBytes))
	decoder.DisallowUnknownFields() // Security: reject unknown fields

	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	return nil
}

// EncodeJSONResponse writes a JSON response with the given status code
func EncodeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// WriteAPIError writes a generic error message to the user while logging detailed information
// SECURITY: publicMsg is shown to users (must be generic), internalErr is only logged
// This prevents information leakage while maintaining detailed logs for debugging
func WriteAPIError(w http.ResponseWriter, statusCode int, publicMsg string, logger interface{ LogError(string, error); Warn(string, ...interface{}) }, internalErr error, logContext string) {
	// Log detailed error information (for debugging/auditing)
	if logger != nil && internalErr != nil {
		logger.LogError(logContext, internalErr)
	} else if logger != nil {
		logger.Warn("%s: %s", logContext, publicMsg)
	}

	// Write generic error message to user
	http.Error(w, publicMsg, statusCode)
}
