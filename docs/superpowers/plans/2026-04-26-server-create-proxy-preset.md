# Server Create `--for proxy` Preset Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--for proxy` flag to `conoha server create` that fills in `--flavor`, `--image`, and `--security-group` from a named preset spec. Closes #182.

**Architecture:** New `cmd/server/preset.go` defines a small registry (`map[string]presetSpec`) with one entry (`proxy`). Pure helpers `resolvePresetImage` (queries `ListImages`, picks lexicographically newest matching active image) and `validatePresetSecurityGroups` (queries `ListSecurityGroups`, errors if any expected name is missing). `cmd/server/create.go` calls a single resolver `resolvePreset(...)` after flag parsing — explicit flags win, otherwise preset values flow through.

**Tech Stack:** Go 1.22+, cobra, existing `internal/api` (ImageAPI, NetworkAPI), `internal/model` (Image, SecurityGroup). Tests use `httptest.NewServer` + `t.Setenv("CONOHA_ENDPOINT", ts.URL)` per the established pattern in `cmd/server/create_test.go` and `internal/api/network_test.go`.

**Spec:** `docs/superpowers/specs/2026-04-26-server-create-proxy-preset-design.md`

---

## File Structure

| File | Purpose | Lines (approx) |
|---|---|---|
| `cmd/server/preset.go` (NEW) | Preset registry + resolver helpers + image-name match function | ~80 |
| `cmd/server/preset_test.go` (NEW) | Unit + integration tests | ~250 |
| `cmd/server/create.go` (MODIFY) | Register `--for` flag; call `resolvePreset` after flag parsing | ~15 added |

Each file has one responsibility:
- `preset.go` knows nothing about cobra — pure functions taking explicit inputs and APIs.
- `create.go` only knows the integration point — it doesn't know preset internals.
- Tests live next to `preset.go` and exercise both pure and API-facing functions via `httptest`.

---

## Task 1: Scaffolding — preset registry, image matcher, unknown-preset error

**Files:**
- Create: `cmd/server/preset.go`
- Create: `cmd/server/preset_test.go`

- [ ] **Step 1: Write failing tests for `matchDockerUbuntuAmd64` and `knownPresetList`**

Create `cmd/server/preset_test.go`:

```go
package server

import (
	"strings"
	"testing"
)

func TestMatchDockerUbuntuAmd64(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"happy", "vmi-docker-25.10-ubuntu-22.04-amd64", true},
		{"happy_alt_version", "vmi-docker-26.04-ubuntu-24.04-amd64", true},
		{"missing_docker", "vmi-ubuntu-22.04-amd64", false},
		{"non_ubuntu", "vmi-docker-25.10-rocky-9-amd64", false},
		{"non_amd64", "vmi-docker-25.10-ubuntu-22.04-arm64", false},
		{"empty", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := matchDockerUbuntuAmd64(tc.in)
			if got != tc.want {
				t.Errorf("matchDockerUbuntuAmd64(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestKnownPresetList(t *testing.T) {
	got := knownPresetList()
	if !strings.Contains(got, "proxy") {
		t.Errorf("knownPresetList() = %q, want it to contain %q", got, "proxy")
	}
}

func TestPresetRegistry_HasProxy(t *testing.T) {
	spec, ok := presets["proxy"]
	if !ok {
		t.Fatal("presets[\"proxy\"] missing")
	}
	if spec.Flavor != "g2l-t-c3m2" {
		t.Errorf("proxy flavor = %q, want %q", spec.Flavor, "g2l-t-c3m2")
	}
	wantSGs := []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"}
	if len(spec.SecurityGroups) != len(wantSGs) {
		t.Fatalf("proxy SGs len = %d, want %d", len(spec.SecurityGroups), len(wantSGs))
	}
	for i, n := range wantSGs {
		if spec.SecurityGroups[i] != n {
			t.Errorf("proxy SG[%d] = %q, want %q", i, spec.SecurityGroups[i], n)
		}
	}
	if spec.ImageMatch == nil {
		t.Error("proxy ImageMatch is nil")
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL (undefined identifiers)**

Run: `go test ./cmd/server/ -run 'TestMatchDockerUbuntuAmd64|TestKnownPresetList|TestPresetRegistry_HasProxy' -v`
Expected: FAIL with "undefined: matchDockerUbuntuAmd64", "undefined: presets", "undefined: knownPresetList".

- [ ] **Step 3: Create `cmd/server/preset.go` with the minimal scaffolding**

```go
package server

