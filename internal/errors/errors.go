package errors

import (
	stderrors "errors"
	"fmt"
)

// ExitCoder is implemented by errors that carry a process exit code.
type ExitCoder interface {
	ExitCode() int
}

// exitCodeErr annotates an underlying error with a specific process exit code
// while preserving its message and wrap chain. Used by packages (cmd/app, ...)
// that want typed sub-failures without defining a new struct per case.
type exitCodeErr struct {
	err  error
	code int
}

func (e *exitCodeErr) Error() string { return e.err.Error() }
func (e *exitCodeErr) Unwrap() error { return e.err }
func (e *exitCodeErr) ExitCode() int { return e.code }

// WithExitCode wraps err so that GetExitCode returns code. The wrapped error
// remains errors.Is/As-compatible with whatever err already wraps.
func WithExitCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return &exitCodeErr{err: err, code: code}
}

// APIError represents an error returned by the ConoHa API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("API error (HTTP %d, %s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("API error (HTTP %d): %s", e.StatusCode, e.Message)
}

func (e *APIError) ExitCode() int {
	return ExitAPI
}

// AuthError represents an authentication or authorization failure.
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("auth error: %s", e.Message)
}

func (e *AuthError) ExitCode() int {
	return ExitAuth
}

// ConfigError represents a configuration problem.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error: %s", e.Message)
}

func (e *ConfigError) ExitCode() int {
	return ExitGeneral
}

// NotFoundError indicates that a requested resource was not found.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

func (e *NotFoundError) ExitCode() int {
	return ExitNotFound
}

// ValidationError represents invalid user input.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on %s: %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func (e *ValidationError) ExitCode() int {
	return ExitValidation
}

// NetworkError wraps an underlying network-level error.
type NetworkError struct {
	Err error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error: %v", e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}

func (e *NetworkError) ExitCode() int {
	return ExitNetwork
}

// GetExitCode returns the exit code for the given error. Traverses the
// error wrap chain (errors.As) to find the first ExitCoder; returns
// ExitGeneral when no ExitCoder is reachable.
func GetExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var ec ExitCoder
	if stderrors.As(err, &ec) {
		return ec.ExitCode()
	}
	return ExitGeneral
}
