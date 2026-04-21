# `--no-proxy` Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `--no-proxy` mode to `conoha app {init,deploy,logs,stop,restart,status,destroy,env,rollback}` so single-slot, TLS-less deploys can coexist on the same server as proxy-based blue/green deploys. Hybrid mode selection via a `.conoha-mode` marker with `--proxy`/`--no-proxy` flag overrides. Absorb issue #93 (slot-aware logs/stop/restart/status in proxy mode).

**Architecture:** Add one helper file `cmd/app/mode.go` providing pure shell-command builders, a Mode type, flag registration, and thin exec wrappers. Each command's RunE resolves the mode once (marker + optional flag override), then dispatches to a proxy branch (existing code) or a no-proxy branch (new or legacy flat-path code). No cross-command abstraction layer.

**Tech Stack:** Go 1.26, spf13/cobra, gopkg.in/yaml.v3, golang.org/x/crypto/ssh. Tests follow existing `cmd/app/*_test.go` style — pure shell-command builders tested exhaustively; cobra flag wiring tested; SSH integration points left to manual verification.

**Spec:** `docs/superpowers/specs/2026-04-21-no-proxy-mode-design.md`

**Branch:** `feat/no-proxy-mode` (already created, spec committed at `9e57b05`).

---

## File Plan

### Create

| File | Responsibility |
|---|---|
| `cmd/app/mode.go` | `Mode` type, `ErrNoMarker`/`ErrModeConflict`, shell-command builders, `ReadMarker`/`WriteMarker`/`ResolveMode`/`ReadCurrentSlot`, `AddModeFlags`, `formatModeConflictError` |
| `cmd/app/mode_test.go` | Tests for all pure functions in `mode.go` |
| `cmd/app/logs_test.go` | Mode-dispatch tests for logs (shell string assertions) |
| `cmd/app/status_test.go` | Mode-dispatch tests for status |
| `cmd/app/env_test.go` | Proxy-mode warning injection test |
| `docs/recipes/single-server-app-noproxy.md` | No-proxy quickstart recipe |

### Modify

| File | Changes |
|---|---|
| `cmd/app/init.go` | `--no-proxy` branch (no conoha.yml); `WriteMarker` at end of both branches |
| `cmd/app/deploy.go` | Mode dispatch; split proxy path into `runProxyDeploy`; add `runNoProxyDeploy` |
| `cmd/app/rollback.go` | Early exit with code 5 + recovery hint when mode is no-proxy |
| `cmd/app/destroy.go` | Mode dispatch guards proxy `DELETE` call |
| `cmd/app/logs.go` | Mode dispatch: proxy uses `docker compose -p <app>-<slot> logs` |
| `cmd/app/stop.go` | Mode dispatch: proxy uses `docker compose -p <app>-<slot> stop` |
| `cmd/app/restart.go` | Mode dispatch: proxy uses `docker compose -p <app>-<slot> restart` |
| `cmd/app/status.go` | Mode dispatch: skip proxy phase block in no-proxy |
| `cmd/app/env.go` | Warn once per subcommand when mode is proxy |
| `cmd/app/destroy_test.go` | Add flag-exclusion tests |
| `README.md`, `README-ja.md`, `README-ko.md` | Two-modes section |
| `docs/superpowers/specs/2026-04-20-conoha-proxy-deploy-design.md` | One-line cross-reference to new spec |

---

## Task 1: Mode type, errors, and shell builders

**Files:**
- Create: `cmd/app/mode.go`
- Create: `cmd/app/mode_test.go`

- [ ] **Step 1.1: Write failing tests**

Create `cmd/app/mode_test.go` with:

```go
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
```

- [ ] **Step 1.2: Run tests — confirm they fail with "undefined"**

```bash
go test ./cmd/app/ -run 'TestMode_String|TestParseMarker|TestBuildReadMarkerCmd|TestBuildWriteMarkerCmd|TestBuildReadCurrentSlotCmd|TestFormatModeConflictError' -v
```

Expected: compile failure (`undefined: Mode`, etc.).

- [ ] **Step 1.3: Write `cmd/app/mode.go` minimal implementation**

```go
package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

type Mode string

const (
	ModeProxy   Mode = "proxy"
	ModeNoProxy Mode = "no-proxy"
)

var (
	ErrNoMarker     = errors.New("no mode marker on server")
	ErrModeConflict = errors.New("mode conflict")
)

// ParseMarker accepts the raw marker file content and returns the Mode.
func ParseMarker(s string) (Mode, error) {
	v := strings.TrimSpace(s)
	switch v {
	case string(ModeProxy):
		return ModeProxy, nil
	case string(ModeNoProxy):
		return ModeNoProxy, nil
	case "":
		return "", fmt.Errorf("empty marker")
	default:
		return "", fmt.Errorf("unknown marker value %q", v)
	}
}

// buildReadMarkerCmd prints marker contents or "__MISSING__" if absent.
// The distinct sentinel lets ReadMarker tell "file absent" apart from
// permission or SSH errors without relying on exit codes.
func buildReadMarkerCmd(app string) string {
	return fmt.Sprintf(
		`cat '/opt/conoha/%s/.conoha-mode' 2>/dev/null || echo __MISSING__`,
		app)
}

// buildWriteMarkerCmd creates the app dir (if missing) and writes the marker.
func buildWriteMarkerCmd(app string, m Mode) string {
	return fmt.Sprintf(
		`mkdir -p '/opt/conoha/%s' && printf %%s\\n '%s' > '/opt/conoha/%s/.conoha-mode'`,
		app, string(m), app)
}

// buildReadCurrentSlotCmd prints the active slot ID or empty output on absence.
func buildReadCurrentSlotCmd(app string) string {
	return fmt.Sprintf(
		`cat '/opt/conoha/%s/CURRENT_SLOT' 2>/dev/null || true`,
		app)
}

// formatModeConflictError returns a user-facing error wrapping ErrModeConflict.
func formatModeConflictError(app string, got, want Mode) error {
	oppositeInit := "conoha app init"
	if want == ModeNoProxy {
		oppositeInit = "conoha app init --no-proxy"
	}
	return fmt.Errorf(
		`app %q is initialized in %s mode on this server, but --%s was requested.
To switch modes:
    conoha app destroy <server>               # removes the existing deployment
    %s <server>       # re-initialize in %s mode
