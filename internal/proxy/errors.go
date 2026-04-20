package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Sentinel errors.
var (
	ErrNotFound      = errors.New("proxy: service not found")
	ErrNoDrainTarget = errors.New("proxy: drain window has closed")
)

// ValidationError is a proxy 400 response.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("proxy validation error (%s): %s", e.Code, e.Message)
}

// ProbeFailedError is a proxy 424 response. State on the server was NOT mutated.
type ProbeFailedError struct {
	Message string
}

func (e *ProbeFailedError) Error() string {
	return fmt.Sprintf("proxy probe failed: %s", e.Message)
}

// ServerError is a proxy 5xx response.
type ServerError struct {
	Status  int
	Code    string
	Message string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("proxy server error (HTTP %d, %s): %s", e.Status, e.Code, e.Message)
}

type apiErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// ParseAPIError returns nil for 2xx and an appropriate typed error otherwise.
func ParseAPIError(status int, body []byte) error {
	if status >= 200 && status < 300 {
		return nil
	}
	var b apiErrorBody
	_ = json.Unmarshal(body, &b)
	msg := b.Error.Message
	code := b.Error.Code
	switch status {
	case 404:
		return fmt.Errorf("%w: %s", ErrNotFound, msg)
	case 409:
		return fmt.Errorf("%w: %s", ErrNoDrainTarget, msg)
	case 400:
		return &ValidationError{Code: code, Message: msg}
	case 424:
		return &ProbeFailedError{Message: msg}
	}
	if status >= 500 {
		return &ServerError{Status: status, Code: code, Message: msg}
	}
	return fmt.Errorf("proxy: unexpected HTTP %d (%s): %s", status, code, msg)
}
