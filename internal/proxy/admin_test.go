package proxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

// fakeExecutor records the most recent command and returns a canned response.
type fakeExecutor struct {
	LastCmd   string
	LastStdin []byte
	Status    int    // HTTP status the fake curl will emit
	Body      string // HTTP body
	RunErr    error  // process-level error
}

func (f *fakeExecutor) Run(cmd string, stdin io.Reader, stdout io.Writer) error {
	f.LastCmd = cmd
	if stdin != nil {
		b, _ := io.ReadAll(stdin)
		f.LastStdin = b
	}
	if f.RunErr != nil {
		return f.RunErr
	}
	// Our admin client expects body then a trailing line "HTTPSTATUS:NNN".
	_, _ = io.WriteString(stdout, f.Body)
	_, _ = io.WriteString(stdout, "\nHTTPSTATUS:")
	_, _ = io.WriteString(stdout, intToStr(f.Status))
	return nil
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	var out []byte
	for n > 0 {
		out = append([]byte{byte('0' + n%10)}, out...)
		n /= 10
	}
	return string(out)
}

func TestClient_Get(t *testing.T) {
	svc := Service{Name: "myapp", Hosts: []string{"a.example.com"}, Phase: PhaseLive, TLSStatus: "unknown"}
	b, _ := json.Marshal(svc)
	fx := &fakeExecutor{Status: 200, Body: string(b)}
	c := NewClient(fx, "/var/lib/conoha-proxy/admin.sock")

	got, err := c.Get("myapp")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "myapp" {
		t.Errorf("Name = %q", got.Name)
	}
	if !strings.Contains(fx.LastCmd, "http://admin/v1/services/myapp") {
		t.Errorf("LastCmd = %q", fx.LastCmd)
	}
	if !strings.Contains(fx.LastCmd, "--unix-socket /var/lib/conoha-proxy/admin.sock") {
		t.Errorf("missing --unix-socket in %q", fx.LastCmd)
	}
}

func TestClient_Get_NotFound(t *testing.T) {
	fx := &fakeExecutor{Status: 404, Body: `{"error":{"code":"not_found","message":"no such"}}`}
	c := NewClient(fx, "/sock")
	_, err := c.Get("nope")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestClient_Upsert(t *testing.T) {
	svc := Service{Name: "myapp", Hosts: []string{"a.example.com"}, Phase: PhaseConfigured}
	b, _ := json.Marshal(svc)
	fx := &fakeExecutor{Status: 201, Body: string(b)}
	c := NewClient(fx, "/sock")

	req := UpsertRequest{Name: "myapp", Hosts: []string{"a.example.com"}}
	got, err := c.Upsert(req)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "myapp" {
		t.Errorf("Name = %q", got.Name)
	}
	if !strings.Contains(fx.LastCmd, "-X POST") {
		t.Errorf("missing POST in %q", fx.LastCmd)
	}
	if !strings.Contains(fx.LastCmd, "http://admin/v1/services") {
		t.Errorf("wrong URL in %q", fx.LastCmd)
	}
	if !bytes.Contains(fx.LastStdin, []byte(`"name":"myapp"`)) {
		t.Errorf("stdin body = %q", fx.LastStdin)
	}
}

func TestClient_Deploy_ProbeFailed(t *testing.T) {
	fx := &fakeExecutor{Status: 424, Body: `{"error":{"code":"probe_failed","message":"bad /up"}}`}
	c := NewClient(fx, "/sock")
	_, err := c.Deploy("myapp", DeployRequest{TargetURL: "http://127.0.0.1:9001"})
	var pe *ProbeFailedError
	if !errors.As(err, &pe) {
		t.Fatalf("got %v, want ProbeFailedError", err)
	}
}

func TestClient_Rollback_NoDrainTarget(t *testing.T) {
	fx := &fakeExecutor{Status: 409, Body: `{"error":{"code":"no_drain_target","message":"closed"}}`}
	c := NewClient(fx, "/sock")
	_, err := c.Rollback("myapp", 30000)
	if !errors.Is(err, ErrNoDrainTarget) {
		t.Errorf("got %v, want ErrNoDrainTarget", err)
	}
}

func TestClient_Delete_204(t *testing.T) {
	fx := &fakeExecutor{Status: 204, Body: ``}
	c := NewClient(fx, "/sock")
	if err := c.Delete("myapp"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fx.LastCmd, "-X DELETE") {
		t.Errorf("missing DELETE in %q", fx.LastCmd)
	}
}

func TestClient_List(t *testing.T) {
	fx := &fakeExecutor{Status: 200, Body: `{"services":[{"name":"a"},{"name":"b"}]}`}
	c := NewClient(fx, "/sock")
	out, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].Name != "a" || out[1].Name != "b" {
		t.Errorf("got %+v", out)
	}
}