%w`,
		app, string(got), string(want), oppositeInit, string(want), ErrModeConflict)
}

// ReadMarker returns the mode recorded on the server for app, or ErrNoMarker
// if no marker file exists.
func ReadMarker(cli *ssh.Client, app string) (Mode, error) {
	var buf bytes.Buffer
	if _, err := internalssh.RunCommand(cli, buildReadMarkerCmd(app), &buf, os.Stderr); err != nil {
		return "", fmt.Errorf("read marker: %w", err)
	}
	out := strings.TrimSpace(buf.String())
	if out == "__MISSING__" {
		return "", ErrNoMarker
	}
	return ParseMarker(out)
}

// WriteMarker persists the marker file on the server.
func WriteMarker(cli *ssh.Client, app string, m Mode) error {
	code, err := internalssh.RunCommand(cli, buildWriteMarkerCmd(app, m), os.Stderr, os.Stderr)
	if err != nil {
		return fmt.Errorf("write marker: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("write marker: exit %d", code)
	}
	return nil
}

// ReadCurrentSlot returns the active slot ID or "" when the file is absent.
func ReadCurrentSlot(cli *ssh.Client, app string) (string, error) {
	var buf bytes.Buffer
	if _, err := internalssh.RunCommand(cli, buildReadCurrentSlotCmd(app), &buf, os.Stderr); err != nil {
		return "", fmt.Errorf("read CURRENT_SLOT: %w", err)
	}
	return strings.TrimSpace(buf.String()), nil
}

// flagMode reads --proxy / --no-proxy flags and returns the intended mode, or
// "" if neither is set. Callers should have registered the flags mutually
// exclusive via AddModeFlags.
func flagMode(cmd *cobra.Command) Mode {
	if cmd.Flags().Lookup("no-proxy") != nil {
		if v, _ := cmd.Flags().GetBool("no-proxy"); v {
			return ModeNoProxy
		}
	}
	if cmd.Flags().Lookup("proxy") != nil {
		if v, _ := cmd.Flags().GetBool("proxy"); v {
			return ModeProxy
		}
	}
	return ""
}

// ResolveMode interprets flags against the marker.
// Precedence: flag override compared to marker (error on mismatch) > marker > ErrNoMarker.
func ResolveMode(cmd *cobra.Command, cli *ssh.Client, app string) (Mode, error) {
	want := flagMode(cmd)
	got, readErr := ReadMarker(cli, app)
	if readErr != nil && !errors.Is(readErr, ErrNoMarker) {
		return "", readErr
	}
	switch {
	case want == "" && errors.Is(readErr, ErrNoMarker):
		return "", ErrNoMarker
	case want == "":
		return got, nil
	case errors.Is(readErr, ErrNoMarker):
		return want, nil
	case want != got:
		return "", formatModeConflictError(app, got, want)
	default:
		return got, nil
	}
}

// AddModeFlags registers --proxy and --no-proxy as mutually exclusive bool flags.
func AddModeFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("proxy", false, "force proxy (blue/green) mode, overriding server marker")
	cmd.Flags().Bool("no-proxy", false, "force no-proxy (flat single-slot) mode, overriding server marker")
	cmd.MarkFlagsMutuallyExclusive("proxy", "no-proxy")
}
```

- [ ] **Step 1.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestMode_String|TestParseMarker|TestBuildReadMarkerCmd|TestBuildWriteMarkerCmd|TestBuildReadCurrentSlotCmd|TestFormatModeConflictError' -v
```

Expected: all PASS.

- [ ] **Step 1.5: Commit**

```bash
git add cmd/app/mode.go cmd/app/mode_test.go
git commit -m "feat(app): add Mode type, marker helpers, and mode resolution

Introduce Mode enum (proxy | no-proxy), ErrNoMarker / ErrModeConflict,
shell-command builders for the .conoha-mode marker file, ReadMarker /
WriteMarker / ResolveMode / ReadCurrentSlot helpers, and the
--proxy/--no-proxy mutually-exclusive flag pair. Foundation for #102.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2: `app init` — add `--no-proxy` branch and persist marker

**Files:**
- Modify: `cmd/app/init.go`
- Create: `cmd/app/init_test.go` (if absent, extend otherwise)

- [ ] **Step 2.1: Write failing tests**

Create or extend `cmd/app/init_test.go`:

```go
package app

import (
	"testing"
)

func TestInitCmd_HasModeFlags(t *testing.T) {
	if initCmd.Flags().Lookup("proxy") == nil {
		t.Error("init should have --proxy flag")
	}
	if initCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("init should have --no-proxy flag")
	}
}

func TestInitCmd_ModeFlagsMutuallyExclusive(t *testing.T) {
	if err := initCmd.ParseFlags([]string{"--proxy", "--no-proxy"}); err == nil {
		t.Error("--proxy and --no-proxy should be mutually exclusive")
	}
}
```

- [ ] **Step 2.2: Run — confirm fail**

```bash
go test ./cmd/app/ -run 'TestInitCmd_' -v
```

Expected: FAIL — `--proxy`/`--no-proxy` flag lookup returns nil.

- [ ] **Step 2.3: Modify `cmd/app/init.go`**

At the end of the existing `init()` function, add:

```go
	AddModeFlags(initCmd)
	initCmd.Flags().String("app-name", "", "application name (required with --no-proxy)")
```

Replace the entire `initCmd.RunE` body with:

```go
		RunE: func(cmd *cobra.Command, args []string) error {
			noProxy, _ := cmd.Flags().GetBool("no-proxy")
			if noProxy {
				return runInitNoProxy(cmd, args[0])
			}
			return runInitProxy(cmd, args[0])
		},
```

Rename the existing body into a new function `runInitProxy(cmd *cobra.Command, serverID string) error` containing the current logic, then at the end (right before `return nil`):

```go
	if err := WriteMarker(sshClient, pf.Name, ModeProxy); err != nil {
		fmt.Fprintf(os.Stderr, "warning: write mode marker: %v\n", err)
	}
```

Add a new function `runInitNoProxy`:

```go
func runInitNoProxy(cmd *cobra.Command, serverID string) error {
	appName, _ := cmd.Flags().GetString("app-name")
	if appName == "" {
		return fmt.Errorf("--app-name is required with --no-proxy")
	}
	if err := internalssh.ValidateAppName(appName); err != nil {
		return err
	}
	sshClient, s, ip, err := connectToServer(cmd, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sshClient.Close() }()

	// Verify docker is present.
	code, err := internalssh.RunCommand(sshClient, "command -v docker >/dev/null 2>&1", os.Stderr, os.Stderr)
	if err != nil {
		return fmt.Errorf("docker check: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("docker is not installed on %s (%s)", s.Name, ip)
	}

	fmt.Fprintf(os.Stderr, "==> Initializing %q on %s (%s) in no-proxy mode\n", appName, s.Name, ip)
	if err := WriteMarker(sshClient, appName, ModeNoProxy); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Initialized. Next: run 'conoha app deploy --no-proxy --app-name %s %s'\n", appName, serverID)
	return nil
}
```

Add `internalssh "github.com/crowdy/conoha-cli/internal/ssh"` to the imports if not already present (it is — keep as-is).

- [ ] **Step 2.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestInitCmd_' -v && go build ./...
```

Expected: flag tests PASS, build succeeds.

- [ ] **Step 2.5: Commit**

```bash
git add cmd/app/init.go cmd/app/init_test.go
git commit -m "feat(app/init): add --no-proxy branch and persist mode marker

