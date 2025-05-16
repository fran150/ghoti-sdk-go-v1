package model

import "fmt"

// GhotiError represents an error from the Ghoti server
type GhotiError struct {
	Code    string
	Message string
}

// Error implements the error interface
func (e *GhotiError) Error() string {
	return fmt.Sprintf("Ghoti error %s: %s", e.Code, e.Message)
}

// NewGhotiError creates a new GhotiError
func NewGhotiError(code string) *GhotiError {
	var message string
	switch code {
	case "001":
		message = "Invalid command"
	case "002":
		message = "Invalid slot number"
	case "003":
		message = "Invalid data length"
	case "004":
		message = "Authentication required"
	case "005":
		message = "Invalid credentials"
	case "006":
		message = "Permission denied"
	case "007":
		message = "Slot locked"
	case "008":
		message = "No tokens available"
	case "009":
		message = "Internal server error"
	default:
		message = "Unknown error"
	}
	return &GhotiError{
		Code:    code,
		Message: message,
	}
}