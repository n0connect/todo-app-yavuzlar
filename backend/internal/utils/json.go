package utils

import (
	"encoding/json"
	"net/http"
)

// DecodeJSONRequest decodes a JSON request body into the provided struct
func DecodeJSONRequest(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields() // Security: reject unknown fields
	return decoder.Decode(v)
}

// EncodeJSONResponse writes a JSON response with the given status code
func EncodeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
