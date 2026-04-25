package app

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

// twoBlockParams returns a proxyDeployParams with one root web + one expose
// block, both participating in blue/green rotation.
func twoBlockParams() proxyDeployParams {
	p := baseParams()
	p.ProjectFile.Expose = []config.ExposeBlock{
		{Label: "dex", Host: "dex.example.com", Service: "dex", Port: 5556},
	}
	return p
}

// multiBlockOps wires successOps for both blocks: `docker port` must return
// distinct host ports for the two containers.
func multiBlockOps() *fakeOps {
	ops := successOps()
	// Default "docker port" override returns one value; replace with a
	// per-container rule via longer-matching substrings.
	delete(ops.Overrides, "docker port")
	ops.Overrides["docker port myapp-abc1234-web 8080"] = fakeOpsResponse{ExitCode: 0, Stdout: "127.0.0.1:34567\n"}
	ops.Overrides["docker port myapp-abc1234-dex 5556"] = fakeOpsResponse{ExitCode: 0, Stdout: "127.0.0.1:34568\n"}
	return ops
}

func TestRunProxyDeployState_TwoBlocks_HappyPath(t *testing.T) {
	ops := multiBlockOps()
	if err := runProxyDeployState(twoBlockParams(), ops); err != nil {
		t.Fatalf("happy 2-block deploy failed: %v", err)
	}

	// Both /deploy calls were made, expose before root.
	if n := len(ops.Proxy_.DeployCalls); n != 2 {
		t.Fatalf("deploys = %d, want 2", n)
	}
	order := []string{ops.Proxy_.DeployCalls[0].Name, ops.Proxy_.DeployCalls[1].Name}
	if order[0] != "myapp-dex" || order[1] != "myapp" {
		t.Errorf("deploy order = %v, want [myapp-dex, myapp]", order)
	}
	if got := ops.Proxy_.DeployCalls[0].Req.TargetURL; got != "http://127.0.0.1:34568" {
		t.Errorf("expose target URL = %q, want http://127.0.0.1:34568", got)
	}
	if got := ops.Proxy_.DeployCalls[1].Req.TargetURL; got != "http://127.0.0.1:34567" {
		t.Errorf("root target URL = %q, want http://127.0.0.1:34567", got)
	}
	if len(ops.Proxy_.RollbackCalls) != 0 {
		t.Errorf("no rollbacks expected on happy path, got %v", ops.Proxy_.RollbackCalls)
	}

	// Override YAML has both services.
	mustPresent(t, ops.Commands, "container_name: myapp-abc1234-web")
	mustPresent(t, ops.Commands, "container_name: myapp-abc1234-dex")
	mustPresent(t, ops.Commands, "up -d --build --no-deps web dex")

	// No teardown of new slot on success.
	mustAbsent(t, ops.Commands, "down 2>/dev/null")
}

func TestRunProxyDeployState_TwoBlocks_RootFailsMidway(t *testing.T) {
	// The root /deploy (last in the order) fails with 424. Expected: the
	// expose block that already swapped is rolled back, and the new slot
	// is torn down.
	ops := multiBlockOps()
	ops.Proxy_ = &fakeProxyAPI{
		DeployErrByName: map[string]error{
			"myapp": &proxypkg.ProbeFailedError{Message: "upstream /up returned 500"},
		},
	}

	err := runProxyDeployState(twoBlockParams(), ops)
	if err == nil {
		t.Fatal("want error from failing root deploy")
	}
	var pe *proxypkg.ProbeFailedError
	if !errors.As(err, &pe) {
		t.Errorf("want ProbeFailedError, got %T: %v", err, err)
	}

	// One rollback issued, against the earlier-swapped expose block.
	if len(ops.Proxy_.RollbackCalls) != 1 {
		t.Fatalf("rollbacks = %d, want 1", len(ops.Proxy_.RollbackCalls))
	}
	if got := ops.Proxy_.RollbackCalls[0].Name; got != "myapp-dex" {
		t.Errorf("rollback target = %q, want myapp-dex", got)
	}
	if got := ops.Proxy_.RollbackCalls[0].DrainMs; got != 2000 {
		t.Errorf("rollback drainMs = %d, want 2000 (pf.Deploy.DrainMs)", got)
	}

	// Slot teardown must happen.
	mustPresent(t, ops.Commands, "down 2>/dev/null")
	mustPresent(t, ops.Commands, "rm -rf '/opt/conoha/myapp/abc1234'")
}