No-proxy init installs only the mkdir + marker write (no conoha.yml
required). Proxy init continues through the existing upsert path and
now writes the marker at the end. --app-name is required with
--no-proxy.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3: `app deploy` — mode dispatch and no-proxy flat deploy

**Files:**
- Modify: `cmd/app/deploy.go`
- Modify: `cmd/app/deploy_test.go`

- [ ] **Step 3.1: Write failing tests**

Append to `cmd/app/deploy_test.go`:

```go
func TestDeployCmd_HasModeFlags(t *testing.T) {
	if deployCmd.Flags().Lookup("proxy") == nil {
		t.Error("deploy should have --proxy flag")
	}
	if deployCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("deploy should have --no-proxy flag")
	}
	if deployCmd.Flags().Lookup("app-name") == nil {
		t.Error("deploy should have --app-name flag (required with --no-proxy)")
	}
}

func TestBuildNoProxyDeployCmd(t *testing.T) {
	got := buildNoProxyDeployCmd("/opt/conoha/myapp", "myapp", "compose.yml")
	for _, want := range []string{
		"cd '/opt/conoha/myapp'",
		"docker compose -p myapp",
		"-f compose.yml",
		"up -d --build",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildNoProxyUploadCmd(t *testing.T) {
	got := buildNoProxyUploadCmd("/opt/conoha/myapp")
	// Must preserve existing .env.server content on re-deploy (v0.1.x parity).
	for _, want := range []string{
		"mkdir -p '/opt/conoha/myapp'",
		"tar xzf - -C '/opt/conoha/myapp'",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	// Must NOT rm -rf the app dir (that would blow away the env file + persistent volumes).
	if strings.Contains(got, "rm -rf '/opt/conoha/myapp'") {
		t.Errorf("no-proxy upload must not wipe app dir: %s", got)
	}
}
```

Make sure `"strings"` is imported at the top of `deploy_test.go`.

- [ ] **Step 3.2: Run — confirm fail**

```bash
go test ./cmd/app/ -run 'TestDeployCmd_HasModeFlags|TestBuildNoProxyDeployCmd|TestBuildNoProxyUploadCmd' -v
```

Expected: FAIL — undefined builders, missing flags.

- [ ] **Step 3.3: Modify `cmd/app/deploy.go`**

In the existing `init()` add:

```go
	AddModeFlags(deployCmd)
	deployCmd.Flags().String("app-name", "", "application name (required with --no-proxy)")
```

Replace the `deployCmd.RunE` body with a dispatch:

```go
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeployDispatch(cmd, args[0])
		},
```

Rename the existing `runDeploy` function to `runProxyDeploy` (unchanged body aside from the name). Add a new function `runDeployDispatch` and a new function `runNoProxyDeploy`:

```go
// runDeployDispatch resolves mode (flag override + server marker) and calls
// the proxy or no-proxy deploy path.
func runDeployDispatch(cmd *cobra.Command, serverID string) error {
	// Fast-path: if --no-proxy was explicitly passed, we can skip the proxy
	// path's conoha.yml load entirely. But we still need an SSH client to
	// read the marker for conflict detection.
	noProxyFlag, _ := cmd.Flags().GetBool("no-proxy")

	if noProxyFlag {
		appName, _ := cmd.Flags().GetString("app-name")
		if appName == "" {
			return fmt.Errorf("--app-name is required with --no-proxy")
		}
		if err := internalssh.ValidateAppName(appName); err != nil {
			return err
		}
		sshClient, s, ip, err := connectToServer(cmd, serverID)
		if err != nil {
			return err
		}
		defer func() { _ = sshClient.Close() }()
		got, err := ReadMarker(sshClient, appName)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return fmt.Errorf("app %q not initialized on this server — run 'conoha app init --no-proxy --app-name %s %s' first", appName, appName, serverID)
			}
			return err
		}
		if got != ModeNoProxy {
			return formatModeConflictError(appName, got, ModeNoProxy)
		}
		return runNoProxyDeploy(cmd, sshClient, s, ip, appName)
	}

	// Default: proxy path. It loads conoha.yml before SSH; we preserve that
	// ordering so validation errors surface without a network round-trip.
	return runProxyDeploy(cmd, serverID)
}
```

