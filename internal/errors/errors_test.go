package errors

import (
	"fmt"
	"testing"
)

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 500, Code: "ServerError", Message: "internal"}
	if err.ExitCode() != ExitAPI {
		t.Errorf("expected %d, got %d", ExitAPI, err.ExitCode())
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}

func TestAuthError(t *testing.T) {
	err := &AuthError{Message: "unauthorized"}
	if err.ExitCode() != ExitAuth {
		t.Errorf("expected %d, got %d", ExitAuth, err.ExitCode())
	}
	if err.Error() != "auth error: unauthorized" {
		t.Errorf("unexpected message: %s", err.Error())
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{Resource: "server", ID: "abc"}
	if err.ExitCode() != ExitNotFound {
		t.Errorf("expected %d, got %d", ExitNotFound, err.ExitCode())
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "name", Message: "required"}
	if err.ExitCode() != ExitValidation {
		t.Errorf("expected %d, got %d", ExitValidation, err.ExitCode())
	}
}

func TestNetworkError(t *testing.T) {
	inner := fmt.Errorf("connection refused")
	err := &NetworkError{Err: inner}
	if err.ExitCode() != ExitNetwork {
		t.Errorf("expected %d, got %d", ExitNetwork, err.ExitCode())
	}
	if err.Unwrap() != inner {
		t.Error("Unwrap should return inner error")
	}
}

func TestConfigError(t *testing.T) {
	err := &ConfigError{Message: "missing profile"}
	if err.ExitCode() != ExitGeneral {
		t.Errorf("expected %d, got %d", ExitGeneral, err.ExitCode())
	}
}

func TestGetExitCode(t *testing.T) {
	if code := GetExitCode(&AuthError{Message: "x"}); code != ExitAuth {
		t.Errorf("expected %d, got %d", ExitAuth, code)
	}
	if code := GetExitCode(fmt.Errorf("generic")); code != ExitGeneral {
		t.Errorf("expected %d, got %d", ExitGeneral, code)
	}
}