func TestRunProxyDeployState_TwoBlocks_ExposeFailsFirst(t *testing.T) {
	// First /deploy (expose) fails. Nothing to roll back. Root /deploy must
	// not have been called. Slot torn down.
	ops := multiBlockOps()
	ops.Proxy_ = &fakeProxyAPI{
		DeployErrByName: map[string]error{
			"myapp-dex": &proxypkg.ProbeFailedError{Message: "dex probe failed"},
		},
	}

	err := runProxyDeployState(twoBlockParams(), ops)
	if err == nil {
		t.Fatal("want error")
	}

	if got := callNames(ops.Proxy_.DeployCalls); len(got) != 1 || got[0] != "myapp-dex" {
		t.Errorf("deploy calls = %v, want only [myapp-dex]", got)
	}
	if len(ops.Proxy_.RollbackCalls) != 0 {
		t.Errorf("no rollbacks expected when first /deploy fails, got %v", rollbackNames(ops.Proxy_.RollbackCalls))
	}
	mustPresent(t, ops.Commands, "down 2>/dev/null")
}

func TestRunProxyDeployState_TwoBlocks_RollbackDrainExpiredDegrades(t *testing.T) {
	// Root /deploy fails after expose swapped; rollback of the expose
	// returns ErrNoDrainTarget (409). The CLI must degrade to a warning
	// and still propagate the original /deploy error (not the rollback's).
	ops := multiBlockOps()
	ops.Proxy_ = &fakeProxyAPI{
		DeployErrByName: map[string]error{
			"myapp": &proxypkg.ProbeFailedError{Message: "root probe failed"},
		},
		RollbackErrByName: map[string]error{
			"myapp-dex": fmt.Errorf("rollback: %w", proxypkg.ErrNoDrainTarget),
		},
	}

	err := runProxyDeployState(twoBlockParams(), ops)
	if err == nil {
		t.Fatal("want error")
	}
	var pe *proxypkg.ProbeFailedError
	if !errors.As(err, &pe) {
		t.Errorf("caller must see the original ProbeFailedError, not the rollback's 409. got %T: %v", err, err)
	}
	if len(ops.Proxy_.RollbackCalls) != 1 {
		t.Errorf("rollback must still have been attempted, got %d", len(ops.Proxy_.RollbackCalls))
	}
}

func TestRunProxyDeployState_BlueGreenFalseExposeGoesToAccessories(t *testing.T) {
	// blue_green:false expose block doesn't participate in slot rotation
	// (no slot /deploy entry), but the proxy still gets a target — the
	// accessory-project container's host port. See issue #163.
	p := baseParams()
	falseB := false
	p.ProjectFile.Expose = []config.ExposeBlock{
		{Label: "admin", Host: "admin.example.com", Service: "studio", Port: 3000, BlueGreen: &falseB},
	}

	ops := multiBlockOps()
	// No slot dex container to port-discover (it's not in the slot blocks).
	delete(ops.Overrides, "docker port myapp-abc1234-dex 5556")
	// Accessory project's docker-default container name = "<project>-<service>-1".
	ops.Overrides["docker port myapp-accessories-studio-1 3000"] = fakeOpsResponse{ExitCode: 0, Stdout: "127.0.0.1:39000\n"}
	// No accessories yet → existence probe returns non-zero (will trigger up).
	ops.Overrides["docker compose -p myapp-accessories ps -q"] = fakeOpsResponse{ExitCode: 1, Stdout: "0\n"}

	if err := runProxyDeployState(p, ops); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	// Two /deploy calls: the fixed expose (admin) is pushed before slot deploys,
	// then root web from the slot loop.
	if n := len(ops.Proxy_.DeployCalls); n != 2 {
		t.Fatalf("deploys = %d, want 2 (fixed expose + root web)", n)
	}
	if ops.Proxy_.DeployCalls[0].Name != "myapp-admin" {
		t.Errorf("first deploy should be the fixed expose %q, got %q", "myapp-admin", ops.Proxy_.DeployCalls[0].Name)
	}
	if ops.Proxy_.DeployCalls[0].Req.TargetURL != "http://127.0.0.1:39000" {
		t.Errorf("fixed expose target = %q, want http://127.0.0.1:39000", ops.Proxy_.DeployCalls[0].Req.TargetURL)
	}
	if ops.Proxy_.DeployCalls[1].Name != "myapp" {
		t.Errorf("last deploy should be root, got %q", ops.Proxy_.DeployCalls[1].Name)
	}
	// Accessory compose up still brings the service up alongside any true accessories.
	mustPresent(t, ops.Commands, "-p myapp-accessories")
	mustPresent(t, ops.Commands, "up -d studio")
	// The accessory override (port mapping for blue_green:false expose) must be applied.
	mustPresent(t, ops.Commands, "-f conoha-accessories-override.yml")
}