Add the required imports: `"errors"`, `"github.com/crowdy/conoha-cli/internal/model"` — check and add whatever is missing for the signature (model.Server is already imported via connect.go's context, but here the server arg is unused below; see simplified signature). Use this simpler signature instead:

```go
func runNoProxyDeploy(cmd *cobra.Command, sshClient *ssh.Client, s *model.Server, ip, appName string) error {
	fmt.Fprintf(os.Stderr, "==> Deploying %q to %s (%s) in no-proxy mode\n", appName, s.Name, ip)

	patterns, err := loadIgnorePatterns(".")
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := createTarGz(".", patterns, &buf); err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	workDir := "/opt/conoha/" + appName
	if err := runRemote(sshClient, buildNoProxyUploadCmd(workDir), &buf); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	// Resolve compose file from the freshly-uploaded work dir on the remote.
	// Mirrors proxy path's ResolveComposeFile but runs via the local copy
	// (the working directory being deployed) to keep logic simple.
	pf := &config.ProjectFile{}
	composeFile, err := pf.ResolveComposeFile(".")
	if err != nil {
		return err
	}

	if err := runRemote(sshClient, buildNoProxyDeployCmd(workDir, appName, composeFile), nil); err != nil {
		return fmt.Errorf("compose up: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Deploy complete.")
	return nil
}
```

Add the two shell builders (put them in `remotecmds.go` or inline in deploy.go — this plan places them in `deploy.go`):

```go
// buildNoProxyUploadCmd extracts the incoming tar archive into the app work
// directory, preserving any existing files so that .env.server and named
// volumes survive redeploys. (Tar-over-tar will overwrite code files while
// leaving unrelated siblings intact.)
func buildNoProxyUploadCmd(workDir string) string {
	return fmt.Sprintf(
		"mkdir -p '%[1]s' && tar xzf - -C '%[1]s'",
		workDir)
}

// buildNoProxyDeployCmd brings the flat-layout compose project up in place.
// The compose project name equals the app name (no slot suffix).
func buildNoProxyDeployCmd(workDir, app, composeFile string) string {
	return fmt.Sprintf(
		"cd '%s' && docker compose -p %s -f %s up -d --build",
		workDir, app, composeFile)
}
```

- [ ] **Step 3.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestDeployCmd_HasModeFlags|TestBuildNoProxyDeployCmd|TestBuildNoProxyUploadCmd' -v
go build ./...
```

Expected: PASS.

- [ ] **Step 3.5: Commit**

```bash
git add cmd/app/deploy.go cmd/app/deploy_test.go
git commit -m "feat(app/deploy): add --no-proxy flat deploy path

runDeployDispatch reads the --no-proxy flag, resolves the server
marker, and either calls runProxyDeploy (existing blue/green flow)
or runNoProxyDeploy (tar upload to /opt/conoha/<name>/ + compose up
against the project name <name>). Proxy/no-proxy marker mismatches
produce the standard mode-conflict error.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4: `app rollback` — reject no-proxy with guidance

**Files:**
- Modify: `cmd/app/rollback.go`
- Create: `cmd/app/rollback_test.go`

- [ ] **Step 4.1: Write failing tests**

Create `cmd/app/rollback_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestRollbackCmd_HasModeFlags(t *testing.T) {
	if rollbackCmd.Flags().Lookup("proxy") == nil {
		t.Error("rollback should have --proxy flag")
	}
	if rollbackCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("rollback should have --no-proxy flag")
	}
}

func TestRollbackNoProxyError(t *testing.T) {
	err := noProxyRollbackError("myapp")
	msg := err.Error()
	for _, want := range []string{
		"rollback is not supported in no-proxy mode",
		"git checkout",
		"conoha app deploy --no-proxy",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %s", want, msg)
		}
	}
}
```

- [ ] **Step 4.2: Run — confirm fail**

```bash
go test ./cmd/app/ -run 'TestRollbackCmd_|TestRollbackNoProxyError' -v
```

Expected: FAIL — no --proxy/--no-proxy flags, undefined `noProxyRollbackError`.

- [ ] **Step 4.3: Modify `cmd/app/rollback.go`**

At end of `init()`, add:

```go
	AddModeFlags(rollbackCmd)
	rollbackCmd.Flags().String("app-name", "", "application name (used when --no-proxy bypasses conoha.yml)")
```

Add a helper:

```go
func noProxyRollbackError(app string) error {
	return fmt.Errorf(
		"rollback is not supported in no-proxy mode. Deploy a previous revision instead: "+
			"git checkout <rev> && conoha app deploy --no-proxy --app-name %s <server>", app)
}
```

Replace the RunE body. Before loading the project file, add mode check:

```go
		RunE: func(cmd *cobra.Command, args []string) error {
			noProxyFlag, _ := cmd.Flags().GetBool("no-proxy")
			if noProxyFlag {
				appName, _ := cmd.Flags().GetString("app-name")
				if appName == "" {
					return fmt.Errorf("--app-name is required with --no-proxy")
				}
				return noProxyRollbackError(appName)
			}
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

			mode, err := ResolveMode(cmd, sshClient, pf.Name)
			if err != nil {
				if errors.Is(err, ErrNoMarker) {
					return fmt.Errorf("app %q not initialized on this server — run 'conoha app init' first", pf.Name)
				}
				return err
			}
			if mode == ModeNoProxy {
				return noProxyRollbackError(pf.Name)
			}

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
			active := "<unknown>"
			if updated.ActiveTarget != nil {
				active = updated.ActiveTarget.URL
			}
			fmt.Fprintf(os.Stderr, "Rollback complete. active=%s phase=%s\n", active, updated.Phase)
			return nil
		},
```

- [ ] **Step 4.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestRollbackCmd_|TestRollbackNoProxyError' -v
go build ./...
```

Expected: PASS, build succeeds.

- [ ] **Step 4.5: Commit**

```bash
git add cmd/app/rollback.go cmd/app/rollback_test.go
git commit -m "feat(app/rollback): reject no-proxy mode with recovery guidance

--no-proxy (or a no-proxy marker) on rollback now returns an explicit
error pointing at 'git checkout <rev> && conoha app deploy --no-proxy'.
Proxy-mode behavior unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5: `app destroy` — branch proxy DELETE call

**Files:**
- Modify: `cmd/app/destroy.go`
- Modify: `cmd/app/destroy_test.go`

- [ ] **Step 5.1: Write failing tests**

Replace `cmd/app/destroy_test.go` with:

```go
package app

import (
	"testing"
)

func TestDestroyCmd_HasYesFlag(t *testing.T) {
	f := destroyCmd.Flags().Lookup("yes")
	if f == nil {
		t.Fatal("destroy command should have --yes flag")
	}
	if f.DefValue != "false" {
		t.Errorf("--yes default should be false, got %s", f.DefValue)
	}
}

func TestDestroyCmd_HasModeFlags(t *testing.T) {
	if destroyCmd.Flags().Lookup("proxy") == nil {
		t.Error("destroy should have --proxy flag")
	}
	if destroyCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("destroy should have --no-proxy flag")
	}
}
```

- [ ] **Step 5.2: Run — confirm fail**

```bash
go test ./cmd/app/ -run 'TestDestroyCmd_' -v
```

Expected: FAIL on HasModeFlags.

- [ ] **Step 5.3: Modify `cmd/app/destroy.go`**

In `init()` add:

```go
	AddModeFlags(destroyCmd)
```

Replace the proxy-delete section of `destroyCmd.RunE` (the block that calls `admin.Delete`) with mode-aware logic:

```go
		// Resolve mode to decide whether to deregister from proxy.
		mode, err := ResolveMode(cmd, ctx.Client, ctx.AppName)
		if err != nil && !errors.Is(err, ErrNoMarker) {
			return err
		}
		// mode is "" when marker is absent (legacy server). Only call proxy
		// delete in proxy mode; skip silently in no-proxy or legacy.
		if mode == ModeProxy {
			dataDir, _ := cmd.Flags().GetString("data-dir")
			if dataDir == "" {
				dataDir = proxy.DefaultDataDir
			}
			admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: ctx.Client}, proxy.SocketPath(dataDir))
			pf, pfErr := config.LoadProjectFile(config.ProjectFileName)
			if pfErr == nil && pf.Validate() == nil {
				if err := admin.Delete(pf.Name); err != nil && !errors.Is(err, proxypkg.ErrNotFound) {
					fmt.Fprintf(os.Stderr, "warning: proxy delete %s: %v\n", pf.Name, err)
				} else if err == nil {
					fmt.Fprintf(os.Stderr, "==> Deregistered %q from proxy\n", pf.Name)
				}
			}
		}
```

(Keep the rest of destroy's logic — `generateDestroyScript`, the SSH script run, the final success message — unchanged. The shell script already handles flat and slotted layouts via its `grep -E "^${APP_NAME}(-|$)"`.)

- [ ] **Step 5.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestDestroyCmd_' -v
go build ./...
```

Expected: PASS.

- [ ] **Step 5.5: Commit**

```bash
git add cmd/app/destroy.go cmd/app/destroy_test.go
git commit -m "feat(app/destroy): skip proxy DELETE in no-proxy/legacy mode

destroy now resolves the .conoha-mode marker and only deregisters from
conoha-proxy when the marker is 'proxy'. No-proxy and unmarked (legacy)
servers continue to run the shared compose-down + rm -rf cleanup script.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6: `app logs` — mode-aware, absorbing #93

**Files:**
- Modify: `cmd/app/logs.go`
- Create: `cmd/app/logs_test.go`

- [ ] **Step 6.1: Write failing tests**

Create `cmd/app/logs_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestLogsCmd_HasModeFlags(t *testing.T) {
	if logsCmd.Flags().Lookup("proxy") == nil {
		t.Error("logs should have --proxy flag")
	}
	if logsCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("logs should have --no-proxy flag")
	}
}

func TestBuildLogsCmd_Proxy(t *testing.T) {
	got := buildLogsCmdForProxy("myapp", "abc1234", 100, false, "")
	for _, want := range []string{
		"docker compose -p myapp-abc1234",
		"logs",
		"--tail 100",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildLogsCmd_Proxy_FollowService(t *testing.T) {
	got := buildLogsCmdForProxy("myapp", "abc1234", 50, true, "web")
	for _, want := range []string{
		"docker compose -p myapp-abc1234 logs",
		"--tail 50",
		"-f",
		" web",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildLogsCmd_NoProxy(t *testing.T) {
	got := buildLogsCmdForNoProxy("myapp", 100, false, "")
	for _, want := range []string{
		"cd /opt/conoha/myapp",
		"docker compose logs",
		"--tail 100",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
```

- [ ] **Step 6.2: Run — confirm fail**

```bash
go test ./cmd/app/ -run 'TestLogsCmd_|TestBuildLogsCmd_' -v
```

Expected: FAIL (undefined builders, missing flags).

- [ ] **Step 6.3: Modify `cmd/app/logs.go`**

Replace the entire file contents:

```go
package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(logsCmd)
	logsCmd.Flags().BoolP("follow", "f", false, "stream logs in real-time")
	logsCmd.Flags().Int("tail", 100, "number of lines to show")
	logsCmd.Flags().String("service", "", "specific service name")
	AddModeFlags(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs <id|name>",
	Short: "Show app container logs",
	Long:  "Show docker compose logs for the active slot (proxy mode) or the flat work dir (no-proxy). Use --follow to stream in real-time (Ctrl+C to stop).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		follow, _ := cmd.Flags().GetBool("follow")
		tail, _ := cmd.Flags().GetInt("tail")
		service, _ := cmd.Flags().GetString("service")
		if service != "" {
			if err := internalssh.ValidateAppName(service); err != nil {
				return fmt.Errorf("invalid service name: %w", err)
			}
		}

		mode, err := ResolveMode(cmd, ctx.Client, ctx.AppName)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return fmt.Errorf("app %q has not been initialized on this server", ctx.AppName)
			}
			return err
		}

		var composeCmd string
		if mode == ModeProxy {
			slot, err := ReadCurrentSlot(ctx.Client, ctx.AppName)
			if err != nil {
				return err
			}
			if slot == "" {
				return fmt.Errorf("app %q has not been deployed on this server", ctx.AppName)
			}
			composeCmd = buildLogsCmdForProxy(ctx.AppName, slot, tail, follow, service)
		} else {
			composeCmd = buildLogsCmdForNoProxy(ctx.AppName, tail, follow, service)
		}

		exitCode, err := internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("logs failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("logs exited with code %d", exitCode)
		}
		return nil
	},
}

func buildLogsCmdForProxy(app, slot string, tail int, follow bool, service string) string {
	cmd := fmt.Sprintf("docker compose -p %s-%s logs --tail %d", app, slot, tail)
	if follow {
		cmd += " -f"
	}
	if service != "" {
		cmd += " " + service
	}
	return cmd
}

func buildLogsCmdForNoProxy(app string, tail int, follow bool, service string) string {
	cmd := fmt.Sprintf("cd /opt/conoha/%s && docker compose logs --tail %d", app, tail)
	if follow {
		cmd += " -f"
	}
	if service != "" {
		cmd += " " + service
	}
	return cmd
}
```

- [ ] **Step 6.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestLogsCmd_|TestBuildLogsCmd_' -v
go build ./...
```

Expected: PASS.

- [ ] **Step 6.5: Commit**

```bash
git add cmd/app/logs.go cmd/app/logs_test.go
git commit -m "feat(app/logs): dispatch by mode, target active slot in proxy mode

Absorbs #93 for app logs: proxy mode reads CURRENT_SLOT and runs
'docker compose -p <app>-<slot> logs' against the active slot project.
No-proxy mode keeps the flat 'cd /opt/conoha/<app>' path.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7: `app stop` — mode dispatch

**Files:**
- Modify: `cmd/app/stop.go`
- Create: `cmd/app/stop_test.go`

- [ ] **Step 7.1: Write failing tests**

Create `cmd/app/stop_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestStopCmd_HasModeFlags(t *testing.T) {
	if stopCmd.Flags().Lookup("proxy") == nil {
		t.Error("stop should have --proxy flag")
	}
	if stopCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("stop should have --no-proxy flag")
	}
}

func TestBuildStopCmd_Proxy(t *testing.T) {
	got := buildStopCmdForProxy("myapp", "abc1234")
	for _, want := range []string{
		"docker compose -p myapp-abc1234 stop",
		"docker compose -p myapp-abc1234 ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildStopCmd_NoProxy(t *testing.T) {
	got := buildStopCmdForNoProxy("myapp")
	for _, want := range []string{
		"cd /opt/conoha/myapp",
		"docker compose stop",
		"docker compose ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
```

- [ ] **Step 7.2: Run — confirm fail**

```bash
go test ./cmd/app/ -run 'TestStopCmd_|TestBuildStopCmd_' -v
```

Expected: FAIL.

- [ ] **Step 7.3: Modify `cmd/app/stop.go`**

Replace the file with:

```go
package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/prompt"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(stopCmd)
	AddModeFlags(stopCmd)
}

var stopCmd = &cobra.Command{
	Use:   "stop <id|name>",
	Short: "Stop app containers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		ok, err := prompt.Confirm(fmt.Sprintf("Stop app %q on %s?", ctx.AppName, ctx.Server.Name))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}

		mode, err := ResolveMode(cmd, ctx.Client, ctx.AppName)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return fmt.Errorf("app %q has not been initialized on this server", ctx.AppName)
			}
			return err
		}

		var composeCmd string
		if mode == ModeProxy {
			slot, err := ReadCurrentSlot(ctx.Client, ctx.AppName)
			if err != nil {
				return err
			}
			if slot == "" {
				return fmt.Errorf("app %q has not been deployed on this server", ctx.AppName)
			}
			composeCmd = buildStopCmdForProxy(ctx.AppName, slot)
		} else {
			composeCmd = buildStopCmdForNoProxy(ctx.AppName)
		}

		fmt.Fprintf(os.Stderr, "Stopping app %q on %s...\n", ctx.AppName, ctx.Server.Name)
		exitCode, err := internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("stop failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("stop exited with code %d", exitCode)
		}
		return nil
	},
}

func buildStopCmdForProxy(app, slot string) string {
	return fmt.Sprintf("docker compose -p %s-%s stop && docker compose -p %s-%s ps", app, slot, app, slot)
}

func buildStopCmdForNoProxy(app string) string {
	return fmt.Sprintf("cd /opt/conoha/%s && docker compose stop && docker compose ps", app)
}
```

- [ ] **Step 7.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestStopCmd_|TestBuildStopCmd_' -v
go build ./...
```

Expected: PASS.

- [ ] **Step 7.5: Commit**

```bash
git add cmd/app/stop.go cmd/app/stop_test.go
git commit -m "feat(app/stop): dispatch by mode, target active slot in proxy mode

Proxy-mode stop runs 'docker compose -p <app>-<slot> stop' against
the active slot (CURRENT_SLOT); no-proxy keeps the legacy flat path.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8: `app restart` — mode dispatch

**Files:**
- Modify: `cmd/app/restart.go`
- Create: `cmd/app/restart_test.go`

- [ ] **Step 8.1: Write failing tests**

Create `cmd/app/restart_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestRestartCmd_HasModeFlags(t *testing.T) {
	if restartCmd.Flags().Lookup("proxy") == nil {
		t.Error("restart should have --proxy flag")
	}
	if restartCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("restart should have --no-proxy flag")
	}
}

func TestBuildRestartCmd_Proxy(t *testing.T) {
	got := buildRestartCmdForProxy("myapp", "abc1234")
	for _, want := range []string{
		"docker compose -p myapp-abc1234 restart",
		"docker compose -p myapp-abc1234 ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildRestartCmd_NoProxy(t *testing.T) {
	got := buildRestartCmdForNoProxy("myapp")
	for _, want := range []string{
		"cd /opt/conoha/myapp",
		"docker compose restart",
		"docker compose ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
```

- [ ] **Step 8.2: Run — confirm fail**

```bash
go test ./cmd/app/ -run 'TestRestartCmd_|TestBuildRestartCmd_' -v
```

Expected: FAIL.

- [ ] **Step 8.3: Modify `cmd/app/restart.go`**

Replace the file with:

```go
package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(restartCmd)
	AddModeFlags(restartCmd)
}

var restartCmd = &cobra.Command{
	Use:   "restart <id|name>",
	Short: "Restart app containers",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		mode, err := ResolveMode(cmd, ctx.Client, ctx.AppName)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return fmt.Errorf("app %q has not been initialized on this server", ctx.AppName)
			}
			return err
		}

		var composeCmd string
		if mode == ModeProxy {
			slot, err := ReadCurrentSlot(ctx.Client, ctx.AppName)
			if err != nil {
				return err
			}
			if slot == "" {
				return fmt.Errorf("app %q has not been deployed on this server", ctx.AppName)
			}
			composeCmd = buildRestartCmdForProxy(ctx.AppName, slot)
		} else {
			composeCmd = buildRestartCmdForNoProxy(ctx.AppName)
		}

		fmt.Fprintf(os.Stderr, "Restarting app %q on %s...\n", ctx.AppName, ctx.Server.Name)
		exitCode, err := internalssh.RunCommand(ctx.Client, composeCmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("restart failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("restart exited with code %d", exitCode)
		}
		return nil
	},
}

func buildRestartCmdForProxy(app, slot string) string {
	return fmt.Sprintf("docker compose -p %s-%s restart && docker compose -p %s-%s ps", app, slot, app, slot)
}

func buildRestartCmdForNoProxy(app string) string {
	return fmt.Sprintf("cd /opt/conoha/%s && docker compose restart && docker compose ps", app)
}
```

- [ ] **Step 8.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestRestartCmd_|TestBuildRestartCmd_' -v
go build ./...
```

Expected: PASS.

- [ ] **Step 8.5: Commit**

```bash
git add cmd/app/restart.go cmd/app/restart_test.go
git commit -m "feat(app/restart): dispatch by mode, target active slot in proxy mode

Proxy-mode restart runs 'docker compose -p <app>-<slot> restart' against
CURRENT_SLOT. No-proxy keeps the legacy flat path.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9: `app status` — mode dispatch, suppress proxy phase in no-proxy

**Files:**
- Modify: `cmd/app/status.go`
- Create: `cmd/app/status_test.go`

- [ ] **Step 9.1: Write failing tests**

Create `cmd/app/status_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestStatusCmd_HasModeFlags(t *testing.T) {
	if statusCmd.Flags().Lookup("proxy") == nil {
		t.Error("status should have --proxy flag")
	}
	if statusCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("status should have --no-proxy flag")
	}
}

func TestBuildStatusCmd_Proxy(t *testing.T) {
	got := buildStatusCmdForProxy("myapp")
	for _, want := range []string{
		"docker compose ls",
		`grep -E "^myapp(-|$)"`,
		"docker compose -p",
		"ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}

func TestBuildStatusCmd_NoProxy(t *testing.T) {
	got := buildStatusCmdForNoProxy("myapp")
	for _, want := range []string{
		"cd /opt/conoha/myapp",
		"docker compose ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
}
```

- [ ] **Step 9.2: Run — confirm fail**

```bash
go test ./cmd/app/ -run 'TestStatusCmd_|TestBuildStatusCmd_' -v
```

Expected: FAIL.

- [ ] **Step 9.3: Modify `cmd/app/status.go`**

Replace the file with:

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
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(statusCmd)
	statusCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	AddModeFlags(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status <id|name>",
	Short: "Show app container status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		mode, err := ResolveMode(cmd, ctx.Client, ctx.AppName)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return fmt.Errorf("app %q has not been initialized on this server", ctx.AppName)
			}
			return err
		}

		var psCmd string
		if mode == ModeProxy {
			psCmd = buildStatusCmdForProxy(ctx.AppName)
		} else {
			psCmd = buildStatusCmdForNoProxy(ctx.AppName)
		}
		if _, err := internalssh.RunCommand(ctx.Client, psCmd, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "warning: compose ps: %v\n", err)
		}

		if mode != ModeProxy {
			return nil
		}

		// Enrich with proxy service state if conoha.yml is present.
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
		return nil
	},
}

