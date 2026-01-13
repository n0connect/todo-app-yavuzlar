// Package models/requests contains HTTP request/response DTOs only.
// These models are NOT database entities and are NOT imported in database migrations.
// They are only used for API request/response serialization.

package models

type LoginRequest struct {
	AccountNumber string `json:"account_number"`
}

type RegisterRequest struct {
	Confirm      bool   `json:"confirm"`       // false = generate AccountNumber only, true = create account
	PendingToken string `json:"pending_token"` // required when confirm=true (contains the AccountNumber)
}

type LoginResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"` // JWT token
}

type RegisterResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	AccountNumber string `json:"account_number"`
	Token         string `json:"token,omitempty"`         // Only sent when confirm=true
	PendingToken  string `json:"pending_token,omitempty"` // Only sent when confirm=false
	Confirmed     bool   `json:"confirmed"`               // true if account was created
}

type TodoRequest struct {
	Title     string   `json:"title"`
	Completed bool     `json:"completed"`
	Tags      []string `json:"tags,omitempty"`
	DueDate   string   `json:"due_date,omitempty"` // ISO 8601 format
	Priority  string   `json:"priority,omitempty"` // low, medium, high
}
