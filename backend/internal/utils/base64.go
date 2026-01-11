package utils

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
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

func DecodeBase64Request(r *http.Request, v interface{}, maxLen int) error {
	var requestBody map[string]string
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		return fmt.Errorf("invalid request body")
	}

	dataBase64, ok := requestBody["data"]
	if !ok {
		return fmt.Errorf("missing data field")
	}

	dataJSON, err := validateBase64(dataBase64, maxLen)
	if err != nil {
		return fmt.Errorf("invalid base64")
	}

	if err := json.Unmarshal(dataJSON, v); err != nil {
		return fmt.Errorf("invalid json")
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
