package app

import (
	"errors"
	"strings"
	"testing"
)

func TestMode_String(t *testing.T) {
	if string(ModeProxy) != "proxy" {
		t.Errorf("ModeProxy = %q, want %q", ModeProxy, "proxy")
	}
	if string(ModeNoProxy) != "no-proxy" {
		t.Errorf("ModeNoProxy = %q, want %q", ModeNoProxy, "no-proxy")
	}
}

func TestParseMarker(t *testing.T) {
	cases := []struct {
		in      string
		want    Mode
		wantErr bool
	}{
		{"proxy\n", ModeProxy, false},
		{"no-proxy\n", ModeNoProxy, false},
		{"proxy", ModeProxy, false},
		{"no-proxy", ModeNoProxy, false},
		{"  no-proxy  \n", ModeNoProxy, false},
		{"", "", true},
		{"garbage", "", true},
		{"Proxy", "", true},
	}
	for _, c := range cases {
		got, err := ParseMarker(c.in)
		if c.wantErr && err == nil {
			t.Errorf("ParseMarker(%q) expected error, got %q", c.in, got)
		}
		if !c.wantErr && err != nil {
			t.Errorf("ParseMarker(%q) err=%v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseMarker(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBuildReadMarkerCmd(t *testing.T) {
	got := buildReadMarkerCmd("myapp")
	for _, want := range []string{
		"/opt/conoha/myapp/.conoha-mode",
		"cat",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildWriteMarkerCmd(t *testing.T) {
	got := buildWriteMarkerCmd("myapp", ModeNoProxy)
	for _, want := range []string{
		"mkdir -p '/opt/conoha/myapp'",
		"/opt/conoha/myapp/.conoha-mode",
		"no-proxy",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildReadCurrentSlotCmd(t *testing.T) {
	got := buildReadCurrentSlotCmd("myapp")
	for _, want := range []string{
		"/opt/conoha/myapp/CURRENT_SLOT",
		"cat",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestFormatModeConflictError(t *testing.T) {
	err := formatModeConflictError("myapp", ModeProxy, ModeNoProxy)
	if !errors.Is(err, ErrModeConflict) {
		t.Errorf("expected ErrModeConflict, got %v", err)
	}
	msg := err.Error()
	for _, want := range []string{
		`"myapp"`,
		"proxy mode",
		"--no-proxy was requested",
		"conoha app destroy",
		"conoha app init --no-proxy",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("conflict error missing %q: %s", want, msg)
		}
	}
}
