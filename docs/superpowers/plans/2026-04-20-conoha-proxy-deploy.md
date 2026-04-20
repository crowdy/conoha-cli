# conoha-proxy blue/green Deploy Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `conoha app deploy` with a conoha-proxy-backed blue/green flow, add a new `conoha proxy` command group, and introduce a `conoha.yml` project file.

**Architecture:** A local CLI drives a remote proxy over SSH. `internal/proxy/admin.go` speaks the proxy Admin API by invoking `curl --unix-socket` on the VPS via SSH. `internal/proxy/bootstrap.go` emits docker shell scripts for proxy lifecycle. `cmd/proxy/*` wraps both. `cmd/app/deploy.go` orchestrates: tar upload → slot-scoped compose up with dynamic port → proxy `/deploy` → old slot teardown after drain. `conoha.yml` declares name, hosts, web service/port, accessories.

**Tech Stack:** Go 1.26, spf13/cobra, gopkg.in/yaml.v3, golang.org/x/crypto/ssh, docker compose on the VPS, conoha-proxy v0.1+.

**Spec:** `docs/superpowers/specs/2026-04-20-conoha-proxy-deploy-design.md`

---

## Phase 1: Project file (`conoha.yml`)

### Task 1: ProjectFile types and loader

**Files:**
- Create: `internal/config/projectfile.go`
- Create: `internal/config/projectfile_test.go`

- [ ] **Step 1: Write failing tests for Load()**

Create `internal/config/projectfile_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectFile_Minimal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conoha.yml")
	data := []byte("name: myapp\nhosts: [app.example.com]\nweb:\n  service: web\n  port: 8080\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	pf, err := LoadProjectFile(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if pf.Name != "myapp" {
		t.Errorf("Name = %q, want %q", pf.Name, "myapp")
	}
	if len(pf.Hosts) != 1 || pf.Hosts[0] != "app.example.com" {
		t.Errorf("Hosts = %v", pf.Hosts)
	}
	if pf.Web.Service != "web" || pf.Web.Port != 8080 {
		t.Errorf("Web = %+v", pf.Web)
	}
}

func TestLoadProjectFile_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conoha.yml")
	data := []byte(`name: myapp
hosts: [a.example.com, b.example.com]
web:
  service: web
  port: 3000
compose_file: docker-compose.yml
accessories: [db, redis]
health:
  path: /healthz
  interval_ms: 1000
  timeout_ms: 500
  healthy_threshold: 2
  unhealthy_threshold: 5
deploy:
  drain_ms: 15000
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	pf, err := LoadProjectFile(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if pf.ComposeFile != "docker-compose.yml" {
		t.Errorf("ComposeFile = %q", pf.ComposeFile)
	}
	if len(pf.Accessories) != 2 {
		t.Errorf("Accessories = %v", pf.Accessories)
	}
	if pf.Health == nil || pf.Health.Path != "/healthz" || pf.Health.IntervalMs != 1000 {
		t.Errorf("Health = %+v", pf.Health)
	}
	if pf.Deploy == nil || pf.Deploy.DrainMs != 15000 {
		t.Errorf("Deploy = %+v", pf.Deploy)
	}
}

func TestLoadProjectFile_Missing(t *testing.T) {
	_, err := LoadProjectFile(filepath.Join(t.TempDir(), "no-such.yml"))
	if err == nil {
		t.Fatal("want error for missing file")
	}
}

func TestLoadProjectFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conoha.yml")
	if err := os.WriteFile(path, []byte("name: [unterminated"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadProjectFile(path)
	if err == nil {
		t.Fatal("want error for invalid YAML")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/ -run TestLoadProjectFile -v`
Expected: FAIL (undefined: LoadProjectFile)

- [ ] **Step 3: Implement LoadProjectFile**

Create `internal/config/projectfile.go`:

```go
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProjectFileName is the canonical project file name.
const ProjectFileName = "conoha.yml"

// ProjectFile is the parsed conoha.yml declaration that lives at the repo root.
type ProjectFile struct {
	Name        string       `yaml:"name"`
	Hosts       []string     `yaml:"hosts"`
	Web         WebSpec      `yaml:"web"`
	ComposeFile string       `yaml:"compose_file,omitempty"`
	Accessories []string     `yaml:"accessories,omitempty"`
	Health      *HealthSpec  `yaml:"health,omitempty"`
	Deploy      *DeploySpec  `yaml:"deploy,omitempty"`
}

// WebSpec declares which compose service is the blue/green target.
type WebSpec struct {
	Service string `yaml:"service"`
	Port    int    `yaml:"port"`
}

// HealthSpec mirrors proxy's health_policy object (all fields optional).
type HealthSpec struct {
	Path               string `yaml:"path,omitempty"`
	IntervalMs         int    `yaml:"interval_ms,omitempty"`
	TimeoutMs          int    `yaml:"timeout_ms,omitempty"`
	HealthyThreshold   int    `yaml:"healthy_threshold,omitempty"`
	UnhealthyThreshold int    `yaml:"unhealthy_threshold,omitempty"`
}

// DeploySpec carries deploy-time parameters (currently only drain_ms).
type DeploySpec struct {
	DrainMs int `yaml:"drain_ms,omitempty"`
}

// LoadProjectFile reads and YAML-decodes a project file. It does NOT validate.
func LoadProjectFile(path string) (*ProjectFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var pf ProjectFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &pf, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/ -run TestLoadProjectFile -v`
Expected: PASS (4 tests)

- [ ] **Step 5: Commit**

```bash
git add internal/config/projectfile.go internal/config/projectfile_test.go
git commit -m "feat(config): add conoha.yml project file loader"
```

---

### Task 2: ProjectFile validation

**Files:**
- Modify: `internal/config/projectfile.go`
- Modify: `internal/config/projectfile_test.go`

- [ ] **Step 1: Add failing validation tests**

Append to `internal/config/projectfile_test.go`:

```go
func TestProjectFile_Validate(t *testing.T) {
	good := ProjectFile{
		Name:  "myapp",
		Hosts: []string{"app.example.com"},
		Web:   WebSpec{Service: "web", Port: 8080},
	}
	if err := good.Validate(); err != nil {
		t.Errorf("good Validate: %v", err)
	}

	cases := []struct {
		name string
		mod  func(*ProjectFile)
		want string // substring of error
	}{
		{"empty name", func(p *ProjectFile) { p.Name = "" }, "name"},
		{"bad name", func(p *ProjectFile) { p.Name = "My-App_1" }, "name"},
		{"too long name", func(p *ProjectFile) { p.Name = "a" + string(make([]byte, 63)) }, "name"},
		{"no hosts", func(p *ProjectFile) { p.Hosts = nil }, "hosts"},
		{"empty host", func(p *ProjectFile) { p.Hosts = []string{""} }, "hosts"},
		{"dup hosts", func(p *ProjectFile) { p.Hosts = []string{"a.com", "a.com"} }, "duplicate"},
		{"no web service", func(p *ProjectFile) { p.Web.Service = "" }, "web.service"},
		{"no port", func(p *ProjectFile) { p.Web.Port = 0 }, "web.port"},
		{"bad port high", func(p *ProjectFile) { p.Web.Port = 70000 }, "web.port"},
		{"accessory = web", func(p *ProjectFile) { p.Accessories = []string{"web"} }, "accessor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := good
			tc.mod(&p)
			err := p.Validate()
			if err == nil {
				t.Fatalf("want error containing %q, got nil", tc.want)
			}
			if !contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/config/ -run TestProjectFile_Validate -v`
Expected: FAIL (undefined method Validate)

- [ ] **Step 3: Implement Validate()**

Append to `internal/config/projectfile.go`:

```go
import "regexp"

var dnsLabelRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// Validate enforces the schema rules documented in the spec.
func (p *ProjectFile) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(p.Name) > 63 || !dnsLabelRe.MatchString(p.Name) {
		return fmt.Errorf("name %q is not a valid DNS-1123 label (lowercase alphanumerics and hyphens, 1-63 chars)", p.Name)
	}
	if len(p.Hosts) == 0 {
		return fmt.Errorf("hosts must list at least one FQDN")
	}
	seen := make(map[string]struct{}, len(p.Hosts))
	for _, h := range p.Hosts {
		if h == "" {
			return fmt.Errorf("hosts contains empty entry")
		}
		if _, dup := seen[h]; dup {
			return fmt.Errorf("hosts contains duplicate %q", h)
		}
		seen[h] = struct{}{}
	}
	if p.Web.Service == "" {
		return fmt.Errorf("web.service is required")
	}
	if p.Web.Port < 1 || p.Web.Port > 65535 {
		return fmt.Errorf("web.port must be between 1 and 65535, got %d", p.Web.Port)
	}
	for _, a := range p.Accessories {
		if a == p.Web.Service {
			return fmt.Errorf("accessory %q conflicts with web.service", a)
		}
	}
	return nil
}
```

Note: place the `import "regexp"` statement inside the existing import block at the top of the file, do not add a second import statement.

- [ ] **Step 4: Add compose-file resolver and test**

Append to `internal/config/projectfile.go`:

```go
// ComposeFileCandidates is the auto-detect order when compose_file is unset.
// Mirrors cmd/app/deploy.go composeFileNames as of the conoha-proxy refactor.
var ComposeFileCandidates = []string{
	"conoha-docker-compose.yml",
	"conoha-docker-compose.yaml",
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// ResolveComposeFile returns the compose file path relative to dir.
// If the project file specified compose_file explicitly it is returned (existence is verified).
// Otherwise the first existing candidate from ComposeFileCandidates is returned.
func (p *ProjectFile) ResolveComposeFile(dir string) (string, error) {
	if p.ComposeFile != "" {
		full := p.ComposeFile
		if _, err := os.Stat(filepathJoin(dir, full)); err != nil {
			return "", fmt.Errorf("compose_file %q not found", full)
		}
		return full, nil
	}
	for _, name := range ComposeFileCandidates {
		if _, err := os.Stat(filepathJoin(dir, name)); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("no compose file found (tried %v)", ComposeFileCandidates)
}

func filepathJoin(dir, name string) string {
	if dir == "" || dir == "." {
		return name
	}
	return dir + string(os.PathSeparator) + name
}
```

Append tests:

```go
func TestResolveComposeFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &ProjectFile{}
	got, err := p.ResolveComposeFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "compose.yml" {
		t.Errorf("got %q, want compose.yml", got)
	}

	p.ComposeFile = "custom.yml"
	if _, err := p.ResolveComposeFile(dir); err == nil {
		t.Error("want error for missing compose_file")
	}
	if err := os.WriteFile(filepath.Join(dir, "custom.yml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = p.ResolveComposeFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "custom.yml" {
		t.Errorf("got %q", got)
	}
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/config/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/config/projectfile.go internal/config/projectfile_test.go
git commit -m "feat(config): validate conoha.yml and resolve compose file"
```

---

## Phase 2: Proxy Admin API client

### Task 3: Proxy domain types

**Files:**
- Create: `internal/proxy/types.go`

- [ ] **Step 1: Create types file**

Create `internal/proxy/types.go`:

```go
// Package proxy provides a client for the conoha-proxy Admin API and
// shell-script generators for managing its Docker container lifecycle.
package proxy

import "time"

// Phase mirrors conoha-proxy's externally observable phases.
type Phase string

const (
	PhaseConfigured Phase = "configured"
	PhaseLive       Phase = "live"
	PhaseSwapping   Phase = "swapping"
)

// Target is an upstream instance deployed to a service slot.
type Target struct {
	URL        string    `json:"url"`
	DeployedAt time.Time `json:"deployed_at"`
}

// HealthPolicy mirrors the proxy health_policy object.
type HealthPolicy struct {
	Path               string `json:"path,omitempty"`
	IntervalMs         int    `json:"interval_ms,omitempty"`
	TimeoutMs          int    `json:"timeout_ms,omitempty"`
	HealthyThreshold   int    `json:"healthy_threshold,omitempty"`
	UnhealthyThreshold int    `json:"unhealthy_threshold,omitempty"`
}

// Service is the proxy response body for the /v1/services endpoints.
type Service struct {
	Name           string        `json:"name"`
	Hosts          []string      `json:"hosts"`
	ActiveTarget   *Target       `json:"active_target,omitempty"`
	DrainingTarget *Target       `json:"draining_target,omitempty"`
	DrainDeadline  *time.Time    `json:"drain_deadline,omitempty"`
	HealthPolicy   *HealthPolicy `json:"health_policy,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	Phase          Phase         `json:"phase"`
	TLSStatus      string        `json:"tls_status"`
	TLSError       string        `json:"tls_error,omitempty"`
	LastDeployAt   *time.Time    `json:"last_deploy_at,omitempty"`
}

// UpsertRequest is the body of POST /v1/services.
type UpsertRequest struct {
	Name         string        `json:"name"`
	Hosts        []string      `json:"hosts"`
	HealthPolicy *HealthPolicy `json:"health_policy,omitempty"`
}

// DeployRequest is the body of POST /v1/services/{name}/deploy.
type DeployRequest struct {
	TargetURL string `json:"target_url"`
	DrainMs   int    `json:"drain_ms,omitempty"`
}

// RollbackRequest is the body of POST /v1/services/{name}/rollback.
type RollbackRequest struct {
	DrainMs int `json:"drain_ms,omitempty"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/proxy/`
Expected: success, no output

- [ ] **Step 3: Commit**

```bash
git add internal/proxy/types.go
git commit -m "feat(proxy): add Admin API domain types"
```

---

### Task 4: Proxy Admin client errors

**Files:**
- Create: `internal/proxy/errors.go`
- Create: `internal/proxy/errors_test.go`

- [ ] **Step 1: Write failing error-mapping tests**

Create `internal/proxy/errors_test.go`:

```go
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
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/proxy/ -v`
Expected: FAIL (undefined symbols)

- [ ] **Step 3: Implement errors.go**

Create `internal/proxy/errors.go`:

```go
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
```

- [ ] **Step 4: Run tests to verify**

Run: `go test ./internal/proxy/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/proxy/errors.go internal/proxy/errors_test.go
git commit -m "feat(proxy): add typed errors and API error parser"
```

---

### Task 5: Admin API client with SSH executor

**Files:**
- Create: `internal/proxy/admin.go`
- Create: `internal/proxy/admin_test.go`

- [ ] **Step 1: Write failing tests**

Create `internal/proxy/admin_test.go`:

```go
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
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./internal/proxy/ -run TestClient -v`
Expected: FAIL (undefined NewClient, Client.*)

- [ ] **Step 3: Implement admin.go**

Create `internal/proxy/admin.go`:

```go
package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Executor runs a shell command on a remote host.
// stdin is streamed to the command; stdout receives its output.
// Implementations should surface process-level failures as errors
// (exit code != 0 on the remote is NOT a process failure — curl returns 0
// even on HTTP errors because we pass -f off and parse the status ourselves).
type Executor interface {
	Run(cmd string, stdin io.Reader, stdout io.Writer) error
}

// Client speaks the conoha-proxy Admin API via an Executor.
type Client struct {
	exec Executor
	sock string
}

// NewClient constructs a Client with the given executor and socket path.
func NewClient(exec Executor, sock string) *Client {
	return &Client{exec: exec, sock: sock}
}

// Get returns a single service by name.
func (c *Client) Get(name string) (*Service, error) {
	body, err := c.call("GET", "/v1/services/"+name, nil)
	if err != nil {
		return nil, err
	}
	var s Service
	if err := json.Unmarshal(body, &s); err != nil {
		return nil, fmt.Errorf("decode service: %w", err)
	}
	return &s, nil
}

// List returns all registered services.
func (c *Client) List() ([]Service, error) {
	body, err := c.call("GET", "/v1/services", nil)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Services []Service `json:"services"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return nil, fmt.Errorf("decode list: %w", err)
	}
	return wrap.Services, nil
}

// Upsert creates or replaces a service.
func (c *Client) Upsert(req UpsertRequest) (*Service, error) {
	return c.postService("/v1/services", req)
}

// Deploy probes the new target and swaps on success.
func (c *Client) Deploy(name string, req DeployRequest) (*Service, error) {
	return c.postService("/v1/services/"+name+"/deploy", req)
}

// Rollback swaps active and draining targets within the drain window.
func (c *Client) Rollback(name string, drainMs int) (*Service, error) {
	return c.postService("/v1/services/"+name+"/rollback", RollbackRequest{DrainMs: drainMs})
}

// Delete removes the service; the drain window (if any) is discarded.
func (c *Client) Delete(name string) error {
	_, err := c.call("DELETE", "/v1/services/"+name, nil)
	return err
}

func (c *Client) postService(path string, body interface{}) (*Service, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	out, err := c.call("POST", path, data)
	if err != nil {
		return nil, err
	}
	var s Service
	if err := json.Unmarshal(out, &s); err != nil {
		return nil, fmt.Errorf("decode service: %w", err)
	}
	return &s, nil
}

// call synthesizes a curl invocation, runs it, splits status from body,
// and converts 4xx/5xx into typed errors via ParseAPIError.
func (c *Client) call(method, path string, body []byte) ([]byte, error) {
	parts := []string{
		"curl", "-sS",
		"--unix-socket", c.sock,
		"-X", method,
		"-w", `"\nHTTPSTATUS:%{http_code}"`,
	}
	if body != nil {
		parts = append(parts, "-H", "'Content-Type: application/json'", "--data-binary", "@-")
	}
	parts = append(parts, "'http://admin"+path+"'")
	cmd := strings.Join(parts, " ")

	var buf bytes.Buffer
	var stdin io.Reader
	if body != nil {
		stdin = bytes.NewReader(body)
	}
	if err := c.exec.Run(cmd, stdin, &buf); err != nil {
		return nil, fmt.Errorf("exec curl: %w", err)
	}
	respBody, status, err := splitStatus(buf.Bytes())
	if err != nil {
		return nil, err
	}
	if apiErr := ParseAPIError(status, respBody); apiErr != nil {
		return nil, apiErr
	}
	return respBody, nil
}

// splitStatus separates the trailing "\nHTTPSTATUS:NNN" line from the body.
// Accepts the body as-is if the tag is missing (maps to status 0 → error).
func splitStatus(raw []byte) (body []byte, status int, err error) {
	tag := []byte("\nHTTPSTATUS:")
	i := bytes.LastIndex(raw, tag)
	if i < 0 {
		return nil, 0, fmt.Errorf("missing HTTPSTATUS tag in curl output: %q", string(raw))
	}
	body = raw[:i]
	statusStr := strings.TrimSpace(string(raw[i+len(tag):]))
	n, convErr := strconv.Atoi(statusStr)
	if convErr != nil {
		return nil, 0, fmt.Errorf("parse HTTPSTATUS %q: %w", statusStr, convErr)
	}
	return body, n, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/proxy/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/proxy/admin.go internal/proxy/admin_test.go
git commit -m "feat(proxy): add Admin API client over generic Executor"
```

---

### Task 6: SSH-backed Executor adapter

**Files:**
- Create: `internal/proxy/sshexec.go`

- [ ] **Step 1: Implement adapter**

Create `internal/proxy/sshexec.go`:

```go
package proxy

import (
	"io"

	"golang.org/x/crypto/ssh"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// SSHExecutor adapts *ssh.Client to the proxy.Executor interface by piping
// curl commands over an SSH session. stdin (when non-nil) is streamed as the
// HTTP request body (--data-binary @-).
type SSHExecutor struct {
	Client *ssh.Client
	Stderr io.Writer // where SSH stderr gets routed (can be nil → discarded)
}

// Run implements proxy.Executor.
func (e *SSHExecutor) Run(cmd string, stdin io.Reader, stdout io.Writer) error {
	stderr := e.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	if stdin != nil {
		_, err := internalssh.RunWithStdin(e.Client, cmd, stdin, stdout, stderr)
		return err
	}
	_, err := internalssh.RunCommand(e.Client, cmd, stdout, stderr)
	return err
}
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./internal/proxy/`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/proxy/sshexec.go
git commit -m "feat(proxy): add SSH-backed executor adapter"
```

---

## Phase 3: Proxy container bootstrap scripts

### Task 7: Bootstrap script generators

**Files:**
- Create: `internal/proxy/bootstrap.go`
- Create: `internal/proxy/bootstrap_test.go`

- [ ] **Step 1: Write tests**

Create `internal/proxy/bootstrap_test.go`:

```go
package proxy

import (
	"strings"
	"testing"
)

func TestBootScript_ContainsEssentials(t *testing.T) {
	s := string(BootScript(BootParams{
		Email:     "ops@example.com",
		Image:     "ghcr.io/crowdy/conoha-proxy:latest",
		DataDir:   "/var/lib/conoha-proxy",
		Container: "conoha-proxy",
	}))
	for _, want := range []string{
		"set -euo pipefail",
		"mkdir -p /var/lib/conoha-proxy",
		"chown 65532:65532 /var/lib/conoha-proxy",
		"-p 80:80",
		"-p 443:443",
		"-v /var/lib/conoha-proxy:/var/lib/conoha-proxy",
		"ghcr.io/crowdy/conoha-proxy:latest",
		"--acme-email=ops@example.com",
		"--name conoha-proxy",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("BootScript missing %q:\n%s", want, s)
		}
	}
}

func TestRebootScript_PullsStopsRemovesStarts(t *testing.T) {
	s := string(RebootScript(BootParams{
		Email:     "ops@example.com",
		Image:     "ghcr.io/crowdy/conoha-proxy:latest",
		DataDir:   "/var/lib/conoha-proxy",
		Container: "conoha-proxy",
	}))
	for _, want := range []string{
		"docker pull ghcr.io/crowdy/conoha-proxy:latest",
		"docker stop conoha-proxy",
		"docker rm conoha-proxy",
		"--acme-email=ops@example.com",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("RebootScript missing %q:\n%s", want, s)
		}
	}
}

func TestSimpleScripts(t *testing.T) {
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"start", string(StartScript("conoha-proxy")), "docker start conoha-proxy"},
		{"stop", string(StopScript("conoha-proxy")), "docker stop conoha-proxy"},
		{"restart", string(RestartScript("conoha-proxy")), "docker restart conoha-proxy"},
	}
	for _, tc := range cases {
		if !strings.Contains(tc.got, tc.want) {
			t.Errorf("%s script missing %q:\n%s", tc.name, tc.want, tc.got)
		}
	}
}

func TestRemoveScript_Purge(t *testing.T) {
	s := string(RemoveScript("conoha-proxy", "/var/lib/conoha-proxy", true))
	if !strings.Contains(s, "docker rm -f conoha-proxy") {
		t.Errorf("missing rm: %s", s)
	}
	if !strings.Contains(s, "rm -rf /var/lib/conoha-proxy") {
		t.Errorf("missing purge: %s", s)
	}

	s = string(RemoveScript("conoha-proxy", "/var/lib/conoha-proxy", false))
	if strings.Contains(s, "rm -rf /var/lib/conoha-proxy") {
		t.Errorf("non-purge should NOT delete data dir: %s", s)
	}
}

func TestLogsScript_FollowAndLines(t *testing.T) {
	s := string(LogsScript("conoha-proxy", true, 50))
	if !strings.Contains(s, "-f") || !strings.Contains(s, "--tail 50") {
		t.Errorf("flags missing: %s", s)
	}
	s = string(LogsScript("conoha-proxy", false, 0))
	if strings.Contains(s, "-f") || strings.Contains(s, "--tail") {
		t.Errorf("no flags expected: %s", s)
	}
}
```

- [ ] **Step 2: Verify failure**

Run: `go test ./internal/proxy/ -run 'TestBootScript|TestRebootScript|TestSimpleScripts|TestRemoveScript|TestLogsScript' -v`
Expected: FAIL (undefined)

- [ ] **Step 3: Implement bootstrap.go**

Create `internal/proxy/bootstrap.go`:

```go
package proxy

import "fmt"

// BootParams bundles the config needed by BootScript / RebootScript.
type BootParams struct {
	Email     string // --acme-email value (required)
	Image     string // docker image reference
	DataDir   string // host path mounted at /var/lib/conoha-proxy
	Container string // container name (e.g. "conoha-proxy")
}

// BootScript installs docker if missing, creates the data volume with the
// correct ownership, and runs the conoha-proxy container.
func BootScript(p BootParams) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
    echo "==> Installing Docker..."
    curl -fsSL https://get.docker.com | sh
fi

echo "==> Preparing data directory %[3]s"
mkdir -p %[3]s
chown 65532:65532 %[3]s

if docker inspect %[4]s >/dev/null 2>&1; then
    echo "Container %[4]s already exists. Use 'conoha proxy reboot' to upgrade."
    exit 0
fi

echo "==> Starting %[4]s from %[2]s"
docker run -d --name %[4]s \
  --restart unless-stopped \
  -p 80:80 -p 443:443 \
  -v %[3]s:%[3]s \
  %[2]s \
  run --acme-email=%[1]s

echo "==> Done. Admin socket: %[3]s/admin.sock"
`, p.Email, p.Image, p.DataDir, p.Container))
}

// RebootScript pulls the image then replaces the existing container, keeping the volume.
func RebootScript(p BootParams) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

echo "==> Pulling %[2]s"
docker pull %[2]s

if docker inspect %[4]s >/dev/null 2>&1; then
    echo "==> Stopping %[4]s"
    docker stop %[4]s >/dev/null
    docker rm %[4]s >/dev/null
fi

echo "==> Starting new %[4]s from %[2]s"
docker run -d --name %[4]s \
  --restart unless-stopped \
  -p 80:80 -p 443:443 \
  -v %[3]s:%[3]s \
  %[2]s \
  run --acme-email=%[1]s
`, p.Email, p.Image, p.DataDir, p.Container))
}

// StartScript / StopScript / RestartScript are trivial wrappers.
func StartScript(container string) []byte {
	return []byte(fmt.Sprintf("#!/bin/bash\nset -e\ndocker start %s\n", container))
}

func StopScript(container string) []byte {
	return []byte(fmt.Sprintf("#!/bin/bash\nset -e\ndocker stop %s\n", container))
}

func RestartScript(container string) []byte {
	return []byte(fmt.Sprintf("#!/bin/bash\nset -e\ndocker restart %s\n", container))
}

// RemoveScript removes the container. When purge=true, the host data dir is also deleted.
func RemoveScript(container, dataDir string, purge bool) []byte {
	script := fmt.Sprintf("#!/bin/bash\nset -e\ndocker rm -f %s 2>/dev/null || true\n", container)
	if purge {
		script += fmt.Sprintf("rm -rf %s\n", dataDir)
	}
	return []byte(script)
}

// LogsScript returns `docker logs` with optional follow/tail flags.
func LogsScript(container string, follow bool, lines int) []byte {
	cmd := "docker logs"
	if follow {
		cmd += " -f"
	}
	if lines > 0 {
		cmd += fmt.Sprintf(" --tail %d", lines)
	}
	cmd += " " + container
	return []byte("#!/bin/bash\nset -e\n" + cmd + "\n")
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/proxy/ -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/proxy/bootstrap.go internal/proxy/bootstrap_test.go
git commit -m "feat(proxy): generate container bootstrap scripts"
```

---

## Phase 4: `conoha proxy` command group

### Task 8: Proxy command group skeleton + shared plumbing

**Files:**
- Create: `cmd/proxy/proxy.go`
- Modify: `cmd/root.go`

- [ ] **Step 1: Create command group file**

Create `cmd/proxy/proxy.go`:

```go
// Package proxy implements the `conoha proxy` command group for managing
// the conoha-proxy container on a ConoHa VPS.
package proxy

import (
	"fmt"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// Cmd is the `conoha proxy` command group.
var Cmd = &cobra.Command{
	Use:   "proxy",
	Short: "Manage conoha-proxy reverse proxy on a ConoHa VPS",
}

// Defaults shared across subcommands.
const (
	DefaultImage     = "ghcr.io/crowdy/conoha-proxy:latest"
	DefaultDataDir   = "/var/lib/conoha-proxy"
	DefaultContainer = "conoha-proxy"
)

// SocketPath derives the admin socket path from the data directory.
func SocketPath(dataDir string) string {
	return dataDir + "/admin.sock"
}

// proxyContext bundles a live SSH connection with identifying metadata.
type proxyContext struct {
	Client *ssh.Client
	Server *model.Server
	IP     string
	User   string
}

// connect opens an SSH connection to the server named in args[0].
// Caller must Close() the returned client.
func connect(cmd *cobra.Command, args []string) (*proxyContext, error) {
	client, err := cmdutil.NewClient(cmd)
	if err != nil {
		return nil, err
	}
	compute := api.NewComputeAPI(client)
	s, err := compute.FindServer(args[0])
	if err != nil {
		return nil, err
	}
	ip, err := internalssh.ServerIP(s)
	if err != nil {
		return nil, err
	}

	user, _ := cmd.Flags().GetString("user")
	port, _ := cmd.Flags().GetString("port")
	identity, _ := cmd.Flags().GetString("identity")
	if identity == "" {
		identity = internalssh.ResolveKeyPath(s.KeyName)
	}
	if identity == "" {
		return nil, fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
	}
	sshClient, err := internalssh.Connect(internalssh.ConnectConfig{
		Host: ip, Port: port, User: user, KeyPath: identity,
	})
	if err != nil {
		return nil, fmt.Errorf("SSH connect: %w", err)
	}
	return &proxyContext{Client: sshClient, Server: s, IP: ip, User: user}, nil
}

// addSSHFlags registers the connection flags common to every proxy subcommand.
func addSSHFlags(c *cobra.Command) {
	c.Flags().StringP("user", "l", "root", "SSH user")
	c.Flags().StringP("port", "p", "22", "SSH port")
	c.Flags().StringP("identity", "i", "", "SSH private key path")
}

// newAdminClient returns a proxy Admin API client wired to the SSH connection.
func newAdminClient(ctx *proxyContext, dataDir string) *proxypkg.Client {
	exec := &proxypkg.SSHExecutor{Client: ctx.Client}
	return proxypkg.NewClient(exec, SocketPath(dataDir))
}
```

- [ ] **Step 2: Register group in root**

Edit `cmd/root.go`, add import and registration. Locate the existing import block that includes `"github.com/crowdy/conoha-cli/cmd/app"` and add:

```go
	"github.com/crowdy/conoha-cli/cmd/proxy"
```

And in `init()`, next to `rootCmd.AddCommand(app.Cmd)`, add:

```go
	rootCmd.AddCommand(proxy.Cmd)
```

- [ ] **Step 3: Verify compilation**

Run: `go build ./...`
Expected: success (no subcommands yet)

- [ ] **Step 4: Commit**

```bash
git add cmd/proxy/proxy.go cmd/root.go
git commit -m "feat(cmd/proxy): add proxy command group skeleton"
```

---

### Task 9: `conoha proxy boot`

**Files:**
- Create: `cmd/proxy/boot.go`
- Modify: `cmd/proxy/proxy.go` (init registration)

- [ ] **Step 1: Implement boot**

Create `cmd/proxy/boot.go`:

```go
package proxy

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

var bootCmd = &cobra.Command{
	Use:   "boot <server>",
	Short: "Install and start conoha-proxy on the server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("acme-email")
		if email == "" {
			return fmt.Errorf("--acme-email is required")
		}
		image, _ := cmd.Flags().GetString("image")
		dataDir, _ := cmd.Flags().GetString("data-dir")
		container, _ := cmd.Flags().GetString("container")

		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		fmt.Fprintf(os.Stderr, "==> Booting conoha-proxy on %s (%s)\n", ctx.Server.Name, ctx.IP)
		script := proxypkg.BootScript(proxypkg.BootParams{
			Email: email, Image: image, DataDir: dataDir, Container: container,
		})
		code, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("boot script: %w", err)
		}
		if code != 0 {
			return fmt.Errorf("boot script exited with %d", code)
		}
		fmt.Fprintln(os.Stderr, "Boot complete.")
		return nil
	},
}

func init() {
	addSSHFlags(bootCmd)
	bootCmd.Flags().String("acme-email", "", "email for Let's Encrypt registration (required)")
	bootCmd.Flags().String("image", DefaultImage, "conoha-proxy docker image")
	bootCmd.Flags().String("data-dir", DefaultDataDir, "host data directory")
	bootCmd.Flags().String("container", DefaultContainer, "docker container name")
	Cmd.AddCommand(bootCmd)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: success

- [ ] **Step 3: Smoke-test CLI surface**

Run: `go run . proxy boot --help`
Expected: help text showing `--acme-email`, `--image`, `--data-dir`, `--container`, `--user`, `--port`, `--identity`.

- [ ] **Step 4: Commit**

```bash
git add cmd/proxy/boot.go
git commit -m "feat(cmd/proxy): add boot subcommand"
```

---

### Task 10: `conoha proxy reboot|start|stop|restart|remove`

**Files:**
- Create: `cmd/proxy/lifecycle.go`

- [ ] **Step 1: Implement lifecycle subcommands**

Create `cmd/proxy/lifecycle.go`:

```go
package proxy

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// runScriptOnServer is the common shape of every lifecycle subcommand:
// connect → run script → close.
func runScriptOnServer(cmd *cobra.Command, args []string, desc string, scriptFn func(container string) []byte) error {
	container, _ := cmd.Flags().GetString("container")
	ctx, err := connect(cmd, args)
	if err != nil {
		return err
	}
	defer func() { _ = ctx.Client.Close() }()
	fmt.Fprintf(os.Stderr, "==> %s on %s (%s)\n", desc, ctx.Server.Name, ctx.IP)
	code, err := internalssh.RunScript(ctx.Client, scriptFn(container), nil, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("script exited with %d", code)
	}
	return nil
}

var rebootCmd = &cobra.Command{
	Use:   "reboot <server>",
	Short: "Pull the latest image and restart conoha-proxy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("acme-email")
		if email == "" {
			return fmt.Errorf("--acme-email is required")
		}
		image, _ := cmd.Flags().GetString("image")
		dataDir, _ := cmd.Flags().GetString("data-dir")
		container, _ := cmd.Flags().GetString("container")
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		fmt.Fprintf(os.Stderr, "==> Rebooting conoha-proxy on %s (%s)\n", ctx.Server.Name, ctx.IP)
		script := proxypkg.RebootScript(proxypkg.BootParams{
			Email: email, Image: image, DataDir: dataDir, Container: container,
		})
		code, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("reboot script exited with %d", code)
		}
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start <server>",
	Short: "Start the conoha-proxy container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScriptOnServer(cmd, args, "Starting conoha-proxy", proxypkg.StartScript)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <server>",
	Short: "Stop the conoha-proxy container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScriptOnServer(cmd, args, "Stopping conoha-proxy", proxypkg.StopScript)
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart <server>",
	Short: "Restart the conoha-proxy container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScriptOnServer(cmd, args, "Restarting conoha-proxy", proxypkg.RestartScript)
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <server>",
	Short: "Remove the conoha-proxy container (volume is kept unless --purge)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		container, _ := cmd.Flags().GetString("container")
		dataDir, _ := cmd.Flags().GetString("data-dir")
		purge, _ := cmd.Flags().GetBool("purge")
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		fmt.Fprintf(os.Stderr, "==> Removing conoha-proxy on %s (purge=%v)\n", ctx.Server.Name, purge)
		code, err := internalssh.RunScript(ctx.Client, proxypkg.RemoveScript(container, dataDir, purge), nil, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("remove script exited with %d", code)
		}
		return nil
	},
}

func init() {
	addSSHFlags(rebootCmd)
	rebootCmd.Flags().String("acme-email", "", "email for Let's Encrypt registration (required)")
	rebootCmd.Flags().String("image", DefaultImage, "conoha-proxy docker image")
	rebootCmd.Flags().String("data-dir", DefaultDataDir, "host data directory")
	rebootCmd.Flags().String("container", DefaultContainer, "docker container name")

	for _, c := range []*cobra.Command{startCmd, stopCmd, restartCmd} {
		addSSHFlags(c)
		c.Flags().String("container", DefaultContainer, "docker container name")
	}

	addSSHFlags(removeCmd)
	removeCmd.Flags().String("container", DefaultContainer, "docker container name")
	removeCmd.Flags().String("data-dir", DefaultDataDir, "host data directory")
	removeCmd.Flags().Bool("purge", false, "also delete the host data directory")

	Cmd.AddCommand(rebootCmd, startCmd, stopCmd, restartCmd, removeCmd)
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: success

- [ ] **Step 3: Smoke test**

Run: `go run . proxy --help`
Expected: boot, reboot, start, stop, restart, remove subcommands listed.

- [ ] **Step 4: Commit**

```bash
git add cmd/proxy/lifecycle.go
git commit -m "feat(cmd/proxy): add reboot/start/stop/restart/remove"
```

---

### Task 11: `conoha proxy logs|details|services`

**Files:**
- Create: `cmd/proxy/observability.go`

- [ ] **Step 1: Implement observability subcommands**

Create `cmd/proxy/observability.go`:

```go
package proxy

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

var logsCmd = &cobra.Command{
	Use:   "logs <server>",
	Short: "Show conoha-proxy container logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		container, _ := cmd.Flags().GetString("container")
		follow, _ := cmd.Flags().GetBool("follow")
		linesStr, _ := cmd.Flags().GetString("tail")
		lines, _ := strconv.Atoi(linesStr)
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		code, err := internalssh.RunScript(ctx.Client, proxypkg.LogsScript(container, follow, lines), nil, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("logs script exited with %d", code)
		}
		return nil
	},
}

var detailsCmd = &cobra.Command{
	Use:   "details <server>",
	Short: "Show conoha-proxy version and readiness",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		client := newAdminClient(ctx, dataDir)
		services, listErr := client.List()

		exec := &proxypkg.SSHExecutor{Client: ctx.Client}
		versionBody, vErr := curlVia(exec, dataDir, "/version")
		readyBody, rErr := curlVia(exec, dataDir, "/readyz")

		fmt.Printf("Server:  %s (%s)\n", ctx.Server.Name, ctx.IP)
		fmt.Printf("Version: %s\n", jsonField(versionBody, "version", vErr))
		fmt.Printf("Ready:   %s\n", jsonField(readyBody, "status", rErr))
		if listErr != nil {
			fmt.Printf("Services: (error: %v)\n", listErr)
		} else {
			fmt.Printf("Services: %d registered\n", len(services))
		}
		return nil
	},
}

var servicesCmd = &cobra.Command{
	Use:   "services <server>",
	Short: "List proxy services registered on the server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		client := newAdminClient(ctx, dataDir)
		services, err := client.List()
		if err != nil {
			return err
		}
		fmt.Printf("%-20s %-10s %-30s %s\n", "NAME", "PHASE", "ACTIVE", "HOSTS")
		for _, s := range services {
			active := "-"
			if s.ActiveTarget != nil {
				active = s.ActiveTarget.URL
			}
			hosts := ""
			for i, h := range s.Hosts {
				if i > 0 {
					hosts += ","
				}
				hosts += h
			}
			fmt.Printf("%-20s %-10s %-30s %s\n", s.Name, s.Phase, active, hosts)
		}
		return nil
	},
}

// curlVia is a tiny helper for non-/v1/ endpoints (/version, /readyz)
// that don't return the proxy error envelope.
func curlVia(exec proxypkg.Executor, dataDir, path string) ([]byte, error) {
	cmd := fmt.Sprintf("curl -sS --unix-socket %s/admin.sock http://admin%s", dataDir, path)
	var buf []byte
	w := byteWriter{&buf}
	if err := exec.Run(cmd, nil, &w); err != nil {
		return nil, err
	}
	return buf, nil
}

type byteWriter struct{ b *[]byte }

func (w *byteWriter) Write(p []byte) (int, error) { *w.b = append(*w.b, p...); return len(p), nil }

func jsonField(body []byte, key string, err error) string {
	if err != nil {
		return fmt.Sprintf("(error: %v)", err)
	}
	var m map[string]interface{}
	if uErr := json.Unmarshal(body, &m); uErr != nil {
		return fmt.Sprintf("(decode error: %v)", uErr)
	}
	v, ok := m[key]
	if !ok {
		return "(missing)"
	}
	return fmt.Sprint(v)
}

func init() {
	addSSHFlags(logsCmd)
	logsCmd.Flags().String("container", DefaultContainer, "docker container name")
	logsCmd.Flags().BoolP("follow", "f", false, "follow log output")
	logsCmd.Flags().String("tail", "0", "number of lines to show (0 = all)")

	for _, c := range []*cobra.Command{detailsCmd, servicesCmd} {
		addSSHFlags(c)
		c.Flags().String("data-dir", DefaultDataDir, "host data directory")
	}

	Cmd.AddCommand(logsCmd, detailsCmd, servicesCmd)
}
```

- [ ] **Step 2: Build**

Run: `go build ./...`
Expected: success

- [ ] **Step 3: Commit**

```bash
git add cmd/proxy/observability.go
git commit -m "feat(cmd/proxy): add logs/details/services subcommands"
```

---

## Phase 5: `conoha app init` refactor

### Task 12: Rewrite `app init` to register service with proxy

**Files:**
- Modify: `cmd/app/init.go`
- Modify: `cmd/app/init_test.go` (delete old generateInitScript tests, add new flow tests)
- Modify: `cmd/app/app.go` (no-op; stays as-is)

- [ ] **Step 1: Understand current init_test.go**

Open and review `cmd/app/init_test.go`. Record which tests cover `generateInitScript` — these will be deleted.

- [ ] **Step 2: Rewrite init.go**

Replace the entire contents of `cmd/app/init.go`:

```go
package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
	"github.com/crowdy/conoha-cli/internal/model"
	"golang.org/x/crypto/ssh"
)

func init() {
	addAppFlags(initCmd)
	initCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
}

var initCmd = &cobra.Command{
	Use:   "init <server>",
	Short: "Register the app's conoha.yml with conoha-proxy on the server",
	Long: `Read conoha.yml from the current directory, verify the server has Docker
and a running conoha-proxy, and upsert the service (name, hosts, health policy)
against the proxy's Admin API.

Run 'conoha proxy boot' on the server first.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pf, err := config.LoadProjectFile(config.ProjectFileName)
		if err != nil {
			return err
		}
		if err := pf.Validate(); err != nil {
			return err
		}

		sshClient, s, ip, err := connectToServer(cmd, args[0])
		if err != nil {
			return err
		}
		defer func() { _ = sshClient.Close() }()

		dataDir, _ := cmd.Flags().GetString("data-dir")
		client := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: sshClient}, proxy.SocketPath(dataDir))

		if err := warnOnLegacyRepo(sshClient, pf.Name); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %v\n", err)
		}

		fmt.Fprintf(os.Stderr, "==> Registering service %q on %s (%s)\n", pf.Name, s.Name, ip)
		svc, err := client.Upsert(proxypkg.UpsertRequest{
			Name:         pf.Name,
			Hosts:        pf.Hosts,
			HealthPolicy: mapHealth(pf.Health),
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Service %q registered. phase=%s tls=%s\n", svc.Name, svc.Phase, svc.TLSStatus)
		fmt.Fprintf(os.Stderr, "Next: run 'conoha app deploy %s' to push your app.\n", args[0])
		return nil
	},
}

// connectToServer opens an SSH session to the server identified by id-or-name.
// Returns the client, the resolved server, and its public IP.
func connectToServer(cmd *cobra.Command, idOrName string) (*ssh.Client, *model.Server, string, error) {
	apiClient, err := cmdutil.NewClient(cmd)
	if err != nil {
		return nil, nil, "", err
	}
	compute := api.NewComputeAPI(apiClient)
	s, err := compute.FindServer(idOrName)
	if err != nil {
		return nil, nil, "", err
	}
	ip, err := internalssh.ServerIP(s)
	if err != nil {
		return nil, nil, "", err
	}
	user, _ := cmd.Flags().GetString("user")
	port, _ := cmd.Flags().GetString("port")
	identity, _ := cmd.Flags().GetString("identity")
	if identity == "" {
		identity = internalssh.ResolveKeyPath(s.KeyName)
	}
	if identity == "" {
		return nil, nil, "", fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
	}
	cli, err := internalssh.Connect(internalssh.ConnectConfig{
		Host: ip, Port: port, User: user, KeyPath: identity,
	})
	if err != nil {
		return nil, nil, "", fmt.Errorf("SSH connect: %w", err)
	}
	return cli, s, ip, nil
}

// mapHealth copies project-file health settings into the proxy request shape.
func mapHealth(h *config.HealthSpec) *proxypkg.HealthPolicy {
	if h == nil {
		return nil
	}
	return &proxypkg.HealthPolicy{
		Path:               h.Path,
		IntervalMs:         h.IntervalMs,
		TimeoutMs:          h.TimeoutMs,
		HealthyThreshold:   h.HealthyThreshold,
		UnhealthyThreshold: h.UnhealthyThreshold,
	}
}

// warnOnLegacyRepo checks for the old /opt/conoha/<name>.git bare repo and
// returns a non-nil (non-fatal) error if present, so users can migrate cleanly.
func warnOnLegacyRepo(cli *ssh.Client, name string) error {
	cmdStr := fmt.Sprintf("test -d /opt/conoha/%s.git && echo yes || echo no", name)
	var buf byteBuf
	_, err := internalssh.RunCommand(cli, cmdStr, &buf, os.Stderr)
	if err != nil {
		return nil
	}
	if string(buf.b) == "yes\n" {
		return fmt.Errorf("legacy git bare repo /opt/conoha/%s.git exists (left untouched). Remove it after migration with 'rm -rf /opt/conoha/%s.git' via SSH", name, name)
	}
	return nil
}

type byteBuf struct{ b []byte }

func (w *byteBuf) Write(p []byte) (int, error) { w.b = append(w.b, p...); return len(p), nil }
```

- [ ] **Step 3: Rewrite init_test.go**

Replace `cmd/app/init_test.go` with:

```go
package app

import (
	"testing"

	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

func TestMapHealth_Nil(t *testing.T) {
	if got := mapHealth(nil); got != nil {
		t.Errorf("want nil, got %+v", got)
	}
}

func TestMapHealth_AllFields(t *testing.T) {
	in := &config.HealthSpec{
		Path: "/up", IntervalMs: 1000, TimeoutMs: 500,
		HealthyThreshold: 2, UnhealthyThreshold: 5,
	}
	want := &proxypkg.HealthPolicy{
		Path: "/up", IntervalMs: 1000, TimeoutMs: 500,
		HealthyThreshold: 2, UnhealthyThreshold: 5,
	}
	got := mapHealth(in)
	if *got != *want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 4: Build and test**

Run: `go build ./... && go test ./cmd/app/ -run TestMapHealth -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/app/init.go cmd/app/init_test.go
git commit -m "refactor(app/init): register service with conoha-proxy instead of git repo"
```

---

## Phase 6: `conoha app deploy` replacement

### Task 13: Slot ID determination

**Files:**
- Create: `cmd/app/slot.go`
- Create: `cmd/app/slot_test.go`

- [ ] **Step 1: Write tests**

Create `cmd/app/slot_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestDetermineSlotID_Timestamp(t *testing.T) {
	id, err := determineSlotID(".", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 14 {
		t.Errorf("expected 14-char timestamp, got %q (%d chars)", id, len(id))
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			t.Errorf("non-digit %q in %q", r, id)
		}
	}
}

func TestSuffixIfTaken(t *testing.T) {
	taken := map[string]bool{"abc1234": true, "abc1234-2": true}
	got := suffixIfTaken("abc1234", func(s string) bool { return taken[s] })
	if got != "abc1234-3" {
		t.Errorf("got %q, want abc1234-3", got)
	}
	got = suffixIfTaken("fresh", func(s string) bool { return false })
	if got != "fresh" {
		t.Errorf("got %q, want fresh", got)
	}
}

func TestDetermineSlotID_GitShortSHA(t *testing.T) {
	// This test runs inside our own repo, so git IS available. If not, skip.
	id, err := determineSlotID(".", true)
	if err != nil {
		t.Skipf("git not available in test env: %v", err)
	}
	if len(id) != 7 {
		t.Errorf("expected 7-char short SHA, got %q", id)
	}
	if strings.ContainsAny(id, " \n\r") {
		t.Errorf("whitespace in %q", id)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/app/ -run 'TestDetermineSlotID|TestSuffixIfTaken' -v`
Expected: FAIL (undefined)

- [ ] **Step 3: Implement slot.go**

Create `cmd/app/slot.go`:

```go
package app

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// determineSlotID returns either the repo's 7-char HEAD short SHA (when useGit)
// or a timestamp "YYYYMMDDHHMMSS" otherwise. useGit=false is used when the caller
// knows or has decided git isn't appropriate.
func determineSlotID(dir string, useGit bool) (string, error) {
	if useGit {
		cmd := exec.Command("git", "-C", dir, "rev-parse", "--short=7", "HEAD")
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git rev-parse: %w", err)
		}
		sha := strings.TrimSpace(string(out))
		if len(sha) != 7 {
			return "", fmt.Errorf("unexpected short SHA %q", sha)
		}
		return sha, nil
	}
	return time.Now().UTC().Format("20060102150405"), nil
}

// suffixIfTaken returns base unchanged when taken(base) is false.
// Otherwise it appends -2, -3, ... until an unused name is found.
func suffixIfTaken(base string, taken func(string) bool) string {
	if !taken(base) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !taken(candidate) {
			return candidate
		}
	}
}

// IsGitRepo reports whether dir is inside a git work tree.
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/app/ -run 'TestDetermineSlotID|TestSuffixIfTaken' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/app/slot.go cmd/app/slot_test.go
git commit -m "feat(app): slot ID from git SHA or timestamp"
```

---

### Task 14: Compose override generator

**Files:**
- Create: `cmd/app/override.go`
- Create: `cmd/app/override_test.go`

- [ ] **Step 1: Write tests**

Create `cmd/app/override_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestComposeOverride_WebPortAndName(t *testing.T) {
	got := composeOverride("myapp", "a1b2c3d", "web", 8080)
	want := []string{
		`services:`,
		`  web:`,
		`    container_name: myapp-a1b2c3d-web`,
		`    ports:`,
		`      - "127.0.0.1:0:8080"`,
	}
	for _, line := range want {
		if !strings.Contains(got, line) {
			t.Errorf("missing line %q in:\n%s", line, got)
		}
	}
}

func TestAccessoryProjectName(t *testing.T) {
	if got := accessoryProjectName("myapp"); got != "myapp-accessories" {
		t.Errorf("got %q", got)
	}
}

func TestSlotProjectName(t *testing.T) {
	if got := slotProjectName("myapp", "a1b2c3d"); got != "myapp-a1b2c3d" {
		t.Errorf("got %q", got)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/app/ -run 'TestComposeOverride|TestAccessoryProjectName|TestSlotProjectName' -v`
Expected: FAIL

- [ ] **Step 3: Implement override.go**

Create `cmd/app/override.go`:

```go
package app

import "fmt"

// composeOverride returns a compose override document (YAML) that:
//   - pins the web service's container name to <project>-<slot>-<web>
//   - maps 127.0.0.1:0:<port> so the kernel picks a free host port
//
// It does not touch any other service; accessories keep their compose-declared
// container_name and ports.
func composeOverride(app, slot, webService string, webPort int) string {
	return fmt.Sprintf(`services:
  %[3]s:
    container_name: %[1]s-%[2]s-%[3]s
    ports:
      - "127.0.0.1:0:%[4]d"
`, app, slot, webService, webPort)
}

// slotProjectName is the compose -p value for a blue/green slot.
func slotProjectName(app, slot string) string {
	return fmt.Sprintf("%s-%s", app, slot)
}

// accessoryProjectName is the compose -p value for the persistent accessory stack.
func accessoryProjectName(app string) string {
	return app + "-accessories"
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/app/ -run 'TestComposeOverride|TestAccessoryProjectName|TestSlotProjectName' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/app/override.go cmd/app/override_test.go
git commit -m "feat(app): generate compose override for slot isolation + dynamic port"
```

---

### Task 15: Remote shell fragments — upload, port discovery, drain teardown

**Files:**
- Create: `cmd/app/remotecmds.go`
- Create: `cmd/app/remotecmds_test.go`

- [ ] **Step 1: Write tests**

Create `cmd/app/remotecmds_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestBuildSlotUploadCmd(t *testing.T) {
	got := buildSlotUploadCmd("/opt/conoha/myapp/abc1234", "127.0.0.1")
	for _, want := range []string{
		"rm -rf '/opt/conoha/myapp/abc1234'",
		"mkdir -p '/opt/conoha/myapp/abc1234'",
		"tar xzf - -C '/opt/conoha/myapp/abc1234'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildComposeUp_Slot(t *testing.T) {
	got := buildSlotComposeUp("/opt/conoha/myapp/abc1234", "myapp-abc1234", "compose.yml", "override.yml", "web")
	for _, want := range []string{
		"cd '/opt/conoha/myapp/abc1234'",
		"docker compose -p myapp-abc1234 -f compose.yml -f override.yml",
		"up -d --build web",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildDockerPort(t *testing.T) {
	got := buildDockerPortCmd("myapp-abc1234-web", 8080)
	if !strings.Contains(got, "docker port myapp-abc1234-web 8080") {
		t.Errorf("got %s", got)
	}
}

func TestExtractHostPort(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{
		{"127.0.0.1:49231\n", 49231, true},
		{"0.0.0.0:49231\n127.0.0.1:49231\n", 49231, true},
		{"", 0, false},
		{"garbage", 0, false},
	}
	for _, c := range cases {
		got, err := extractHostPort(c.in)
		if c.ok && err != nil {
			t.Errorf("in=%q got err=%v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("in=%q expected error", c.in)
		}
		if got != c.want {
			t.Errorf("in=%q got %d, want %d", c.in, got, c.want)
		}
	}
}

func TestBuildScheduleDrainCmd(t *testing.T) {
	got := buildScheduleDrainCmd("/opt/conoha/myapp/old", "myapp-old", 30000)
	if !strings.Contains(got, "sleep 30") {
		t.Errorf("expected sleep 30s, got %s", got)
	}
	if !strings.Contains(got, "docker compose -p myapp-old") {
		t.Errorf("missing project in %s", got)
	}
	if !strings.Contains(got, "down") {
		t.Errorf("missing down in %s", got)
	}
	if !strings.Contains(got, "nohup") {
		t.Errorf("expected nohup in %s", got)
	}
}

func TestBuildAccessoryUp(t *testing.T) {
	got := buildAccessoryUp("/opt/conoha/myapp/abc1234", "myapp-accessories", "compose.yml", []string{"db", "redis"})
	for _, want := range []string{
		"docker compose -p myapp-accessories",
		"-f compose.yml",
		"up -d db redis",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./cmd/app/ -run 'TestBuildSlot|TestBuildCompose|TestBuildDocker|TestExtractHost|TestBuildSchedule|TestBuildAccessory' -v`
Expected: FAIL

- [ ] **Step 3: Implement remotecmds.go**

Create `cmd/app/remotecmds.go`:

```go
package app

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// buildSlotUploadCmd extracts the incoming tar archive into the slot-specific
// work directory. .env preservation from the old single-slot flow is NOT
// applied here — env handling belongs to accessories or app.env commands.
func buildSlotUploadCmd(workDir, _ string) string {
	return fmt.Sprintf(
		"rm -rf '%[1]s' && mkdir -p '%[1]s' && tar xzf - -C '%[1]s'",
		workDir)
}

// buildSlotComposeUp starts the single web service inside a slot-scoped project.
func buildSlotComposeUp(workDir, project, composeFile, overrideFile, webService string) string {
	return fmt.Sprintf(
		"cd '%s' && docker compose -p %s -f %s -f %s up -d --build %s",
		workDir, project, composeFile, overrideFile, webService)
}

// buildDockerPortCmd produces a command that prints the host:port mapping
// for the web container's internal port.
func buildDockerPortCmd(containerName string, port int) string {
	return fmt.Sprintf("docker port %s %d", containerName, port)
}

var hostPortRe = regexp.MustCompile(`(?m)^(?:\d+\.\d+\.\d+\.\d+|\[::\]|\[::1\]):(\d+)`)

// extractHostPort parses "docker port" output and returns the first loopback
// (127.0.0.1) mapping if present, otherwise the first IPv4/IPv6 mapping found.
// Returns an error if no mapping line exists.
func extractHostPort(out string) (int, error) {
	// Prefer 127.0.0.1 lines explicitly.
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "127.0.0.1:") {
			return parseColonPort(line)
		}
	}
	m := hostPortRe.FindStringSubmatch(out)
	if m == nil {
		return 0, fmt.Errorf("no host port in docker port output: %q", out)
	}
	return strconv.Atoi(m[1])
}

func parseColonPort(line string) (int, error) {
	line = strings.TrimSpace(line)
	i := strings.LastIndex(line, ":")
	if i < 0 {
		return 0, fmt.Errorf("no colon in %q", line)
	}
	return strconv.Atoi(line[i+1:])
}

// buildScheduleDrainCmd fires a detached shell that, after drainMs, brings the
// old slot down with `docker compose down`. Uses nohup + background shell;
// does not rely on `at` availability.
func buildScheduleDrainCmd(workDir, project string, drainMs int) string {
	seconds := drainMs / 1000
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf(
		`nohup bash -c "sleep %d && cd '%s' && docker compose -p %s down" >/dev/null 2>&1 & disown`,
		seconds, workDir, project)
}

// buildAccessoryUp starts the accessories listed, using a dedicated compose
// project so they survive slot teardown. WorkDir is the slot's work directory —
// we read the compose file from there because accessories share the same file.
func buildAccessoryUp(workDir, project, composeFile string, accessories []string) string {
	args := strings.Join(accessories, " ")
	return fmt.Sprintf(
		"cd '%s' && docker compose -p %s -f %s up -d %s",
		workDir, project, composeFile, args)
}

// buildAccessoryExists reports (via shell exit 0/1) whether the accessory project
// has any containers. Exit 0 means "already up".
func buildAccessoryExists(project string) string {
	return fmt.Sprintf(
		`[ "$(docker compose -p %s ps -q | wc -l)" -gt 0 ]`,
		project)
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/app/ -run 'TestBuildSlot|TestBuildCompose|TestBuildDocker|TestExtractHost|TestBuildSchedule|TestBuildAccessory' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/app/remotecmds.go cmd/app/remotecmds_test.go
git commit -m "feat(app): remote shell fragments for blue/green flow"
```

---

### Task 16: Rewrite `app deploy`

**Files:**
- Modify: `cmd/app/deploy.go`
- Modify: `cmd/app/deploy_test.go` (delete old, add new)

- [ ] **Step 1: Delete obsolete deploy tests**

Open `cmd/app/deploy_test.go` and delete every test. The old tests cover `buildUploadCmd`, `buildComposeCmd`, `validateComposeFilePath`, `detectComposeFile` — those functions are gone in the new model.

Leave only the `package app` declaration:

```go
package app
```

- [ ] **Step 2: Replace deploy.go with the complete implementation**

Replace the entire contents of `cmd/app/deploy.go`:

```go
package app

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(deployCmd)
	deployCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	deployCmd.Flags().String("slot", "", "override slot ID (default: git short SHA or timestamp)")
}

var deployCmd = &cobra.Command{
	Use:   "deploy <server>",
	Short: "Deploy the current directory via conoha-proxy blue/green",
	Long: `Archive the current directory, upload via SSH, start the web container
in a new compose slot on a dynamic port, then ask conoha-proxy to probe and
swap. The previous slot is torn down after the drain window.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeploy(cmd, args[0])
	},
}

func runDeploy(cmd *cobra.Command, serverID string) error {
	pf, err := config.LoadProjectFile(config.ProjectFileName)
	if err != nil {
		return err
	}
	if err := pf.Validate(); err != nil {
		return err
	}
	composeFile, err := pf.ResolveComposeFile(".")
	if err != nil {
		return err
	}

	sshClient, s, ip, err := connectToServer(cmd, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sshClient.Close() }()

	dataDir, _ := cmd.Flags().GetString("data-dir")
	admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: sshClient}, proxy.SocketPath(dataDir))

	// Service must exist — init registers it. Missing = user skipped init.
	if _, err := admin.Get(pf.Name); err != nil {
		return fmt.Errorf("service %q not found on proxy — run 'conoha app init %s' first: %w", pf.Name, serverID, err)
	}

	slotOverride, _ := cmd.Flags().GetString("slot")
	slot := slotOverride
	if slot == "" {
		slot, err = determineSlotID(".", IsGitRepo("."))
		if err != nil {
			return err
		}
	}

	fmt.Fprintf(os.Stderr, "==> Deploying %q to %s (%s)\n", pf.Name, s.Name, ip)
	fmt.Fprintf(os.Stderr, "==> Slot: %s (compose project: %s)\n", slot, slotProjectName(pf.Name, slot))

	// Upload archive to slot work dir.
	patterns, err := loadIgnorePatterns(".")
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := createTarGz(".", patterns, &buf); err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	slotWork := fmt.Sprintf("/opt/conoha/%s/%s", pf.Name, slot)
	if err := runRemote(sshClient, buildSlotUploadCmd(slotWork, ""), &buf); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	// Write compose override into the slot dir.
	overrideContent := composeOverride(pf.Name, slot, pf.Web.Service, pf.Web.Port)
	overridePath := "conoha-override.yml"
	writeOverride := fmt.Sprintf("cat > '%s/%s' <<'EOF'\n%sEOF", slotWork, overridePath, overrideContent)
	if err := runRemote(sshClient, writeOverride, nil); err != nil {
		return fmt.Errorf("write override: %w", err)
	}

	// First-run: bring up accessories (idempotent via existence probe).
	if len(pf.Accessories) > 0 {
		check := buildAccessoryExists(accessoryProjectName(pf.Name))
		code, _ := internalssh.RunCommand(sshClient, check, os.Stderr, os.Stderr)
		if code != 0 {
			fmt.Fprintf(os.Stderr, "==> Starting accessories: %v\n", pf.Accessories)
			if err := runRemote(sshClient, buildAccessoryUp(slotWork, accessoryProjectName(pf.Name), composeFile, pf.Accessories), nil); err != nil {
				return fmt.Errorf("accessory up: %w", err)
			}
		}
	}

	// Start the new slot's web service.
	fmt.Fprintf(os.Stderr, "==> Building and starting %s in new slot\n", pf.Web.Service)
	if err := runRemote(sshClient, buildSlotComposeUp(slotWork, slotProjectName(pf.Name, slot), composeFile, overridePath, pf.Web.Service), nil); err != nil {
		return fmt.Errorf("compose up (slot): %w", err)
	}

	// Discover kernel-picked host port.
	containerName := fmt.Sprintf("%s-%s-%s", pf.Name, slot, pf.Web.Service)
	var portOut bytes.Buffer
	if _, err := internalssh.RunCommand(sshClient, buildDockerPortCmd(containerName, pf.Web.Port), &portOut, os.Stderr); err != nil {
		tearDownSlot(sshClient, pf.Name, slot)
		return fmt.Errorf("docker port: %w", err)
	}
	hostPort, err := extractHostPort(portOut.String())
	if err != nil {
		tearDownSlot(sshClient, pf.Name, slot)
		return err
	}
	targetURL := fmt.Sprintf("http://127.0.0.1:%d", hostPort)
	fmt.Fprintf(os.Stderr, "==> Host port: %d. Calling proxy /deploy\n", hostPort)

	drainMs := 30000
	if pf.Deploy != nil && pf.Deploy.DrainMs > 0 {
		drainMs = pf.Deploy.DrainMs
	}

	// Call proxy /deploy. On 424 the proxy did not mutate state — tear down new slot.
	updated, err := admin.Deploy(pf.Name, proxypkg.DeployRequest{TargetURL: targetURL, DrainMs: drainMs})
	if err != nil {
		tearDownSlot(sshClient, pf.Name, slot)
		return err
	}

	// Read old slot pointer (empty on first deploy), then update to current.
	ptrPath := fmt.Sprintf("/opt/conoha/%s/CURRENT_SLOT", pf.Name)
	var ptrBuf bytes.Buffer
	_, _ = internalssh.RunCommand(sshClient, fmt.Sprintf("cat '%s' 2>/dev/null || true", ptrPath), &ptrBuf, os.Stderr)
	oldSlot := strings.TrimSpace(ptrBuf.String())

	if err := runRemote(sshClient, fmt.Sprintf("echo %s > '%s'", slot, ptrPath), nil); err != nil {
		fmt.Fprintf(os.Stderr, "warning: update CURRENT_SLOT pointer: %v\n", err)
	}

	if oldSlot != "" && oldSlot != slot {
		oldWork := fmt.Sprintf("/opt/conoha/%s/%s", pf.Name, oldSlot)
		schedule := buildScheduleDrainCmd(oldWork, slotProjectName(pf.Name, oldSlot), drainMs)
		if err := runRemote(sshClient, schedule, nil); err != nil {
			fmt.Fprintf(os.Stderr, "warning: schedule drain teardown: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "==> Scheduled teardown of old slot %q in %dms\n", oldSlot, drainMs)
		}
	}

	fmt.Fprintf(os.Stderr, "Deploy complete. active=%s phase=%s\n", updated.ActiveTarget.URL, updated.Phase)
	return nil
}

// runRemote runs command on cli. When stdinData is non-nil it is streamed as stdin.
// Returns an error if the remote exit status is not zero.
func runRemote(cli *ssh.Client, command string, stdinData *bytes.Buffer) error {
	var code int
	var err error
	if stdinData != nil {
		code, err = internalssh.RunWithStdin(cli, command, stdinData, os.Stdout, os.Stderr)
	} else {
		code, err = internalssh.RunCommand(cli, command, os.Stdout, os.Stderr)
	}
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("remote exit %d", code)
	}
	return nil
}

// tearDownSlot brings down a slot's compose project and removes its work dir.
// Best-effort; the caller already has a more interesting error to return.
func tearDownSlot(cli *ssh.Client, app, slot string) {
	work := fmt.Sprintf("/opt/conoha/%s/%s", app, slot)
	cmd := fmt.Sprintf(
		"docker compose -p %s -f '%s/conoha-override.yml' down 2>/dev/null || true; rm -rf '%s' || true",
		slotProjectName(app, slot), work, work)
	_, _ = internalssh.RunCommand(cli, cmd, os.Stderr, os.Stderr)
}
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: success

- [ ] **Step 4: Run tests**

Run: `go test ./cmd/app/ -v`
Expected: PASS (all remaining tests)

- [ ] **Step 5: Commit**

```bash
git add cmd/app/deploy.go cmd/app/deploy_test.go
git commit -m "feat(app/deploy): blue/green via conoha-proxy with slot isolation"
```

---

## Phase 7: Rollback, destroy, status updates

### Task 17: `conoha app rollback`

**Files:**
- Create: `cmd/app/rollback.go`
- Modify: `cmd/app/app.go` (register)

- [ ] **Step 1: Implement rollback**

Create `cmd/app/rollback.go`:

```go
package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

func init() {
	addAppFlags(rollbackCmd)
	rollbackCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	rollbackCmd.Flags().Int("drain-ms", 0, "drain window for the swapped-back target (0 = proxy default)")
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback <server>",
	Short: "Swap back to the previous target (within the drain window)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pf, err := config.LoadProjectFile(config.ProjectFileName)
		if err != nil {
			return err
		}
		if err := pf.Validate(); err != nil {
			return err
		}
		sshClient, s, ip, err := connectToServer(cmd, args[0])
		if err != nil {
			return err
		}
		defer func() { _ = sshClient.Close() }()

		dataDir, _ := cmd.Flags().GetString("data-dir")
		admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: sshClient}, proxy.SocketPath(dataDir))

		drainMs, _ := cmd.Flags().GetInt("drain-ms")
		fmt.Fprintf(os.Stderr, "==> Rolling back %q on %s (%s)\n", pf.Name, s.Name, ip)
		updated, err := admin.Rollback(pf.Name, drainMs)
		if err != nil {
			if errors.Is(err, proxypkg.ErrNoDrainTarget) {
				return fmt.Errorf("drain window has closed — redeploy the previous slot (git SHA) instead")
			}
			return err
		}
		fmt.Fprintf(os.Stderr, "Rollback complete. active=%s phase=%s\n", updated.ActiveTarget.URL, updated.Phase)
		return nil
	},
}
```

- [ ] **Step 2: Register in app.go**

Modify `cmd/app/app.go`:

```go
package app

import "github.com/spf13/cobra"

// Cmd is the app command group.
var Cmd = &cobra.Command{
	Use:   "app",
	Short: "Application deployment commands",
}

func init() {
	Cmd.AddCommand(initCmd)
	Cmd.AddCommand(deployCmd)
	Cmd.AddCommand(rollbackCmd)
	Cmd.AddCommand(logsCmd)
	Cmd.AddCommand(statusCmd)
	Cmd.AddCommand(stopCmd)
	Cmd.AddCommand(restartCmd)
	Cmd.AddCommand(envCmd)
	Cmd.AddCommand(destroyCmd)
	Cmd.AddCommand(listCmd)
	Cmd.AddCommand(resetCmd)
}
```

- [ ] **Step 3: Build + smoke test**

Run: `go build ./... && go run . app rollback --help`
Expected: help text visible.

- [ ] **Step 4: Commit**

```bash
git add cmd/app/rollback.go cmd/app/app.go
git commit -m "feat(app): add rollback subcommand"
```

---

### Task 18: Update `conoha app destroy` to deregister from proxy

**Files:**
- Modify: `cmd/app/destroy.go`
- Modify: `cmd/app/destroy_test.go` (likely just adjust or keep empty)

- [ ] **Step 1: Read existing destroy.go**

Open `cmd/app/destroy.go`. Identify the place that runs compose down (the current implementation).

- [ ] **Step 2: Add proxy DELETE call**

After the existing compose-down logic, add a proxy deregister step. Modify `cmd/app/destroy.go` — locate the `RunE` of `destroyCmd`. Inside it, after successful compose down and before the final "complete" message, add:

```go
		// Best-effort deregister from proxy.
		dataDir, _ := cmd.Flags().GetString("data-dir")
		if dataDir == "" {
			dataDir = proxy.DefaultDataDir
		}
		admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: ctx.Client}, proxy.SocketPath(dataDir))
		pf, pfErr := config.LoadProjectFile(config.ProjectFileName)
		if pfErr == nil && pf.Validate() == nil {
			if err := admin.Delete(pf.Name); err != nil {
				fmt.Fprintf(os.Stderr, "warning: proxy delete %s: %v\n", pf.Name, err)
			} else {
				fmt.Fprintf(os.Stderr, "==> Deregistered %q from proxy\n", pf.Name)
			}
		}
```

Also add the missing imports:

```go
import (
	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)
```

Register the `--data-dir` flag in `destroy.go`'s `init()`:

```go
	destroyCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
```

- [ ] **Step 3: Build + test**

Run: `go build ./... && go test ./cmd/app/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add cmd/app/destroy.go
git commit -m "feat(app/destroy): also deregister service from conoha-proxy"
```

---

### Task 19: `conoha app status` — add proxy phase display

**Files:**
- Modify: `cmd/app/status.go`

- [ ] **Step 1: Read existing status.go**

Open `cmd/app/status.go`. Locate where it prints `docker compose ps` output.

- [ ] **Step 2: Add proxy state lookup**

At the beginning of the `RunE` (after SSH connect + compose ps), fetch the proxy service and print its key fields. Insert near the end of the RunE body, before the final return:

```go
		// Query proxy state for added context.
		pf, pfErr := config.LoadProjectFile(config.ProjectFileName)
		if pfErr == nil && pf.Validate() == nil {
			dataDir, _ := cmd.Flags().GetString("data-dir")
			if dataDir == "" {
				dataDir = proxy.DefaultDataDir
			}
			admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: ctx.Client}, proxy.SocketPath(dataDir))
			if svc, err := admin.Get(pf.Name); err == nil {
				fmt.Fprintf(os.Stderr, "\n==> Proxy service %q: phase=%s tls=%s\n", svc.Name, svc.Phase, svc.TLSStatus)
				if svc.ActiveTarget != nil {
					fmt.Fprintf(os.Stderr, "    active:   %s\n", svc.ActiveTarget.URL)
				}
				if svc.DrainingTarget != nil {
					fmt.Fprintf(os.Stderr, "    draining: %s\n", svc.DrainingTarget.URL)
				}
				if svc.DrainDeadline != nil {
					fmt.Fprintf(os.Stderr, "    drain deadline: %s\n", svc.DrainDeadline.Format("2006-01-02 15:04:05 MST"))
				}
			} else {
				fmt.Fprintf(os.Stderr, "\n==> Proxy service %q: (error: %v)\n", pf.Name, err)
			}
		}
```

Add imports (if missing):

```go
	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
```

And the flag in `init()`:

```go
	statusCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
```

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add cmd/app/status.go
git commit -m "feat(app/status): show proxy service phase and targets"
```

---

## Phase 8: Documentation and migration

### Task 20: Update README.md

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add `proxy` to the command table**

In `README.md`, find the "コマンド一覧" table that lists `conoha auth`, `conoha server`, etc. After the `conoha app` row, add:

```markdown
| `conoha proxy` | conoha-proxy リバースプロキシ管理（boot / reboot / start / stop / restart / remove / logs / details / services） |
```

Update the `conoha app` row to include `rollback`:

```markdown
| `conoha app` | アプリデプロイ・管理（init / deploy / rollback / logs / status / stop / restart / env / destroy / reset / list） |
```

- [ ] **Step 2: Add a "blue/green デプロイ" section**

Below the "サーバー作成" section, insert a new section:

```markdown
## アプリデプロイ（conoha-proxy 経由 blue/green）

v0.2.0 から `conoha app deploy` は [conoha-proxy](https://github.com/crowdy/conoha-proxy) 経由の blue/green デプロイに統一されました。初回セットアップの流れ:

1. レポジトリルートに `conoha.yml` を作成:

   ```yaml
   name: myapp
   hosts:
     - app.example.com
   web:
     service: web
     port: 8080
   ```

2. プロキシコンテナを VPS にブート:

   ```bash
   conoha proxy boot my-server --acme-email ops@example.com
   ```

3. DNS の A レコードを VPS に向ける（Let's Encrypt HTTP-01 検証に必要）。

4. アプリをプロキシに登録:

   ```bash
   conoha app init my-server
   ```

5. デプロイ:

   ```bash
   conoha app deploy my-server
   ```

ロールバック（drain 窓内のみ）:

```bash
conoha app rollback my-server
```
```

- [ ] **Step 3: Commit**

```bash
git add README.md
git commit -m "docs: document conoha-proxy blue/green deploy flow"
```

---

### Task 21: Update README-en.md and README-ko.md

**Files:**
- Modify: `README-en.md`
- Modify: `README-ko.md`

- [ ] **Step 1: Mirror the Japanese README changes**

Apply equivalent changes to both translations:
- Add `conoha proxy` to the command table
- Append `rollback` to the `conoha app` row
- Add the "App deploy (blue/green via conoha-proxy)" / "앱 배포" section

Use the same `conoha.yml` example and step sequence. Translate the prose only.

- [ ] **Step 2: Commit**

```bash
git add README-en.md README-ko.md
git commit -m "docs: translate proxy-based deploy section (en/ko)"
```

---

### Task 22: Update `recipes/single-server-app.md`

**Files:**
- Modify: `recipes/single-server-app.md`

- [ ] **Step 1: Read the recipe**

Open `recipes/single-server-app.md` and identify sections that reference `conoha app init`, `conoha app deploy`, and the git push workflow.

- [ ] **Step 2: Replace git-push flow with conoha.yml + proxy flow**

Every occurrence of the git bare repo / `git push conoha main` must be removed. Replace the "Deploying your app" section with the five-step sequence from README Task 20 (conoha.yml → proxy boot → DNS → app init → app deploy).

- [ ] **Step 3: Add a "Migrating from git-push deploy" subsection**

Add near the bottom:

```markdown
### Migrating from v0.1.x git-push deploy

1. Keep your server running (no data loss).
2. Add `conoha.yml` to your repo (see above).
3. `conoha proxy boot <server> --acme-email you@example.com`
4. `conoha app init <server>` — this upserts your service into the proxy.
5. `conoha app deploy <server>` — first blue/green deploy.
6. (Optional) Delete the legacy bare repo on the server:
   ```bash
   ssh <user>@<server> rm -rf /opt/conoha/<appname>.git
   ```
```

- [ ] **Step 4: Commit**

```bash
git add recipes/single-server-app.md
git commit -m "docs(recipe): update single-server-app for proxy deploy"
```

---

### Task 23: Update Claude Code skill (if applicable)

**Files:**
- Modify: `SKILL.md` (if it describes `app init`/`app deploy`)

- [ ] **Step 1: Read SKILL.md**

Open `SKILL.md` and grep for `app deploy`, `app init`, `git push`. Any prose that still describes the old single-slot flow needs updating.

- [ ] **Step 2: Update the "Deploying apps" section**

Wherever the skill describes the deploy flow, replace with:

- "First-time: ensure `conoha.yml` exists, then `conoha proxy boot --acme-email …`, then `conoha app init`, then `conoha app deploy`."
- "Subsequent deploys: just `conoha app deploy`."
- "Rollback within 30s drain window: `conoha app rollback`."

- [ ] **Step 3: Commit**

```bash
git add SKILL.md
git commit -m "docs(skill): update deploy description for proxy blue/green"
```

---

## Phase 9: Final verification

### Task 24: Full build, test, lint

**Files:** none (verification only)

- [ ] **Step 1: Run full test suite**

Run: `go test ./...`
Expected: PASS. Failures likely indicate an overlooked old test or missing edit in earlier tasks.

- [ ] **Step 2: Run go vet**

Run: `go vet ./...`
Expected: clean.

- [ ] **Step 3: Run the binary help**

Run: `go run . --help`
Expected: `proxy` appears in the command list.

Run: `go run . proxy --help`
Expected: `boot`, `reboot`, `start`, `stop`, `restart`, `remove`, `logs`, `details`, `services` appear.

Run: `go run . app --help`
Expected: `rollback` appears next to `init`, `deploy`.

- [ ] **Step 4: Build release binary**

Run: `make build`
Expected: binary produced at `bin/conoha`.

- [ ] **Step 5: Smoke-run on binary**

Run: `./bin/conoha proxy boot --help`
Expected: help text with `--acme-email`, `--image`, etc.

- [ ] **Step 6: No commit needed — all changes already committed**

```bash
git log --oneline -25
```

Review commit list and ensure the series of commits matches the phase structure.

---

## Spec-to-Task Coverage Map

| Spec section | Task(s) |
|---|---|
| §1 Background / Goals | Plan header + README sections (Tasks 20–22) |
| §2 Scope / Breaking changes | Tasks 12, 16, 20–23 |
| §3.1 `conoha proxy` group | Tasks 8–11 |
| §3.2 `conoha app` modifications | Tasks 12, 16, 17, 18, 19 |
| §4 `conoha.yml` schema | Tasks 1, 2 |
| §5 Deploy flow | Tasks 13–16 |
| §6 Admin API access | Tasks 3–6 |
| §7 Internal layer structure | Tasks 1–6, 8–11, 12–17 |
| §8 Errors / observability | Tasks 4, 16, 17 |
| §9 Test strategy | Tasks 1, 2, 4, 5, 7, 13, 14, 15 |
| §10 Invariants | Tasks 5 (error handling), 16 (tearDown on 424) |
| §11 Migration | Tasks 12 (warn on legacy repo), 20, 22, 23 |
| §12 Implementation order | This plan's phase order |

## Notes

- **Frequent commits**: each task produces a single focused commit. Do not squash across tasks during implementation.
- **TDD**: unit-testable tasks (1, 2, 4, 5, 7, 13, 14, 15) follow red → green → commit. Wiring tasks (8–12, 17–23) rely on build + smoke runs because they depend on live SSH/Docker.
- **Integration testing** is out of scope per the spec. A real-VPS dry run should be done before tagging v0.2.0 and noted in the release PR description, not in code.
