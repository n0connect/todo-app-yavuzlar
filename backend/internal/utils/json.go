package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	// MaxRequestBodySize limits request body size to prevent DoS attacks
	// 1 MB should be sufficient for all API endpoints
	MaxRequestBodySize = 1 << 20 // 1 MB
)

// DecodeJSONRequest decodes a JSON request body into the provided struct
// SECURITY: Limits request body size to prevent DoS attacks
func DecodeJSONRequest(r *http.Request, v interface{}) error {
	// Limit request body size to prevent memory exhaustion DoS
	limitedReader := io.LimitReader(r.Body, MaxRequestBodySize)
	decoder := json.NewDecoder(limitedReader)
	decoder.DisallowUnknownFields() // Security: reject unknown fields
	
	err := decoder.Decode(v)
	if err != nil {
		// Check if error is due to size limit (EOF after reading max bytes)
		if err == io.EOF {
			return fmt.Errorf("request body too large (max %d bytes)", MaxRequestBodySize)
		}
		return err
	}
	return nil
}

// EncodeJSONResponse writes a JSON response with the given status code
func EncodeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
