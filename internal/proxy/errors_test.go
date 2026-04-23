package proxy

import (
	"errors"
	"strings"
	"testing"
)

func TestParseAPIError(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   error
	}{
		{200, `{}`, nil},
		{201, `{}`, nil},
		{204, ``, nil},
		{299, `{}`, nil}, // upper 2xx boundary — must stay success
		{404, `{"error":{"code":"not_found","message":"nope"}}`, ErrNotFound},
		{409, `{"error":{"code":"no_drain_target","message":"closed"}}`, ErrNoDrainTarget},
	}
	for _, tc := range cases {
		err := ParseAPIError(tc.status, []byte(tc.body))
		if tc.want == nil {
			if err != nil {
				t.Errorf("status %d: got %v, want nil", tc.status, err)
			}
			continue
		}
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: got %v, want errors.Is == %v", tc.status, err, tc.want)
		}
	}
}

func TestParseAPIError_MethodNotAllowed(t *testing.T) {
	// 405 currently has no typed handler — hits the unhandled-status
	// fallthrough. Lock in that shape (should surface enough detail for ops).
	err := ParseAPIError(405, []byte(`{"error":{"code":"method_not_allowed","message":"use POST"}}`))
	if err == nil {
		t.Fatal("want non-nil error for 405")
	}
	msg := err.Error()
	for _, want := range []string{"405", "method_not_allowed", "use POST"} {
		if !strings.Contains(msg, want) {
			t.Errorf("405 error missing %q in: %s", want, msg)
		}
	}
}

func TestParseAPIError_Unhandled3xx(t *testing.T) {
	// 3xx is not success (per HTTP) and not a typed proxy error. Falls
	// through to the generic "unexpected HTTP %d" branch; regression-guard.
	err := ParseAPIError(302, []byte(`{}`))
	if err == nil {
		t.Fatal("want non-nil error for 302")
	}
	if !strings.Contains(err.Error(), "302") {
		t.Errorf("302 error should include status; got: %v", err)
	}
}

func TestParseAPIError_MalformedBody(t *testing.T) {
	// Garbage body for an error status — should not panic; fall through to
	// a message-free typed error.
	err := ParseAPIError(500, []byte("not-json-at-all"))
	if err == nil {
		t.Fatal("want non-nil error")
	}
	var se *ServerError
	if !errors.As(err, &se) {
		t.Fatalf("want ServerError, got %T: %v", err, err)
	}
	if se.Status != 500 {
		t.Errorf("Status = %d, want 500", se.Status)
	}
}


func TestParseAPIError_ProbeFailed(t *testing.T) {
	err := ParseAPIError(424, []byte(`{"error":{"code":"probe_failed","message":"upstream /up returned 500"}}`))
	var pe *ProbeFailedError
	if !errors.As(err, &pe) {
		t.Fatalf("want ProbeFailedError, got %v", err)
	}
	if pe.Message != "upstream /up returned 500" {
		t.Errorf("Message = %q", pe.Message)
	}
}

func TestParseAPIError_Validation(t *testing.T) {
	err := ParseAPIError(400, []byte(`{"error":{"code":"validation_failed","message":"name empty"}}`))
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("want ValidationError, got %v", err)
	}
}

func TestParseAPIError_ServerError(t *testing.T) {
	err := ParseAPIError(503, []byte(`{"error":{"code":"store_error","message":"disk full"}}`))
	if err == nil {
		t.Fatal("want error")
	}
	var se *ServerError
	if !errors.As(err, &se) {
		t.Fatalf("want ServerError, got %v", err)
	}
	if se.Code != "store_error" {
		t.Errorf("Code = %q", se.Code)
	}
}