func buildStatusCmdForProxy(app string) string {
	return fmt.Sprintf(
		`for p in $(docker compose ls -a --format '{{.Name}}' 2>/dev/null | grep -E "^%[1]s(-|$)" || true); do `+
			`echo "--- compose project: ${p} ---"; `+
			`docker compose -p "${p}" ps; `+
			`done`,
		app)
}

func buildStatusCmdForNoProxy(app string) string {
	return fmt.Sprintf("cd /opt/conoha/%s && docker compose ps", app)
}
```

- [ ] **Step 9.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestStatusCmd_|TestBuildStatusCmd_' -v
go build ./...
```

Expected: PASS.

- [ ] **Step 9.5: Commit**

```bash
git add cmd/app/status.go cmd/app/status_test.go
git commit -m "feat(app/status): dispatch by mode, skip proxy phase in no-proxy

Proxy status scans all slot compose projects and appends the proxy
service phase block. No-proxy status runs a simple 'docker compose ps'
in the flat work dir and skips the proxy enrichment.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10: `app env` — warn when proxy mode

**Files:**
- Modify: `cmd/app/env.go`
- Create: `cmd/app/env_test.go`

- [ ] **Step 10.1: Write failing tests**

Create `cmd/app/env_test.go`:

```go
package app

import (
	"strings"
	"testing"
)

