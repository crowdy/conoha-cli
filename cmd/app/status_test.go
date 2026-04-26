package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
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
		`docker ps -a --format '{{.Label "com.docker.compose.project"}}'`,
		`grep -E "^myapp(-|$)"`,
		"docker compose -p",
		"ps",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}
	// Regression: the legacy 'docker compose ls --format "{{.Name}}"' pattern
	// stops working on Docker Compose v5 (#114) — must not come back.
	if strings.Contains(got, "{{.Name}}") {
		t.Errorf("buildStatusCmdForProxy must not use the legacy Go-template format; got: %s", got)
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

// fakeStatusClient is a recording statusClient: returns services from a
// name-keyed map, or an error when the name maps into errs.
type fakeStatusClient struct {
	services map[string]*proxypkg.Service
	errs     map[string]error
	calls    []string
}

func (f *fakeStatusClient) Get(name string) (*proxypkg.Service, error) {
	f.calls = append(f.calls, name)
	if err, ok := f.errs[name]; ok {
		return nil, err
	}
	if s, ok := f.services[name]; ok {
		return s, nil
	}
	return nil, errors.New("fakeStatusClient: unknown service " + name)
}

// Regression for #176: when conoha.yml is absent or invalid, status must
// degrade gracefully — query the root by name, return empty (non-nil)
// expose slice. Before this, status bailed in JSON mode with
// `load conoha.yml: ... no such file or directory`, which was useless for
// monitoring scripts running outside the project dir.
func TestCollectRootOnlyStatus_HappyPath(t *testing.T) {
	admin := &fakeStatusClient{
		services: map[string]*proxypkg.Service{
			"myapp": {Name: "myapp", Phase: proxypkg.PhaseLive},
		},
	}
	r, err := collectRootOnlyStatus(admin, "myapp")
	if err != nil {
		t.Fatalf("collectRootOnlyStatus: %v", err)
	}
	if r.Root == nil || r.Root.Name != "myapp" {
		t.Fatalf("root = %+v, want service myapp", r.Root)
	}
	if r.Expose == nil {
		t.Fatal("Expose should be non-nil empty slice — JSON consumers expect [] not null")
	}
	if len(r.Expose) != 0 {
		t.Errorf("Expose = %+v, want empty", r.Expose)
	}
	// Verify the JSON shape — `expose: []` not `expose: null`. Stable
	// contract for parsers that may have started checking this field.
	buf := &bytes.Buffer{}
	if err := renderStatusJSON(buf, r); err != nil {
		t.Fatalf("renderStatusJSON: %v", err)
	}
	if !strings.Contains(buf.String(), `"expose": []`) {
		t.Errorf("JSON missing `\"expose\": []`:\n%s", buf.String())
	}
}

func TestCollectRootOnlyStatus_RootMissing(t *testing.T) {
	// Proxy doesn't know this app at all. Surface that so JSON consumers
	// can treat it as "service not registered" rather than empty success.
	admin := &fakeStatusClient{
		errs: map[string]error{"myapp": errors.New("not found")},
	}
	_, err := collectRootOnlyStatus(admin, "myapp")
	if err == nil {
		t.Fatal("want error when root not found, got nil")
	}
	if !strings.Contains(err.Error(), "myapp") {
		t.Errorf("error should name the app; got: %v", err)
	}
}

func TestCollectAppStatus_RootOnly(t *testing.T) {
	pf := &config.ProjectFile{Name: "myapp"}
	admin := &fakeStatusClient{
		services: map[string]*proxypkg.Service{
			"myapp": {Name: "myapp", Phase: proxypkg.PhaseLive},
		},
	}
	r, err := collectAppStatus(admin, pf, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("collectAppStatus: %v", err)
	}
	if r.Root == nil || r.Root.Name != "myapp" {
		t.Fatalf("root = %+v, want service myapp", r.Root)
	}
	if len(r.Expose) != 0 {
		t.Errorf("Expose = %+v, want empty", r.Expose)
	}
	// JSON shape parity with collectRootOnlyStatus (#176): zero-expose
	// projects must serialize as `"expose": []`, not `null`. Guards
	// against a future "drop the empty-slice init" refactor that would
	// silently make this path emit null and break consumers that started
	// trusting `[]`.
	if r.Expose == nil {
		t.Error("Expose should be non-nil empty slice for JSON shape stability")
	}
	buf := &bytes.Buffer{}
	if err := renderStatusJSON(buf, r); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"expose": []`) {
		t.Errorf("expected `\"expose\": []` in JSON; got:\n%s", buf.String())
	}
	if got := admin.calls; len(got) != 1 || got[0] != "myapp" {
		t.Errorf("calls = %v, want [myapp]", got)
	}
}

func TestCollectAppStatus_RootAndExpose(t *testing.T) {
	pf := &config.ProjectFile{
		Name: "gitea",
		Expose: []config.ExposeBlock{
			{Label: "dex", Host: "dex.example.com", Service: "dex", Port: 5556},
		},
	}
	admin := &fakeStatusClient{
		services: map[string]*proxypkg.Service{
			"gitea":     {Name: "gitea", Phase: proxypkg.PhaseLive},
			"gitea-dex": {Name: "gitea-dex", Phase: proxypkg.PhaseConfigured},
		},
	}
	r, err := collectAppStatus(admin, pf, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("collectAppStatus: %v", err)
	}
	if r.Root.Name != "gitea" {
		t.Errorf("root = %s, want gitea", r.Root.Name)
	}
	if len(r.Expose) != 1 {
		t.Fatalf("Expose length = %d, want 1", len(r.Expose))
	}
	if r.Expose[0].Label != "dex" {
		t.Errorf("expose[0].label = %q, want dex", r.Expose[0].Label)
	}
	if r.Expose[0].Service == nil || r.Expose[0].Service.Name != "gitea-dex" {
		t.Errorf("expose[0].service = %+v, want gitea-dex", r.Expose[0].Service)
	}
}