import (
	"sort"
	"strings"
)

// presetSpec describes the values a `--for <name>` preset fills in
// when the corresponding explicit flag is empty.
type presetSpec struct {
	Flavor         string
	SecurityGroups []string
	ImageMatch     func(name string) bool
}

var presets = map[string]presetSpec{
	"proxy": {
		Flavor:         "g2l-t-c3m2",
		SecurityGroups: []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"},
		ImageMatch:     matchDockerUbuntuAmd64,
	},
}

// matchDockerUbuntuAmd64 returns true for ConoHa images named like
// "vmi-docker-<version>-ubuntu-<release>-amd64".
func matchDockerUbuntuAmd64(name string) bool {
	return strings.HasPrefix(name, "vmi-docker-") &&
		strings.Contains(name, "-ubuntu-") &&
		strings.HasSuffix(name, "-amd64")
}

// knownPresetList returns a sorted, comma-joined list of preset names,
// suitable for inclusion in error messages.
func knownPresetList() string {
	names := make([]string, 0, len(presets))
	for n := range presets {
		names = append(names, n)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./cmd/server/ -run 'TestMatchDockerUbuntuAmd64|TestKnownPresetList|TestPresetRegistry_HasProxy' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/preset.go cmd/server/preset_test.go
git commit -m "feat(server/preset): registry skeleton + image name matcher (#182)"
```

---

## Task 2: `validatePresetSecurityGroups` — pre-flight SG check

**Files:**
- Modify: `cmd/server/preset.go` (add function)
- Modify: `cmd/server/preset_test.go` (add tests)

- [ ] **Step 1: Write failing tests**

Append to `cmd/server/preset_test.go` (and add imports as needed: `encoding/json`, `net/http`, `net/http/httptest`, `github.com/crowdy/conoha-cli/internal/api`):

```go
func TestValidatePresetSecurityGroups_AllPresent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v2.0/security-groups") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{
				{"id": "sg-1", "name": "default"},
				{"id": "sg-2", "name": "IPv4v6-SSH"},
				{"id": "sg-3", "name": "IPv4v6-Web"},
				{"id": "sg-4", "name": "IPv4v6-ICMP"},
				{"id": "sg-5", "name": "extra"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	networkAPI := api.NewNetworkAPI(client)

	want := []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"}
	if err := validatePresetSecurityGroups(networkAPI, want); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestValidatePresetSecurityGroups_Missing(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{
				{"id": "sg-1", "name": "default"},
				{"id": "sg-2", "name": "IPv4v6-SSH"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	networkAPI := api.NewNetworkAPI(client)

	want := []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"}
	err := validatePresetSecurityGroups(networkAPI, want)
	if err == nil {
		t.Fatal("expected error for missing SGs, got nil")
	}
	msg := err.Error()
	for _, missing := range []string{"IPv4v6-Web", "IPv4v6-ICMP"} {
		if !strings.Contains(msg, missing) {
			t.Errorf("error %q does not mention missing SG %q", msg, missing)
		}
	}
	for _, present := range []string{"default", "IPv4v6-SSH"} {
		if !strings.Contains(msg, present) {
			t.Errorf("error %q does not list actual SG %q (operator needs the full picture)", msg, present)
		}
	}
}

func TestValidatePresetSecurityGroups_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	networkAPI := api.NewNetworkAPI(client)

	err := validatePresetSecurityGroups(networkAPI, []string{"default"})
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL (undefined: validatePresetSecurityGroups)**

Run: `go test ./cmd/server/ -run TestValidatePresetSecurityGroups -v`
Expected: FAIL with "undefined: validatePresetSecurityGroups".

- [ ] **Step 3: Implement `validatePresetSecurityGroups` in `cmd/server/preset.go`**

Add to imports: `"fmt"`, `"github.com/crowdy/conoha-cli/internal/api"`.

Append to `cmd/server/preset.go`:

```go
// validatePresetSecurityGroups returns nil if every name in want exists in
// the tenant's security-group list. On a missing entry it returns an error
// listing the missing names plus the actual SG list, so the operator can
// self-diagnose without rerunning `conoha server list-sg`.
func validatePresetSecurityGroups(networkAPI *api.NetworkAPI, want []string) error {
	sgs, err := networkAPI.ListSecurityGroups()
	if err != nil {
		return fmt.Errorf("listing security groups: %w", err)
	}
	have := make(map[string]bool, len(sgs))
	names := make([]string, 0, len(sgs))
	for _, sg := range sgs {
		if sg.Name == "" {
			continue
		}
		have[sg.Name] = true
		names = append(names, sg.Name)
	}
	var missing []string
	for _, w := range want {
		if !have[w] {
			missing = append(missing, w)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	sort.Strings(names)
	return fmt.Errorf("preset security groups not found: %s (available: %s)",
		strings.Join(missing, ", "), strings.Join(names, ", "))
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./cmd/server/ -run TestValidatePresetSecurityGroups -v`
Expected: PASS for all three cases.

- [ ] **Step 5: Commit**

```bash
git add cmd/server/preset.go cmd/server/preset_test.go
git commit -m "feat(server/preset): validatePresetSecurityGroups with diagnostic error (#182)"
```

---

## Task 3: `resolvePresetImage` — pick latest matching image

**Files:**
- Modify: `cmd/server/preset.go` (add function)
- Modify: `cmd/server/preset_test.go` (add tests)

- [ ] **Step 1: Write failing tests**

Append to `cmd/server/preset_test.go`:

```go
func TestResolvePresetImage_PicksLatest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v2/images") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-old", "name": "vmi-docker-25.04-ubuntu-22.04-amd64", "status": "active"},
				{"id": "img-new", "name": "vmi-docker-25.10-ubuntu-22.04-amd64", "status": "active"},
				{"id": "img-other", "name": "vmi-docker-25.10-rocky-9-amd64", "status": "active"},
				{"id": "img-deactivated", "name": "vmi-docker-26.04-ubuntu-24.04-amd64", "status": "deactivated"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	imageAPI := api.NewImageAPI(client)

	id, err := resolvePresetImage(imageAPI, matchDockerUbuntuAmd64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "img-new" {
		t.Errorf("resolved image = %q, want %q (newer name should win, deactivated should not)", id, "img-new")
	}
}

func TestResolvePresetImage_NoMatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-1", "name": "vmi-ubuntu-24.04-amd64", "status": "active"},
				{"id": "img-2", "name": "vmi-docker-25.10-rocky-9-amd64", "status": "active"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	imageAPI := api.NewImageAPI(client)

	_, err := resolvePresetImage(imageAPI, matchDockerUbuntuAmd64)
	if err == nil {
		t.Fatal("expected error for no-match list, got nil")
	}
	if !strings.Contains(err.Error(), "no image") {
		t.Errorf("error %q does not mention 'no image'", err.Error())
	}
}

func TestResolvePresetImage_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	imageAPI := api.NewImageAPI(client)

	_, err := resolvePresetImage(imageAPI, matchDockerUbuntuAmd64)
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL (undefined: resolvePresetImage)**

Run: `go test ./cmd/server/ -run TestResolvePresetImage -v`
Expected: FAIL with "undefined: resolvePresetImage".

- [ ] **Step 3: Implement `resolvePresetImage` in `cmd/server/preset.go`**

Append to `cmd/server/preset.go`:

```go
// resolvePresetImage queries ListImages and returns the lexicographically
// newest active image whose name satisfies match. ConoHa rotates these
// images periodically, so resolution at preset-apply time avoids stale
// hardcoded IDs in the CLI binary.
func resolvePresetImage(imageAPI *api.ImageAPI, match func(string) bool) (string, error) {
	images, err := imageAPI.ListImages()
	if err != nil {
		return "", fmt.Errorf("listing images: %w", err)
	}
	var matched []string // names
	idByName := make(map[string]string)
	for _, img := range images {
		if img.Status != "active" {
			continue
		}
		if !match(img.Name) {
			continue
		}
		matched = append(matched, img.Name)
		idByName[img.Name] = img.ID
	}
	if len(matched) == 0 {
		return "", fmt.Errorf("no image matched preset criteria (try `conoha image list` to see what is available)")
	}
	sort.Sort(sort.Reverse(sort.StringSlice(matched)))
	return idByName[matched[0]], nil
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./cmd/server/ -run TestResolvePresetImage -v`
Expected: PASS for all three cases.

- [ ] **Step 5: Run the full server-package tests to confirm no regressions**

Run: `go test ./cmd/server/ -v`
Expected: all existing + new tests PASS.

- [ ] **Step 6: Commit**

```bash
git add cmd/server/preset.go cmd/server/preset_test.go
git commit -m "feat(server/preset): resolvePresetImage queries latest matching image (#182)"
```

---

## Task 4: `resolvePreset` orchestrator + wire into `create.go`

**Files:**
- Modify: `cmd/server/preset.go` (add `resolvePreset` orchestrator)
- Modify: `cmd/server/preset_test.go` (add orchestrator tests)
- Modify: `cmd/server/create.go` (register `--for` flag, call `resolvePreset`)

- [ ] **Step 1: Write failing tests for the orchestrator**

Append to `cmd/server/preset_test.go`:

```go
func TestResolvePreset_AllDefaults(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/images", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-latest", "name": "vmi-docker-25.10-ubuntu-22.04-amd64", "status": "active"},
			},
		})
	})
	mux.HandleFunc("GET /v2.0/security-groups", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{
				{"id": "sg-1", "name": "default"},
				{"id": "sg-2", "name": "IPv4v6-SSH"},
				{"id": "sg-3", "name": "IPv4v6-Web"},
				{"id": "sg-4", "name": "IPv4v6-ICMP"},
			},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	imageAPI := api.NewImageAPI(client)
	networkAPI := api.NewNetworkAPI(client)

	flavor, image, sgs, err := resolvePreset("proxy", "", "", nil, imageAPI, networkAPI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != "g2l-t-c3m2" {
		t.Errorf("flavor = %q, want %q", flavor, "g2l-t-c3m2")
	}
	if image != "img-latest" {
		t.Errorf("image = %q, want %q", image, "img-latest")
	}
	wantSGs := []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"}
	if len(sgs) != len(wantSGs) {
		t.Fatalf("sgs len = %d, want %d", len(sgs), len(wantSGs))
	}
	for i, n := range wantSGs {
		if sgs[i] != n {
			t.Errorf("sgs[%d] = %q, want %q", i, sgs[i], n)
		}
	}
}

func TestResolvePreset_ExplicitFlavorWins(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/images", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-latest", "name": "vmi-docker-25.10-ubuntu-22.04-amd64", "status": "active"},
			},
		})
	})
	mux.HandleFunc("GET /v2.0/security-groups", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{
				{"id": "sg-1", "name": "default"},
				{"id": "sg-2", "name": "IPv4v6-SSH"},
				{"id": "sg-3", "name": "IPv4v6-Web"},
				{"id": "sg-4", "name": "IPv4v6-ICMP"},
			},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	flavor, image, _, err := resolvePreset("proxy", "g2l-t-c4m4", "", nil,
		api.NewImageAPI(client), api.NewNetworkAPI(client))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != "g2l-t-c4m4" {
		t.Errorf("flavor = %q, want explicit %q", flavor, "g2l-t-c4m4")
	}
	if image != "img-latest" {
		t.Errorf("image = %q, want preset-resolved %q", image, "img-latest")
	}
}

