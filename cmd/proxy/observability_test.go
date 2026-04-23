package proxy

import (
	"errors"
	"io"
	"strings"
	"testing"

	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

func TestJSONField_HappyPath(t *testing.T) {
	body := []byte(`{"version":"1.2.3","build":"abc"}`)
	if got := jsonField(body, "version", nil); got != "1.2.3" {
		t.Errorf("version: got %q, want 1.2.3", got)
	}
	if got := jsonField(body, "build", nil); got != "abc" {
		t.Errorf("build: got %q, want abc", got)
	}
}

func TestJSONField_UpstreamError(t *testing.T) {
	upErr := errors.New("curl failed")
	got := jsonField(nil, "version", upErr)
	if !strings.Contains(got, "error:") {
		t.Errorf("upstream err should be surfaced as \"(error: ...)\": got %q", got)
	}
	if !strings.Contains(got, "curl failed") {
		t.Errorf("original error text missing: got %q", got)
	}
}

func TestJSONField_DecodeError(t *testing.T) {
	got := jsonField([]byte("not json"), "version", nil)
	if !strings.Contains(got, "decode error") {
		t.Errorf("bad JSON should produce decode error: got %q", got)
	}
}

func TestJSONField_MissingKey(t *testing.T) {
	got := jsonField([]byte(`{"version":"1"}`), "build", nil)
	if got != "(missing)" {
		t.Errorf("missing key: got %q, want (missing)", got)
	}
}

func TestJSONField_NonStringValue(t *testing.T) {
	// Numbers stringify via fmt.Sprint — callers that want exact format
	// should post-process. Lock in the current behavior so refactors that
	// switch to strconv don't silently change output.
	got := jsonField([]byte(`{"ready":true}`), "ready", nil)
	if got != "true" {
		t.Errorf("bool value: got %q, want true", got)
	}
	got = jsonField([]byte(`{"count":42}`), "count", nil)
	if got != "42" {
		t.Errorf("number value: got %q, want 42", got)
	}
}

// fakeExec implements proxypkg.Executor by recording the command and writing
// a canned response to stdout.
type fakeExec struct {
	lastCmd  string
	response string
	runErr   error
}

func (f *fakeExec) Run(cmd string, stdin io.Reader, stdout io.Writer) error {
	f.lastCmd = cmd
	if f.runErr != nil {
		return f.runErr
	}
	_, _ = io.WriteString(stdout, f.response)
	return nil
}

var _ proxypkg.Executor = (*fakeExec)(nil)

func TestCurlVia_CommandShape(t *testing.T) {
	fx := &fakeExec{response: `{"ok":true}`}
	body, err := curlVia(fx, "/var/lib/conoha-proxy", "/version")
	if err != nil {
		t.Fatalf("curlVia: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q", string(body))
	}
	// #100: both the socket and URL must be single-quoted so paths with
	// whitespace survive the shell.
	for _, want := range []string{
		"curl -sS",
		"--unix-socket '/var/lib/conoha-proxy/admin.sock'",
		"'http://admin/version'",
	} {
		if !strings.Contains(fx.lastCmd, want) {
			t.Errorf("missing %q in %q", want, fx.lastCmd)
		}
	}
}

func TestCurlVia_PropagatesRunError(t *testing.T) {
	fx := &fakeExec{runErr: errors.New("ssh broken")}
	if _, err := curlVia(fx, "/data", "/version"); err == nil {
		t.Fatal("expected Run error to propagate")
	}
}