func TestCollectDeployBlocks_Shape(t *testing.T) {
	trueB, falseB := true, false
	pf := &config.ProjectFile{
		Name: "app",
		Web:  config.WebSpec{Service: "web", Port: 80},
		Expose: []config.ExposeBlock{
			{Label: "dex", Service: "dex", Host: "dex.example.com", Port: 5556, BlueGreen: &trueB},
			{Label: "admin", Service: "studio", Host: "admin.example.com", Port: 3000, BlueGreen: &falseB},
			{Label: "api", Service: "api", Host: "api.example.com", Port: 8000}, // nil → default true
		},
	}
	got := collectDeployBlocks(pf)
	if len(got) != 3 {
		t.Fatalf("blocks = %d, want 3 (root + 2 blue/green expose; admin is blue_green:false)", len(got))
	}
	want := []struct{ svc, proxy string }{
		{"web", "app"},
		{"dex", "app-dex"},
		{"api", "app-api"},
	}
	for i, w := range want {
		if got[i].Service != w.svc || got[i].ProxyName != w.proxy {
			t.Errorf("block[%d] = %+v, want svc=%q proxy=%q", i, got[i], w.svc, w.proxy)
		}
	}
	if !got[0].isRoot() {
		t.Errorf("block[0] must be root")
	}
	if got[1].isRoot() {
		t.Errorf("block[1] must not be root")
	}
}

func TestCollectEffectiveAccessories_IncludesBlueGreenFalse(t *testing.T) {
	falseB := true
	falseB2 := false
	pf := &config.ProjectFile{
		Accessories: []string{"db"},
		Expose: []config.ExposeBlock{
			{Service: "on", BlueGreen: &falseB},
			{Service: "off", BlueGreen: &falseB2},
		},
	}
	got := collectEffectiveAccessories(pf)
	if got := strings.Join(got, ","); got != "db,off" {
		t.Errorf("effective accessories = %q, want %q", got, "db,off")
	}
}

func TestComposeOverrideFor_TwoBlocks(t *testing.T) {
	blocks := []DeployBlock{
		{Service: "web", Port: 8080},
		{Label: "dex", Service: "dex", Port: 5556},
	}
	got := composeOverrideFor("myapp", "abc1234", blocks, false)
	for _, want := range []string{
		"  web:",
		"    container_name: myapp-abc1234-web",
		`      - "127.0.0.1:0:8080"`,
		"  dex:",
		"    container_name: myapp-abc1234-dex",
		`      - "127.0.0.1:0:5556"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// Single env_file line per block.
	if n := strings.Count(got, "/opt/conoha/myapp/.env.server"); n != 2 {
		t.Errorf("env_file lines = %d, want 2 (one per block)", n)
	}
}

func TestComposeOverride_BackwardCompat_SingleRoot(t *testing.T) {
	// No-expose fixture should produce output equivalent to pre-phase-3
	// composeOverride (tested as substring coverage in override_test.go;
	// re-assert here that the single-block wrapper path does not accidentally
	// include an empty expose section or extra blocks).
	got := composeOverride("myapp", "a1b2c3d", "web", 8080, false)
	if strings.Count(got, "container_name:") != 1 {
		t.Errorf("single-block form should emit exactly one container_name, got:\n%s", got)
	}
	if strings.Contains(got, "  :") {
		t.Errorf("empty service key leaked into output:\n%s", got)
	}
}

func callNames(calls []fakeDeployCall) []string {
	out := make([]string, len(calls))
	for i, c := range calls {
		out[i] = c.Name
	}
	return out
}

func rollbackNames(calls []fakeRollbackCall) []string {
	out := make([]string, len(calls))
	for i, c := range calls {
		out[i] = c.Name
	}
	return out
}
