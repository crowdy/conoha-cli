package app

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

// Compile-time assertions: the real proxy client and our fake must both
// satisfy DeployProxyAPI. If either drifts (e.g. Get/Deploy renamed), the
// test binary will fail to build.
var (
	_ DeployOps      = (*fakeOps)(nil)
	_ DeployProxyAPI = (*proxypkg.Client)(nil)
	_ DeployProxyAPI = (*fakeProxyAPI)(nil)
)

// fakeOps is a DeployOps fake that records every command it receives.
// Commands are matched against a RunOverrides table so individual tests can
// inject specific responses (stdout bytes, exit codes, errors) without
// having to stub the full sequence.
type fakeOps struct {
	Commands  []fakeOpsCall
	Overrides map[string]fakeOpsResponse // substring → response
	Default   fakeOpsResponse            // applied when no override matches
	Proxy_    *fakeProxyAPI
}

type fakeOpsCall struct {
	Cmd    string
	Stdin  []byte
	Stream bool // was this a RunStream call?
}

type fakeOpsResponse struct {
	ExitCode int
	Stdout   string
	Err      error
}

func (f *fakeOps) resolve(cmd string) fakeOpsResponse {
	for sub, r := range f.Overrides {
		if strings.Contains(cmd, sub) {
			return r
		}
	}
	return f.Default
}

func (f *fakeOps) record(cmd string, stdin io.Reader, stream bool) []byte {
	call := fakeOpsCall{Cmd: cmd, Stream: stream}
	if stdin != nil {
		call.Stdin, _ = io.ReadAll(stdin)
	}
	f.Commands = append(f.Commands, call)
	return call.Stdin
}

func (f *fakeOps) Run(cmd string, stdin io.Reader) (int, []byte, error) {
	f.record(cmd, stdin, false)
	r := f.resolve(cmd)
	return r.ExitCode, []byte(r.Stdout), r.Err
}

func (f *fakeOps) RunStream(cmd string, stdin io.Reader, stdout io.Writer) (int, error) {
	f.record(cmd, stdin, true)
	r := f.resolve(cmd)
	if r.Stdout != "" {
		_, _ = io.WriteString(stdout, r.Stdout)
	}
	return r.ExitCode, r.Err
}

func (f *fakeOps) Proxy() DeployProxyAPI {
	if f.Proxy_ == nil {
		f.Proxy_ = &fakeProxyAPI{}
	}
	return f.Proxy_
}

// fakeProxyAPI is a minimal DeployProxyAPI fake.
type fakeProxyAPI struct {
	GetCalls      int
	DeployCalls   []fakeDeployCall
	RollbackCalls []fakeRollbackCall
	GetReturn     *proxypkg.Service
	GetErr        error
	DeployReturn  *proxypkg.Service
	// DeployErrByName lets tests inject per-service /deploy errors for the
	// multi-block path. If nil, falls back to DeployErr (applied to every
	// call) for compatibility with single-block tests.
	DeployErrByName map[string]error
	DeployErr       error
	RollbackErr     error
	// RollbackErrByName, like DeployErrByName, lets tests inject per-service
	// rollback errors (ErrNoDrainTarget on one block, success on another).
	RollbackErrByName map[string]error
}

type fakeDeployCall struct {
	Name string
	Req  proxypkg.DeployRequest
}

type fakeRollbackCall struct {
	Name    string
	DrainMs int
}

func (f *fakeProxyAPI) Get(name string) (*proxypkg.Service, error) {
	f.GetCalls++
	if f.GetErr != nil {
		return nil, f.GetErr
	}
	if f.GetReturn != nil {
		return f.GetReturn, nil
	}
	return &proxypkg.Service{Name: name}, nil
}

func (f *fakeProxyAPI) Deploy(name string, req proxypkg.DeployRequest) (*proxypkg.Service, error) {
	f.DeployCalls = append(f.DeployCalls, fakeDeployCall{Name: name, Req: req})
	if err, ok := f.DeployErrByName[name]; ok {
		return nil, err
	}
	if f.DeployErr != nil {
		return nil, f.DeployErr
	}
	if f.DeployReturn != nil {
		return f.DeployReturn, nil
	}
	return &proxypkg.Service{
		Name:         name,
		Phase:        proxypkg.PhaseLive,
		ActiveTarget: &proxypkg.Target{URL: req.TargetURL},
	}, nil
}

func (f *fakeProxyAPI) Rollback(name string, drainMs int) (*proxypkg.Service, error) {
	f.RollbackCalls = append(f.RollbackCalls, fakeRollbackCall{Name: name, DrainMs: drainMs})
	if err, ok := f.RollbackErrByName[name]; ok {
		return nil, err
	}
	if f.RollbackErr != nil {
		return nil, f.RollbackErr
	}
	return &proxypkg.Service{Name: name, Phase: proxypkg.PhaseLive}, nil
}

