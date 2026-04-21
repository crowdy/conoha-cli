package proxy

import (
	"errors"
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
