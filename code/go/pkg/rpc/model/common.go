// Package model provides data models for the RPC API.
package model

// Response is the standard API response format.
// All API responses use this JSON structure.
type Response struct {
	Success bool        `json:"success"`           // Indicates if the request was successful
	Data    interface{} `json:"data,omitempty"`    // Response data (omitted if empty or null)
	Error   string      `json:"error,omitempty"`   // Error code or message (omitted on success)
	Message string      `json:"message,omitempty"` // Optional message or error details
}

// Pagination represents pagination metadata.
type Pagination struct {
	Page       int `json:"page"`       // Current page number
	PageSize   int `json:"pageSize"`   // Number of items per page
	TotalPages int `json:"totalPages"` // Total number of pages
}