func TestCollectAppStatus_RootErrIsFatal(t *testing.T) {
	pf := &config.ProjectFile{Name: "myapp"}
	admin := &fakeStatusClient{errs: map[string]error{"myapp": errors.New("404 not_found")}}
	_, err := collectAppStatus(admin, pf, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error when root Get fails")
	}
}

func TestCollectAppStatus_ExposeErrIsWarning(t *testing.T) {
	pf := &config.ProjectFile{
		Name: "gitea",
		Expose: []config.ExposeBlock{
			{Label: "dex", Host: "dex.example.com", Service: "dex", Port: 5556},
		},
	}
	admin := &fakeStatusClient{
		services: map[string]*proxypkg.Service{"gitea": {Name: "gitea"}},
		errs:     map[string]error{"gitea-dex": errors.New("503 upstream down")},
	}
	var warn bytes.Buffer
	r, err := collectAppStatus(admin, pf, &warn)
	if err != nil {
		t.Fatalf("collectAppStatus must tolerate expose errors, got %v", err)
	}
	if len(r.Expose) != 1 || r.Expose[0].Label != "dex" || r.Expose[0].Service != nil {
		t.Errorf("expose entry = %+v, want label=dex service=nil", r.Expose)
	}
	if !strings.Contains(warn.String(), "gitea-dex") {
		t.Errorf("warn buffer missing expose name; got %q", warn.String())
	}
}

func TestRenderStatusTable_RootAndExpose(t *testing.T) {
	drain := time.Date(2026, 4, 24, 12, 30, 0, 0, time.UTC)
	r := &appStatusReport{
		Root: &proxypkg.Service{
			Name:         "gitea",
			Hosts:        []string{"gitea.example.com"},
			Phase:        proxypkg.PhaseLive,
			ActiveTarget: &proxypkg.Target{URL: "http://127.0.0.1:34567"},
			TLSStatus:    "ready",
		},
		Expose: []exposeStatusEntry{
			{
				Label: "dex",
				Service: &proxypkg.Service{
					Name:           "gitea-dex",
					Hosts:          []string{"dex.example.com"},
					Phase:          proxypkg.PhaseSwapping,
					ActiveTarget:   &proxypkg.Target{URL: "http://127.0.0.1:34568"},
					DrainingTarget: &proxypkg.Target{URL: "http://127.0.0.1:34555"},
					DrainDeadline:  &drain,
					TLSStatus:      "pending",
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := renderStatusTable(&buf, r); err != nil {
		t.Fatalf("renderStatusTable: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"TARGET",
		"HOST",
		"PHASE",
		"ACTIVE",
		"DRAIN DEADLINE",
		"TLS",
		"web",
		"gitea.example.com",
		"live",
		"http://127.0.0.1:34567",
		"ready",
		"dex",
		"dex.example.com",
		"swapping",
		"http://127.0.0.1:34568",
		"2026-04-24T12:30:00Z",
		"pending",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q\n%s", want, out)
		}
	}
}

func TestRenderStatusTable_NilServiceRendersDashes(t *testing.T) {
	r := &appStatusReport{
		Root: &proxypkg.Service{Name: "myapp", Hosts: []string{"app.example.com"}, Phase: proxypkg.PhaseLive},
		Expose: []exposeStatusEntry{
			{Label: "dex", Service: nil},
		},
	}
	var buf bytes.Buffer
	if err := renderStatusTable(&buf, r); err != nil {
		t.Fatal(err)
	}
	// Find the dex row; every unknown column should be "-".
	var dexLine string
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(line, "dex") {
			dexLine = line
			break
		}
	}
	if dexLine == "" {
		t.Fatalf("no dex row in output:\n%s", buf.String())
	}
	if strings.Count(dexLine, "-") < 5 {
		t.Errorf("dex row should have 5 dashes for missing fields, got %q", dexLine)
	}
}

func TestRenderStatusJSON_Shape(t *testing.T) {
	r := &appStatusReport{
		Root: &proxypkg.Service{Name: "gitea", Phase: proxypkg.PhaseLive, Hosts: []string{"gitea.example.com"}},
		Expose: []exposeStatusEntry{
			{Label: "dex", Service: &proxypkg.Service{Name: "gitea-dex", Phase: proxypkg.PhaseLive, Hosts: []string{"dex.example.com"}}},
		},
	}
	var buf bytes.Buffer
	if err := renderStatusJSON(&buf, r); err != nil {
		t.Fatalf("renderStatusJSON: %v", err)
	}
	var parsed struct {
		Root   *proxypkg.Service `json:"root"`
		Expose []struct {
			Label   string            `json:"label"`
			Service *proxypkg.Service `json:"service"`
		} `json:"expose"`
	}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("JSON unmarshal: %v\nraw: %s", err, buf.String())
	}
	if parsed.Root == nil || parsed.Root.Name != "gitea" {
		t.Errorf("root.name = %+v, want gitea", parsed.Root)
	}
	if len(parsed.Expose) != 1 || parsed.Expose[0].Label != "dex" {
		t.Fatalf("expose = %+v, want one dex entry", parsed.Expose)
	}
	if parsed.Expose[0].Service == nil || parsed.Expose[0].Service.Name != "gitea-dex" {
		t.Errorf("expose[0].service = %+v, want gitea-dex", parsed.Expose[0].Service)
	}
}
