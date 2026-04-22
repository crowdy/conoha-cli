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

func TestResolveModeLogic(t *testing.T) {
	sentinelReadErr := errors.New("ssh broken")

	cases := []struct {
		name     string
		want     Mode  // flag value, "" if unset
		got      Mode  // ReadMarker return, used when readErr is nil or ErrNoMarker
		readErr  error // nil, ErrNoMarker, or some other error
		expMode  Mode  // expected return
		expErr   error // if non-nil, errors.Is must match (or "any non-nil" when expMode == "" and no sentinel)
		conflict bool  // expect ErrModeConflict specifically
	}{
		// Flag unset
		{"no flag + no marker", "", "", ErrNoMarker, "", ErrNoMarker, false},
		{"no flag + proxy marker", "", ModeProxy, nil, ModeProxy, nil, false},
		{"no flag + no-proxy marker", "", ModeNoProxy, nil, ModeNoProxy, nil, false},
		// Flag=proxy
		{"proxy flag + no marker", ModeProxy, "", ErrNoMarker, ModeProxy, nil, false},
		{"proxy flag + proxy marker", ModeProxy, ModeProxy, nil, ModeProxy, nil, false},
		{"proxy flag + no-proxy marker", ModeProxy, ModeNoProxy, nil, "", nil, true},
		// Flag=no-proxy
		{"no-proxy flag + no marker", ModeNoProxy, "", ErrNoMarker, ModeNoProxy, nil, false},
		{"no-proxy flag + no-proxy marker", ModeNoProxy, ModeNoProxy, nil, ModeNoProxy, nil, false},
		{"no-proxy flag + proxy marker", ModeNoProxy, ModeProxy, nil, "", nil, true},
		// SSH/read error — propagated regardless of flag
		{"ssh error + no flag", "", "", sentinelReadErr, "", sentinelReadErr, false},
		{"ssh error + proxy flag", ModeProxy, "", sentinelReadErr, "", sentinelReadErr, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mode, err := resolveModeLogic("myapp", "my-server", c.want, c.got, c.readErr)
			if mode != c.expMode {
				t.Errorf("mode = %q, want %q", mode, c.expMode)
			}
			switch {
			case c.conflict:
				if !errors.Is(err, ErrModeConflict) {
					t.Errorf("expected ErrModeConflict, got %v", err)
				}
			case c.expErr != nil:
				if !errors.Is(err, c.expErr) {
					t.Errorf("expected %v, got %v", c.expErr, err)
				}
			default:
				if err != nil {
					t.Errorf("expected nil err, got %v", err)
				}
			}
		})
	}
}

func TestFormatModeConflictError(t *testing.T) {
	err := formatModeConflictError("myapp", "my-server", ModeProxy, ModeNoProxy)
	if !errors.Is(err, ErrModeConflict) {
		t.Errorf("expected ErrModeConflict, got %v", err)
	}
	msg := err.Error()
	for _, want := range []string{
		`"myapp"`,
		"proxy mode",
		"--no-proxy was requested",
		"conoha app destroy my-server",
		"conoha app init --no-proxy my-server",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("conflict error missing %q: %s", want, msg)
		}
	}
}

func TestFormatModeConflictError_missingServerID(t *testing.T) {
	err := formatModeConflictError("myapp", "", ModeProxy, ModeNoProxy)
	if !strings.Contains(err.Error(), "conoha app destroy <server>") {
		t.Errorf("expected <server> placeholder when serverID is empty: %s", err.Error())
	}
}

func TestNotInitializedError(t *testing.T) {
	cases := []struct {
		name    string
		mode    Mode
		wantSub string
	}{
		{"proxy mode", ModeProxy, "conoha app init my-server"},
		{"no-proxy mode", ModeNoProxy, "conoha app init --no-proxy --app-name myapp my-server"},
		{"mode unknown", "", "conoha app init my-server"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := notInitializedError("myapp", "my-server", c.mode).Error()
			if !strings.Contains(got, c.wantSub) {
				t.Errorf("want substring %q, got %q", c.wantSub, got)
			}
			if !strings.Contains(got, `app "myapp"`) {
				t.Errorf("missing quoted app name: %s", got)
			}
		})
	}
}

func TestNotDeployedError(t *testing.T) {
	got := notDeployedError("myapp", "my-server").Error()
	for _, want := range []string{`"myapp"`, "not been deployed", "conoha app deploy my-server"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