// baseParams constructs a minimal proxyDeployParams suitable for most tests.
// Tests override specific fields.
func baseParams() proxyDeployParams {
	return proxyDeployParams{
		ProjectFile: &config.ProjectFile{
			Name:  "myapp",
			Hosts: []string{"app.example.com"},
			Web: config.WebSpec{
				Service: "web",
				Port:    8080,
			},
			Deploy: &config.DeploySpec{DrainMs: 2000},
		},
		ComposeFile:  "docker-compose.yml",
		ServerID:     "srv-1",
		ServerName:   "test-vps",
		ServerIP:     "203.0.113.9",
		SlotOverride: "abc1234",
		Archive: func() (io.Reader, error) {
			return bytes.NewReader([]byte("fake-tar-payload")), nil
		},
	}
}

// successOps returns a fakeOps pre-configured so the happy path flows
// end-to-end. Port discovery emits the expected `docker port` format.
func successOps() *fakeOps {
	return &fakeOps{
		Default: fakeOpsResponse{ExitCode: 0},
		Overrides: map[string]fakeOpsResponse{
			// `docker port` output is "addr:port" per line, not the "->" form
			// emitted by `docker ps`. See extractHostPort / parseColonPort.
			"docker port": {ExitCode: 0, Stdout: "127.0.0.1:34567\n"},
			// accessory probe: 0 = exists (skip up). Happy path has no accessories
			// so this isn't hit, but defensive.
			"docker inspect": {ExitCode: 0},
			// CURRENT_SLOT read (cat form only — the write uses printf).
			"cat '/opt/conoha/myapp/CURRENT_SLOT'": {ExitCode: 0, Stdout: ""},
		},
	}
}

func TestRunProxyDeployState_HappyPath_FirstDeploy(t *testing.T) {
	ops := successOps()
	if err := runProxyDeployState(baseParams(), ops); err != nil {
		t.Fatalf("happy path failed: %v", err)
	}
	if ops.Proxy_.GetCalls != 1 {
		t.Errorf("expected 1 admin.Get call, got %d", ops.Proxy_.GetCalls)
	}
	if len(ops.Proxy_.DeployCalls) != 1 {
		t.Fatalf("expected 1 admin.Deploy call, got %d", len(ops.Proxy_.DeployCalls))
	}
	if got := ops.Proxy_.DeployCalls[0].Req.TargetURL; got != "http://127.0.0.1:34567" {
		t.Errorf("TargetURL = %q, want http://127.0.0.1:34567", got)
	}
	if got := ops.Proxy_.DeployCalls[0].Req.DrainMs; got != 2000 {
		t.Errorf("DrainMs = %d, want 2000 (from pf.Deploy override)", got)
	}
	if got := ops.Proxy_.DeployCalls[0].Name; got != "myapp" {
		t.Errorf("Name = %q, want myapp", got)
	}

	// Expected command ordering:
	//   1. upload (RunStream with stdin)
	//   2. write compose override
	//   3. compose up slot
	//   4. docker port
	//   5. cat CURRENT_SLOT
	//   6. printf slot > CURRENT_SLOT
	// Teardown must NOT appear.
	mustOrdered(t, ops.Commands,
		"tar xzf",
		"conoha-override.yml",
		"docker compose -p myapp-abc1234",
		"docker port",
		"CURRENT_SLOT",
		"printf %s 'abc1234'",
	)
	mustAbsent(t, ops.Commands, "down 2>/dev/null")
}

func TestRunProxyDeployState_ServiceNotInitialized(t *testing.T) {
	ops := successOps()
	ops.Proxy_ = &fakeProxyAPI{GetErr: errors.New("404 not_found")}
	err := runProxyDeployState(baseParams(), ops)
	if err == nil {
		t.Fatal("expected error when admin.Get fails")
	}
	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("want 'not initialized' hint, got: %v", err)
	}
	// No side-effecting commands should have run.
	for _, c := range ops.Commands {
		if strings.Contains(c.Cmd, "tar xzf") || strings.Contains(c.Cmd, "compose up") {
			t.Errorf("unexpected command executed after init check failed: %q", c.Cmd)
		}
	}
}

func TestRunProxyDeployState_PortDiscoveryFails_TearsDown(t *testing.T) {
	ops := successOps()
	ops.Overrides["docker port"] = fakeOpsResponse{Err: errors.New("ssh channel closed")}
	err := runProxyDeployState(baseParams(), ops)
	if err == nil {
		t.Fatal("expected error from docker port failure")
	}
	if !strings.Contains(err.Error(), "docker port") {
		t.Errorf("want 'docker port' context in error, got: %v", err)
	}
	// Teardown must be called.
	mustPresent(t, ops.Commands, "down 2>/dev/null")
	mustPresent(t, ops.Commands, "rm -rf '/opt/conoha/myapp/abc1234'")
	// admin.Deploy must NOT have been called.
	if len(ops.Proxy_.DeployCalls) != 0 {
		t.Errorf("admin.Deploy should not be called after port discovery failure, got %d calls", len(ops.Proxy_.DeployCalls))
	}
}

