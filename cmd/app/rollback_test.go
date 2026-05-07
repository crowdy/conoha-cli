package app

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

func TestRollbackCmd_HasModeFlags(t *testing.T) {
	if rollbackCmd.Flags().Lookup("proxy") == nil {
		t.Error("rollback should have --proxy flag")
	}
	if rollbackCmd.Flags().Lookup("no-proxy") == nil {
		t.Error("rollback should have --no-proxy flag")
	}
}

func TestRollbackCmd_HasTargetFlag(t *testing.T) {
	f := rollbackCmd.Flags().Lookup("target")
	if f == nil {
		t.Fatal("rollback should have --target flag")
		return
	}
	if f.DefValue != "" {
		t.Errorf("--target default = %q, want empty", f.DefValue)
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

func bg(v bool) *bool { return &v }

func TestValidateRollbackTarget(t *testing.T) {
	pf := &config.ProjectFile{
		Name: "gitea",
		Expose: []config.ExposeBlock{
			{Label: "dex"},
			{Label: "api"},
		},
	}
	for _, target := range []string{"", "web", "dex", "api"} {
		if err := validateRollbackTarget(pf, target); err != nil {
			t.Errorf("target=%q: unexpected err %v", target, err)
		}
	}
	err := validateRollbackTarget(pf, "nope")
	if err == nil {
		t.Fatal("target=nope: expected error")
	}
	for _, want := range []string{"nope", "web", "dex", "api"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("err missing %q: %v", want, err)
		}
	}
}

func TestRollbackServiceOrder(t *testing.T) {
	// Deploy order (phase 3): expose[0], expose[1], root.
	// Rollback reverse: root, expose[1], expose[0].
	pf := &config.ProjectFile{
		Name: "gitea",
		Expose: []config.ExposeBlock{
			{Label: "dex"},
			{Label: "api"},
		},
	}
	got := rollbackServiceOrder(pf)
	want := []string{"gitea", "gitea-api", "gitea-dex"}
	if !equalStrings(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}

func TestRollbackServiceOrder_SkipsBlueGreenFalse(t *testing.T) {
	pf := &config.ProjectFile{
		Name: "gitea",
		Expose: []config.ExposeBlock{
			{Label: "dex", BlueGreen: bg(false)},
			{Label: "api"},
		},
	}
	got := rollbackServiceOrder(pf)
	want := []string{"gitea", "gitea-api"}
	if !equalStrings(got, want) {
		t.Errorf("order = %v, want %v (dex is blue_green:false → accessory, no rotation)", got, want)
	}
}

func TestRollbackServiceOrder_NoExpose(t *testing.T) {
	pf := &config.ProjectFile{Name: "myapp"}
	got := rollbackServiceOrder(pf)
	if !equalStrings(got, []string{"myapp"}) {
		t.Errorf("order = %v, want [myapp]", got)
	}
}

// fakeRollbackClient records every call and dispatches per-name errors.
type fakeRollbackClient struct {
	calls      []fakeRollbackCall
	errByName  map[string]error
	svcByName  map[string]*proxypkg.Service
	defaultErr error
}

func (f *fakeRollbackClient) Rollback(name string, drainMs int) (*proxypkg.Service, error) {
	f.calls = append(f.calls, fakeRollbackCall{Name: name, DrainMs: drainMs})
	if err, ok := f.errByName[name]; ok {
		return nil, err
	}
	if f.defaultErr != nil {
		return nil, f.defaultErr
	}
	if s, ok := f.svcByName[name]; ok {
		return s, nil
	}
	return &proxypkg.Service{Name: name, Phase: proxypkg.PhaseLive}, nil
}

func TestRunRollbackAll_OrderRootFirstThenExposeReverse(t *testing.T) {
	pf := &config.ProjectFile{
		Name: "gitea",
		Expose: []config.ExposeBlock{
			{Label: "dex"},
			{Label: "api"},
		},
	}
	admin := &fakeRollbackClient{}
	var buf bytes.Buffer
	if err := runRollbackAll(admin, pf, 5000, &buf); err != nil {
		t.Fatalf("runRollbackAll: %v", err)
	}
	if len(admin.calls) != 3 {
		t.Fatalf("calls = %d, want 3", len(admin.calls))
	}
	gotNames := []string{admin.calls[0].Name, admin.calls[1].Name, admin.calls[2].Name}
	wantNames := []string{"gitea", "gitea-api", "gitea-dex"}
	if !equalStrings(gotNames, wantNames) {
		t.Errorf("call order = %v, want %v", gotNames, wantNames)
	}
	for _, c := range admin.calls {
		if c.DrainMs != 5000 {
			t.Errorf("drainMs = %d, want 5000", c.DrainMs)
		}
	}
}

func TestRunRollbackAll_409NoDrainTargetContinues(t *testing.T) {
	pf := &config.ProjectFile{
		Name:   "gitea",
		Expose: []config.ExposeBlock{{Label: "dex"}},
	}
	admin := &fakeRollbackClient{
		errByName: map[string]error{"gitea": proxypkg.ErrNoDrainTarget},
	}
	var buf bytes.Buffer
	if err := runRollbackAll(admin, pf, 0, &buf); err != nil {
		t.Fatalf("409 must not abort runRollbackAll, got %v", err)
	}
	if len(admin.calls) != 2 {
		t.Fatalf("want 2 calls (continued after 409), got %d", len(admin.calls))
	}
	if !strings.Contains(buf.String(), "warning") || !strings.Contains(buf.String(), "gitea") {
		t.Errorf("expected 409 warning in output, got %q", buf.String())
	}
}

func TestRunRollbackAll_NonDrainErrorRecordedAsExit(t *testing.T) {
	pf := &config.ProjectFile{
		Name:   "gitea",
		Expose: []config.ExposeBlock{{Label: "dex"}},
	}
	admin := &fakeRollbackClient{
		errByName: map[string]error{"gitea-dex": errors.New("500 internal")},
	}
	var buf bytes.Buffer
	err := runRollbackAll(admin, pf, 0, &buf)
	if err == nil {
		t.Fatal("non-409 error must bubble up as return value")
	}
	if !strings.Contains(err.Error(), "gitea-dex") {
		t.Errorf("err should mention failing service: %v", err)
	}
	// Root rollback still succeeded before the expose failure — both attempted.
	if len(admin.calls) != 2 {
		t.Errorf("want 2 calls (root first, then failing expose), got %d", len(admin.calls))
	}
}

func TestRunRollbackSingle_WebIsRoot(t *testing.T) {
	pf := &config.ProjectFile{Name: "gitea"}
	admin := &fakeRollbackClient{}
	var buf bytes.Buffer
	if err := runRollbackSingle(admin, pf, "web", 0, &buf); err != nil {
		t.Fatal(err)
	}
	if len(admin.calls) != 1 || admin.calls[0].Name != "gitea" {
		t.Errorf("calls = %+v, want single [gitea]", admin.calls)
	}
}

func TestRunRollbackSingle_ExposeLabel(t *testing.T) {
	pf := &config.ProjectFile{
		Name:   "gitea",
		Expose: []config.ExposeBlock{{Label: "dex"}},
	}
	admin := &fakeRollbackClient{}
	var buf bytes.Buffer
	if err := runRollbackSingle(admin, pf, "dex", 0, &buf); err != nil {
		t.Fatal(err)
	}
	if len(admin.calls) != 1 || admin.calls[0].Name != "gitea-dex" {
		t.Errorf("calls = %+v, want single [gitea-dex]", admin.calls)
	}
}

func TestRunRollbackSingle_NoDrainTargetIsFatal(t *testing.T) {
	pf := &config.ProjectFile{Name: "gitea"}
	admin := &fakeRollbackClient{defaultErr: proxypkg.ErrNoDrainTarget}
	err := runRollbackSingle(admin, pf, "web", 0, &bytes.Buffer{})
	if err == nil {
		t.Fatal("single-target rollback should surface 409 as a user-facing error")
	}
	if !strings.Contains(err.Error(), "drain window") {
		t.Errorf("err should mention drain window: %v", err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