func TestResolvePreset_ExplicitSGReplaces(t *testing.T) {
	// When user passes their own --security-group, the preset's SG list is
	// REPLACED, not appended. Verify the API is not even called for SG
	// validation (would be wasted work and could give a misleading error).
	mux := http.NewServeMux()
	sgCalls := 0
	mux.HandleFunc("GET /v2/images", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-latest", "name": "vmi-docker-25.10-ubuntu-22.04-amd64", "status": "active"},
			},
		})
	})
	mux.HandleFunc("GET /v2.0/security-groups", func(w http.ResponseWriter, r *http.Request) {
		sgCalls++
		_ = json.NewEncoder(w).Encode(map[string]any{
			"security_groups": []map[string]any{},
		})
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	_, _, sgs, err := resolvePreset("proxy", "", "", []string{"custom"},
		api.NewImageAPI(client), api.NewNetworkAPI(client))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sgs) != 1 || sgs[0] != "custom" {
		t.Errorf("sgs = %v, want [custom]", sgs)
	}
	if sgCalls != 0 {
		t.Errorf("ListSecurityGroups was called %d times; should not be called when user passed explicit SGs", sgCalls)
	}
}

func TestResolvePreset_UnknownName(t *testing.T) {
	client := &api.Client{Token: "fake-token", TenantID: "tenant-1"}
	_, _, _, err := resolvePreset("nonexistent", "", "", nil,
		api.NewImageAPI(client), api.NewNetworkAPI(client))
	if err == nil {
		t.Fatal("expected error for unknown preset, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "nonexistent") {
		t.Errorf("error %q does not mention bad name", msg)
	}
	if !strings.Contains(msg, "proxy") {
		t.Errorf("error %q does not list known preset 'proxy'", msg)
	}
}

func TestResolvePreset_EmptyName_NoOp(t *testing.T) {
	// Empty preset name means user did not pass --for; resolvePreset must
	// return its inputs unchanged and make zero API calls.
	client := &api.Client{Token: "fake-token", TenantID: "tenant-1"}
	flavor, image, sgs, err := resolvePreset("", "fl-1", "im-1", []string{"sg-1"},
		api.NewImageAPI(client), api.NewNetworkAPI(client))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flavor != "fl-1" || image != "im-1" || len(sgs) != 1 || sgs[0] != "sg-1" {
		t.Errorf("expected pass-through, got flavor=%q image=%q sgs=%v", flavor, image, sgs)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL (undefined: resolvePreset)**

Run: `go test ./cmd/server/ -run TestResolvePreset -v`
Expected: FAIL with "undefined: resolvePreset".

- [ ] **Step 3: Implement `resolvePreset` in `cmd/server/preset.go`**

Append to `cmd/server/preset.go`:

```go
// resolvePreset returns flavor/image/sgNames after applying the named
// preset. If forName is empty, inputs pass through unchanged (no API
// calls). Explicit non-zero inputs always win over preset values; for
// security groups, a non-empty user-supplied list REPLACES the preset's
// list rather than appending to it (see spec § Override semantics).
func resolvePreset(
	forName string,
	flavorIn, imageIn string,
	sgIn []string,
	imageAPI *api.ImageAPI,
	networkAPI *api.NetworkAPI,
) (flavor, image string, sgs []string, err error) {
	flavor, image, sgs = flavorIn, imageIn, sgIn
	if forName == "" {
		return
	}
	spec, ok := presets[forName]
	if !ok {
		err = fmt.Errorf("unknown preset %q (known: %s)", forName, knownPresetList())
		return
	}
	if flavor == "" {
		flavor = spec.Flavor
	}
	if image == "" {
		image, err = resolvePresetImage(imageAPI, spec.ImageMatch)
		if err != nil {
			return
		}
	}
	if len(sgs) == 0 {
		if vErr := validatePresetSecurityGroups(networkAPI, spec.SecurityGroups); vErr != nil {
			err = vErr
			return
		}
		sgs = spec.SecurityGroups
	}
	return
}
```

- [ ] **Step 4: Run tests — expect PASS**

Run: `go test ./cmd/server/ -run TestResolvePreset -v`
Expected: PASS for all five cases.

- [ ] **Step 5: Wire `--for` flag into `cmd/server/create.go`**

Edit `cmd/server/create.go`. In `init()`, append after the existing `StringArray("security-group", ...)` line (currently around line 45):

```go
	createCmd.Flags().String("for", "", "preset that fills in flavor, image, and security groups (e.g. \"proxy\")")
```

Then in the `RunE` body, locate the block (currently around lines 63–68) where `flavorID`, `imageID`, `keyName`, `adminPass` are read from flags. Right after that block — and BEFORE the `// Resolve user_data` comment — insert:

```go
	forName, _ := cmd.Flags().GetString("for")
	if forName != "" {
		imageAPI := api.NewImageAPI(client)
		networkAPI := api.NewNetworkAPI(client)
		sgFromFlag, _ := cmd.Flags().GetStringArray("security-group")
		var presetSGs []string
		flavorID, imageID, presetSGs, err = resolvePreset(forName, flavorID, imageID, sgFromFlag, imageAPI, networkAPI)
		if err != nil {
			return err
		}
		// Push preset-resolved SGs back into the flag so the existing
		// `cmd.Flags().GetStringArray("security-group")` call below sees them.
		if len(sgFromFlag) == 0 && len(presetSGs) > 0 {
			for _, n := range presetSGs {
				_ = cmd.Flags().Set("security-group", n)
			}
		}
	}
```

**Important:** verify the imports in `create.go` already include `"github.com/crowdy/conoha-cli/internal/api"` (they do — line 13). No new imports needed.

- [ ] **Step 6: Add an integration-style test for the wired-up `create.go`**

Append to `cmd/server/preset_test.go` (top imports may need `os`, `bytes`, `path/filepath` — add only if missing):

```go
func TestCreateCmd_ForFlagRegistered(t *testing.T) {
	f := createCmd.Flags().Lookup("for")
	if f == nil {
		t.Fatal("create command missing --for flag")
	}
	if f.DefValue != "" {
		t.Errorf("--for default = %q, want empty", f.DefValue)
	}
}
```

- [ ] **Step 7: Run tests — expect PASS**

Run: `go test ./cmd/server/ -v`
Expected: all tests PASS, no regressions.

- [ ] **Step 8: Run lint + full test suite to catch package-wide issues**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: no errors. The full suite should still pass — preset.go is additive, create.go change only fires when `--for` is set.

- [ ] **Step 9: Commit**

```bash
git add cmd/server/preset.go cmd/server/preset_test.go cmd/server/create.go
git commit -m "feat(server): --for proxy preset wires into server create (#182)

Adds --for <preset> flag that fills in --flavor, --image, and
--security-group from a named preset spec. Single preset shipped:
proxy. Explicit flags always win; explicit --security-group replaces
(does not append) the preset list.

Closes #182."
```

---

## Task 5: PR

**Files:** none (workflow step)

- [ ] **Step 1: Push branch and open PR**

```bash
git push -u origin HEAD
gh pr create --title "feat(server): --for proxy preset (closes #182)" --body "$(cat <<'EOF'
## Summary

- Adds `--for proxy` to `conoha server create` that fills in `--flavor` (`g2l-t-c3m2`), `--image` (latest active `vmi-docker-*-ubuntu-*-amd64`), and `--security-group` (`default`, `IPv4v6-SSH`, `IPv4v6-Web`, `IPv4v6-ICMP`).
- Explicit flags win; explicit `--security-group` *replaces* the preset list rather than appending.
- Pre-flight `ListSecurityGroups` validates all four expected names exist; missing-name error lists what is actually present.
- Image resolved at preset-apply time (`ListImages`, prefix+suffix match, lexicographic newest) so the binary does not carry a stale image ID.
- Single preset shipped (`proxy`); `--for unknown` errors with the known list, leaving room for future presets without surface redesign.

## Design

Spec: `docs/superpowers/specs/2026-04-26-server-create-proxy-preset-design.md`

## Test plan

- [x] `go test ./cmd/server/ -v` — new + existing pass
- [x] `go build ./... && go vet ./...` clean
- [ ] Smoke on c3j1: `conoha server create --no-input --yes --wait --name preset-smoke --key-name <key> --for proxy` → server reaches ACTIVE with the four expected SGs and the latest docker image.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

- [ ] **Step 2: Verify CI green**

Run: `gh pr checks $(gh pr view --json number -q .number)`
Expected: build, lint, test, e2e all SUCCESS.

---

## Self-Review Checklist (run after writing — not part of execution)

**Spec coverage:**
- ✓ Q1 (region SG): hardcoded list + validation — Task 2
- ✓ Q2 (image freshness): `ListImages` + match + sort — Task 3
- ✓ Q3 (flavor minimum): `g2l-t-c3m2` literal in registry — Task 1
- ✓ CLI shape `--for proxy`: flag registered + orchestrator — Task 4
- ✓ Override semantics (explicit wins, SG replaces): unit-tested — Task 4
- ✓ Failure modes (unknown preset, missing SG, no image, API error): tested — Tasks 1–4
- ✓ Out-of-scope items (region map, multi-preset stacking, YAML config): not implemented — correct

**Placeholder scan:** no TBD/TODO/"add error handling"/"similar to" — every step shows the actual code or command.

**Type consistency:** `presetSpec` / `presets` / `resolvePresetImage` / `validatePresetSecurityGroups` / `resolvePreset` / `matchDockerUbuntuAmd64` / `knownPresetList` — names identical across all task references.