func TestRunProxyDeployState_PortParseFails_TearsDown(t *testing.T) {
	ops := successOps()
	// docker port returns garbage — extractHostPort should fail.
	ops.Overrides["docker port"] = fakeOpsResponse{ExitCode: 0, Stdout: "not a port mapping\n"}
	err := runProxyDeployState(baseParams(), ops)
	if err == nil {
		t.Fatal("expected error from unparseable port output")
	}
	mustPresent(t, ops.Commands, "down 2>/dev/null")
}

func TestRunProxyDeployState_ProxyDeployFails_TearsDown(t *testing.T) {
	ops := successOps()
	ops.Proxy_ = &fakeProxyAPI{
		DeployErr: &proxypkg.ProbeFailedError{Message: "upstream /up returned 500"},
	}
	err := runProxyDeployState(baseParams(), ops)
	if err == nil {
		t.Fatal("expected error from admin.Deploy failure")
	}
	var pe *proxypkg.ProbeFailedError
	if !errors.As(err, &pe) {
		t.Errorf("expected ProbeFailedError to propagate, got %T: %v", err, err)
	}
	// Teardown MUST run on 424 — the proxy didn't mutate state.
	mustPresent(t, ops.Commands, "down 2>/dev/null")
}

func TestRunProxyDeployState_OldSlotDrainScheduled(t *testing.T) {
	ops := successOps()
	ops.Overrides["cat '/opt/conoha/myapp/CURRENT_SLOT'"] = fakeOpsResponse{ExitCode: 0, Stdout: "prev5678\n"}
	if err := runProxyDeployState(baseParams(), ops); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}
	// The old slot's compose project must be the target of the drain
	// teardown — buildScheduleDrainCmd embeds the slot name in the project
	// (myapp-prev5678) and in the work dir (/opt/conoha/myapp/prev5678).
	mustPresent(t, ops.Commands, "myapp-prev5678")
	mustPresent(t, ops.Commands, "/opt/conoha/myapp/prev5678")
	// The NEW slot must NOT be torn down on the success path — a regression
	// that inverted old/new would still leak the app live but kill it mid-
	// drain. Guard that explicitly.
	for _, c := range ops.Commands {
		if strings.Contains(c.Cmd, "myapp-abc1234") && strings.Contains(c.Cmd, " down ") {
			t.Errorf("new slot abc1234 was torn down on success path: %q", c.Cmd)
		}
	}
}

func TestRunProxyDeployState_InvalidCurrentSlot_IgnoresWithWarning(t *testing.T) {
	ops := successOps()
	ops.Overrides["cat '/opt/conoha/myapp/CURRENT_SLOT'"] = fakeOpsResponse{ExitCode: 0, Stdout: "CapitalsNotAllowed!"}
	err := runProxyDeployState(baseParams(), ops)
	if err != nil {
		t.Fatalf("deploy should succeed despite invalid CURRENT_SLOT: %v", err)
	}
	// No schedule drain teardown should happen for an invalid slot name.
	for _, c := range ops.Commands {
		if strings.Contains(c.Cmd, "CapitalsNotAllowed") {
			t.Errorf("invalid slot name leaked into a follow-up command: %q", c.Cmd)
		}
	}
}

// --- command-sequence helpers ---

func commandTexts(cs []fakeOpsCall) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Cmd
	}
	return out
}

// mustOrdered checks that each substring appears in the command list in the
// given order (not necessarily contiguously).
func mustOrdered(t *testing.T, cs []fakeOpsCall, subs ...string) {
	t.Helper()
	cmds := commandTexts(cs)
	pos := 0
	for _, want := range subs {
		found := -1
		for i := pos; i < len(cmds); i++ {
			if strings.Contains(cmds[i], want) {
				found = i
				break
			}
		}
		if found == -1 {
			t.Errorf("missing (in order) %q after position %d. Commands:\n  %s", want, pos, strings.Join(cmds, "\n  "))
			return
		}
		pos = found + 1
	}
}

func mustPresent(t *testing.T, cs []fakeOpsCall, sub string) {
	t.Helper()
	for _, c := range cs {
		if strings.Contains(c.Cmd, sub) {
			return
		}
	}
	t.Errorf("expected a command containing %q, got:\n  %s", sub, strings.Join(commandTexts(cs), "\n  "))
}

func mustAbsent(t *testing.T, cs []fakeOpsCall, sub string) {
	t.Helper()
	for _, c := range cs {
		if strings.Contains(c.Cmd, sub) {
			t.Errorf("command %q should not be present. Actual: %q", sub, c.Cmd)
			return
		}
	}
}