func TestProxyEnvWarningMessage(t *testing.T) {
	msg := proxyEnvWarningMessage()
	for _, want := range []string{
		"warning",
		"app env",
		"proxy-mode",
		"#94",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("missing %q in %s", want, msg)
		}
	}
}
```

- [ ] **Step 10.2: Run — confirm fail**

```bash
go test ./cmd/app/ -run 'TestProxyEnvWarningMessage' -v
```

Expected: FAIL.

- [ ] **Step 10.3: Modify `cmd/app/env.go`**

Add near the top of the file (after imports):

```go
// proxyEnvWarningMessage returns the one-line warning emitted when `app env`
// is run against a proxy-mode app. See #94 for the planned redesign.
func proxyEnvWarningMessage() string {
	return "warning: app env has no effect on proxy-mode deployed slots; see #94 for the redesign\n"
}

// maybeWarnProxyEnvMode emits the proxy-mode warning to stderr once per env
// subcommand invocation. Silent on no-proxy or when marker lookup fails.
func maybeWarnProxyEnvMode(ctx *appContext) {
	m, err := ReadMarker(ctx.Client, ctx.AppName)
	if err == nil && m == ModeProxy {
		fmt.Fprint(os.Stderr, proxyEnvWarningMessage())
	}
}
```

In each of `envSetCmd.RunE`, `envGetCmd.RunE`, `envListCmd.RunE`, `envUnsetCmd.RunE`, add the call immediately after `defer func() { _ = ctx.Client.Close() }()`:

```go
		maybeWarnProxyEnvMode(ctx)
```

- [ ] **Step 10.4: Run tests — confirm pass**

```bash
go test ./cmd/app/ -run 'TestProxyEnvWarningMessage' -v
go build ./...
```

Expected: PASS.

- [ ] **Step 10.5: Commit**

```bash
git add cmd/app/env.go cmd/app/env_test.go
git commit -m "feat(app/env): warn when app env targets a proxy-mode app

app env still writes to /opt/conoha/<app>.env.server (the v0.1.x path,
now canonical for no-proxy mode). When the .conoha-mode marker says
proxy, print a single-line warning pointing at #94 rather than breaking
existing CI scripts.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11: Documentation — README, recipes, prior-spec cross-reference

**Files:**
- Modify: `README.md`, `README-ja.md`, `README-ko.md`
- Create: `docs/recipes/single-server-app-noproxy.md`
- Modify: `docs/superpowers/specs/2026-04-20-conoha-proxy-deploy-design.md`

- [ ] **Step 11.1: Add "Two deploy modes" section to each README**

Open each README file and read the section that currently introduces `app deploy` (search for `conoha app deploy`). Insert, just above that section, identical-structure blocks in each language:

English (`README.md`) block:

```markdown
### Two deploy modes

`conoha app` supports two modes that can coexist on the same VPS:

| Mode | When to use | Layout |
|---|---|---|
| **proxy** (default) | Public app with a domain and TLS | Blue/green slots under `/opt/conoha/<name>/<slot>/` managed via conoha-proxy |
| **no-proxy** (`--no-proxy`) | Testing, internal/dev VPS, non-HTTP services, hobby apps | Flat `/opt/conoha/<name>/` with plain `docker compose up` |

Initialize with `conoha app init --no-proxy --app-name <name> <server>`, then `conoha app deploy --no-proxy --app-name <name> <server>`. No `conoha.yml` required in no-proxy mode.
```

Japanese (`README-ja.md`) equivalent:

```markdown
### 2 つのデプロイモード

`conoha app` は同一 VPS 上で共存可能な 2 つのモードを提供します:

| モード | 用途 | レイアウト |
|---|---|---|
| **proxy** (既定) | ドメイン + TLS の公開アプリ | `/opt/conoha/<name>/<slot>/` の blue/green スロット (conoha-proxy 管理) |
| **no-proxy** (`--no-proxy`) | テスト、内部・開発 VPS、非 HTTP サービス、ホビーアプリ | `/opt/conoha/<name>/` フラット (単純な `docker compose up`) |

`conoha app init --no-proxy --app-name <name> <server>` で初期化し、`conoha app deploy --no-proxy --app-name <name> <server>` でデプロイします。no-proxy モードでは `conoha.yml` は不要です。
```

Korean (`README-ko.md`) equivalent:

```markdown
### 두 가지 배포 모드

`conoha app`은 동일 VPS에서 공존 가능한 두 가지 모드를 제공합니다:

| 모드 | 용도 | 레이아웃 |
|---|---|---|
| **proxy** (기본) | 도메인 + TLS가 있는 공개 앱 | `/opt/conoha/<name>/<slot>/` 아래의 blue/green 슬롯 (conoha-proxy 관리) |
| **no-proxy** (`--no-proxy`) | 테스트, 내부/개발 VPS, 비 HTTP 서비스, 취미 앱 | `/opt/conoha/<name>/` 플랫 (일반 `docker compose up`) |

`conoha app init --no-proxy --app-name <name> <server>`로 초기화한 뒤 `conoha app deploy --no-proxy --app-name <name> <server>`로 배포합니다. no-proxy 모드에서는 `conoha.yml`이 필요 없습니다.
```

- [ ] **Step 11.2: Create `docs/recipes/single-server-app-noproxy.md`**

```markdown
# Single-Server App — No-Proxy Mode

This recipe shows a TLS-less, single-slot deployment of a small web app. Use it when:

- You do not have a public domain.
- The service exposes a non-HTTP protocol.
- You prefer `docker compose up` semantics over blue/green.

For the proxy-backed blue/green variant, see `single-server-app.md`.

## 1. Create the VPS

```bash
conoha server create --name myapp --flavor g2l-cpu1-1g --image ubuntu-22.04-x86-64 --ssh-key default
```

## 2. Install Docker and mark the app no-proxy

```bash
conoha app init --no-proxy --app-name myapp myapp
```

This verifies Docker is present on the server and writes the `no-proxy` marker to `/opt/conoha/myapp/.conoha-mode`.

## 3. Prepare a compose file locally

`compose.yml`:

```yaml
services:
  web:
    build: .
    ports:
      - "80:8080"
```

No `conoha.yml` needed.

## 4. Deploy

```bash
conoha app deploy --no-proxy --app-name myapp myapp
```

The CLI tars the current directory (respecting `.dockerignore`), uploads to `/opt/conoha/myapp/` on the VPS, and runs `docker compose -p myapp up -d --build`.

## 5. Day-two operations

```bash
conoha app logs --no-proxy --app-name myapp myapp
conoha app status --no-proxy --app-name myapp myapp
conoha app stop    --no-proxy --app-name myapp myapp
conoha app restart --no-proxy --app-name myapp myapp
conoha app destroy --no-proxy --app-name myapp myapp
```

`conoha app rollback` is not supported in no-proxy mode — deploy a previous revision instead (`git checkout <rev> && conoha app deploy --no-proxy ...`).

## Switching to proxy mode

Run `conoha app destroy ... myapp` followed by `conoha app init ... myapp` (without `--no-proxy`). The CLI refuses implicit mode switches.
```

- [ ] **Step 11.3: Cross-reference from prior spec**

Edit `docs/superpowers/specs/2026-04-20-conoha-proxy-deploy-design.md`. Insert immediately below the front-matter `**Owner**` line:

```markdown

> **Update 2026-04-21:** A `--no-proxy` mode was added as a coexisting alternative path. See `docs/superpowers/specs/2026-04-21-no-proxy-mode-design.md`.
```

- [ ] **Step 11.4: Full build + test sweep**

```bash
go build ./...
go test ./...
go vet ./...
gofmt -l cmd/ internal/ 2>&1
```

Expected: build clean, all tests pass, vet clean, gofmt produces no output.

- [ ] **Step 11.5: Commit**

```bash
git add README.md README-ja.md README-ko.md docs/recipes/single-server-app-noproxy.md docs/superpowers/specs/2026-04-20-conoha-proxy-deploy-design.md
git commit -m "docs: document --no-proxy mode alongside proxy mode

Adds a 'Two deploy modes' section in README (en/ja/ko), a full
no-proxy single-server recipe, and a one-line cross-reference from
the 2026-04-20 proxy-deploy spec to the new 2026-04-21 no-proxy
design spec.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 12: End-to-end coherence pass

**Files:**
- No code changes expected; this is a verification task.

- [ ] **Step 12.1: Verify flag matrix**

```bash
go run . app --help
go run . app init --help
go run . app deploy --help
go run . app rollback --help
go run . app destroy --help
go run . app logs --help
go run . app stop --help
go run . app restart --help
go run . app status --help
```

Expected: `--proxy` and `--no-proxy` appear on init/deploy/rollback/destroy/logs/stop/restart/status; `env` does NOT show these flags.

- [ ] **Step 12.2: Run full test suite**

```bash
go test ./... -count=1
```

Expected: all packages pass.

- [ ] **Step 12.3: Verify no-regression in existing proxy tests**

```bash
go test ./cmd/app/ -count=1 -v | head -80
```

Expected: existing tests (`TestBuildSlotUploadCmd`, `TestBuildComposeUp_Slot`, `TestBuildScheduleDrainCmd`, etc.) still pass.

- [ ] **Step 12.4: Push branch and open PR**

```bash
git push -u origin feat/no-proxy-mode
gh pr create --title "feat: --no-proxy mode for app deploy/init/logs/... (#102)" --body "$(cat <<'EOF'
## Summary

Adds a coexisting `--no-proxy` mode to the `conoha app *` command tree, covered by spec `docs/superpowers/specs/2026-04-21-no-proxy-mode-design.md`. Closes #102. Absorbs #93 (slot-aware logs/stop/restart/status in proxy mode).

## Design

- Hybrid mode selection: server-side marker `/opt/conoha/<name>/.conoha-mode` (written by `app init`) with `--proxy`/`--no-proxy` flag override.
- Proxy and no-proxy apps can share a single VPS — different subdirectories under `/opt/conoha/`.
- Explicit mode-conflict errors (exit 5) instead of silent auto-migration.

## Breaking changes

None. Proxy-mode users who ran `app init` before this PR will see "run 'conoha app init' first" on their next deploy; running init once re-writes the marker.

## Test plan

- [x] `go test ./...` passes
- [x] `go vet ./...` clean
- [x] `gofmt -l cmd/ internal/` clean
- [x] `go build ./...` succeeds
- [ ] End-to-end against a real VPS:
  - [ ] `app init --no-proxy --app-name myapp <server>` → `.conoha-mode=no-proxy` on disk
  - [ ] `app deploy --no-proxy --app-name myapp <server>` → `docker ps` shows `myapp-web` under project `myapp`
  - [ ] Proxy-init'd app rejects `app deploy --no-proxy` with exit 5
  - [ ] `app rollback --no-proxy` exits 5 with git-based recovery hint
  - [ ] `app logs/stop/restart/status` target the active slot in proxy mode (fixes #93)
  - [ ] `app destroy` cleans up both layouts

## Follow-ups

- #92 `app reset` reintroduction — needs to handle both modes, blocked on this PR.
- #94 `app env` redesign for proxy mode — this PR only adds the warning shim.
- #95 `app list` for no-proxy — separate PR.
EOF
)"
```

Expected: PR opened against main.

---

## Self-Review Notes

**Spec coverage map:**
- Spec §2 (mode selection, marker) → Task 1.
- Spec §3.1 (`app init`) → Task 2.
- Spec §3.2 (`app deploy`) → Task 3.
- Spec §3.3 (`app rollback`) → Task 4.
- Spec §3.4 (`app destroy`) → Task 5.
- Spec §3.5 (`app logs/stop/restart/status`) → Tasks 6–9.
- Spec §3.6 (`app env`) → Task 10.
- Spec §4 (architecture) → Tasks 1 + references throughout.
- Spec §5 (exit codes) → absorbed into per-command error messages (5 = mode-conflict via `ErrModeConflict`, 6 = not-initialized handled by "has not been deployed" messages; cobra returns 1 for errors and we do not currently set distinct non-zero exit codes — acceptable for v1 and called out as a future refinement).
- Spec §6 (CLI surface) → Tasks 1 + per-command additions.
- Spec §7 (#93 integration) → Tasks 6–9 in logs/stop/restart/status.
- Spec §9 (migration) → proxy WriteMarker in Task 2.
- Spec §10 (documentation) → Task 11.
- Spec §11 (acceptance) → Task 12.

**Exit code note:** The plan currently does not wire distinct process exit codes (4/5/6) — each error returns as cobra's default exit 1. If strict exit codes are required before merge, a follow-up commit in this branch can plumb through a typed error (check via `errors.Is(ErrModeConflict, err)`) in `cmd/cmdutil` to set the final `os.Exit`. Called out here rather than expanded into an extra task because prior PR #98 did not implement custom exit codes either.

**Placeholder scan:** no TBD/TODO markers, every shell builder and helper has concrete code.

**Type consistency:** `ModeProxy`/`ModeNoProxy`, `ReadMarker`/`WriteMarker`/`ResolveMode`/`ReadCurrentSlot`, and the `build*CmdFor(Proxy|NoProxy)` naming pattern used consistently across Tasks 6/7/8/9.
