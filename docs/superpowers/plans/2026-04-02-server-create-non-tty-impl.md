# Server Create Non-TTY Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Allow `conoha server create` to work in non-TTY environments by auto-creating boot volumes with sensible defaults when `--volume` is not specified.

**Architecture:** Modify `resolveBootVolume()` to detect non-interactive context and auto-create a boot volume with defaults (name from server name, 100GB size). Modify `createBootVolume()` to accept pre-filled parameters that skip prompts.

**Tech Stack:** Go, cobra, golang.org/x/term

---

### Task 1: Add `isInteractive` helper

**Files:**
- Create: `internal/prompt/interactive.go`
- Create: `internal/prompt/interactive_test.go`

- [ ] **Step 1: Write the failing test**

In `internal/prompt/interactive_test.go`:

```go
package prompt

import (
	"os"
	"testing"
)

func TestIsInteractive_NoInput(t *testing.T) {
	t.Setenv("CONOHA_NO_INPUT", "1")
	if IsInteractive() {
		t.Error("expected non-interactive when CONOHA_NO_INPUT=1")
	}
}

func TestIsInteractive_NoTTY(t *testing.T) {
	// In test environment, stdin is not a TTY
	if IsInteractive() {
		t.Error("expected non-interactive in test environment (no TTY)")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/prompt/ -run TestIsInteractive -v`
Expected: FAIL — `IsInteractive` not defined

- [ ] **Step 3: Write implementation**

In `internal/prompt/interactive.go`:

```go
package prompt

import (
	"os"

	"golang.org/x/term"

	"github.com/crowdy/conoha-cli/internal/config"
)

// IsInteractive returns true if the current session supports interactive prompts.
// Returns false if stdin is not a TTY or if CONOHA_NO_INPUT is set.
func IsInteractive() bool {
	if config.IsNoInput() {
		return false
	}
	return term.IsTerminal(int(os.Stdin.Fd()))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/prompt/ -run TestIsInteractive -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/prompt/interactive.go internal/prompt/interactive_test.go
git commit -m "Add IsInteractive helper to prompt package"
```

---

### Task 2: Refactor `createBootVolume` to accept default parameters

**Files:**
- Modify: `cmd/server/create.go:441-487` (`createBootVolume` function)

- [ ] **Step 1: Write the failing test**

In `cmd/server/create_test.go`, add a test for the auto-create path using `httptest`:

```go
func TestCreateBootVolume_WithDefaults(t *testing.T) {
	mux := http.NewServeMux()

	var gotReq model.VolumeCreateRequest
	mux.HandleFunc("POST /v2/{tenant}/volumes", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq)
		resp := `{"volume":{"id":"vol-new-123","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(resp))
	})
	mux.HandleFunc("GET /v2/{tenant}/volumes/vol-new-123", func(w http.ResponseWriter, r *http.Request) {
		resp := `{"volume":{"id":"vol-new-123","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := api.NewClient(ts.URL, "fake-token", "tenant-1")
	volumeAPI := api.NewVolumeAPI(client)
	flavor := &model.Flavor{Name: "g2l-t-c2m1", RAM: 1024}

	volID, created, err := createBootVolumeWithDefaults(volumeAPI, flavor, "img-abc", "myserver")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected created=true")
	}
	if volID != "vol-new-123" {
		t.Errorf("volID = %q, want %q", volID, "vol-new-123")
	}
	if gotReq.Volume.Name != "myserver-boot" {
		t.Errorf("volume name = %q, want %q", gotReq.Volume.Name, "myserver-boot")
	}
	if gotReq.Volume.Size != 100 {
		t.Errorf("volume size = %d, want 100", gotReq.Volume.Size)
	}
	if gotReq.Volume.VolumeType != "c3j1-ds02-boot" {
		t.Errorf("volume type = %q, want %q", gotReq.Volume.VolumeType, "c3j1-ds02-boot")
	}
	if gotReq.Volume.ImageRef != "img-abc" {
		t.Errorf("image ref = %q, want %q", gotReq.Volume.ImageRef, "img-abc")
	}
}
```

Add necessary imports at top of test file:

```go
import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/crowdy/conoha-cli/internal/api"
)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/server/ -run TestCreateBootVolume_WithDefaults -v`
Expected: FAIL — `createBootVolumeWithDefaults` not defined

- [ ] **Step 3: Write implementation**

Add `createBootVolumeWithDefaults` function in `cmd/server/create.go`:

```go
// createBootVolumeWithDefaults creates a boot volume using sensible defaults (no prompts).
// Used in non-interactive environments when --volume is not specified.
func createBootVolumeWithDefaults(volumeAPI *api.VolumeAPI, flavor *model.Flavor, imageID, serverName string) (string, bool, error) {
	volName := serverName + "-boot"
	sizeGB := maxBootVolumeGB(flavor)

	fmt.Fprintf(os.Stderr, "Creating boot volume %q (%dGB, %s)...\n", volName, sizeGB, defaultBootVolumeType)
	req := &model.VolumeCreateRequest{}
	req.Volume.Size = sizeGB
	req.Volume.Name = volName
	req.Volume.VolumeType = defaultBootVolumeType
	req.Volume.ImageRef = imageID
	vol, err := volumeAPI.CreateVolume(req)
	if err != nil {
		return "", false, fmt.Errorf("creating boot volume: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Waiting for volume %s to become available...\n", vol.ID)
	if err := waitForVolumeAvailable(volumeAPI, vol.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: boot volume %s was created but may not be ready.\n", vol.ID)
		fmt.Fprintf(os.Stderr, "You can delete it with: conoha volume delete %s\n", vol.ID)
		return "", true, err
	}
	fmt.Fprintf(os.Stderr, "Volume %s is ready.\n", vol.ID)
	return vol.ID, true, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/server/ -run TestCreateBootVolume_WithDefaults -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/server/create.go cmd/server/create_test.go
git commit -m "Add createBootVolumeWithDefaults for non-interactive boot volume creation"
```

---

### Task 3: Update `resolveBootVolume` to use auto-create in non-interactive mode

**Files:**
- Modify: `cmd/server/create.go:403-439` (`resolveBootVolume` function)
- Modify: `cmd/server/create.go:139` (call site)

- [ ] **Step 1: Write the failing test**

In `cmd/server/create_test.go`:

```go
func TestResolveBootVolume_NonInteractive_AutoCreates(t *testing.T) {
	t.Setenv("CONOHA_NO_INPUT", "1")

	mux := http.NewServeMux()
	var gotReq model.VolumeCreateRequest
	mux.HandleFunc("POST /v2/{tenant}/volumes", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotReq)
		resp := `{"volume":{"id":"vol-auto-1","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(resp))
	})
	mux.HandleFunc("GET /v2/{tenant}/volumes/vol-auto-1", func(w http.ResponseWriter, r *http.Request) {
		resp := `{"volume":{"id":"vol-auto-1","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := api.NewClient(ts.URL, "fake-token", "tenant-1")
	volumeAPI := api.NewVolumeAPI(client)
	flavor := &model.Flavor{Name: "g2l-t-c2m1", RAM: 1024}

	volID, created, err := resolveBootVolume(volumeAPI, flavor, "img-abc", "", "testserver")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected created=true for auto-created volume")
	}
	if volID != "vol-auto-1" {
		t.Errorf("volID = %q, want %q", volID, "vol-auto-1")
	}
	if gotReq.Volume.Name != "testserver-boot" {
		t.Errorf("auto-created volume name = %q, want %q", gotReq.Volume.Name, "testserver-boot")
	}
}

func TestResolveBootVolume_DedicatedFlavor_NoVolume(t *testing.T) {
	flavor := &model.Flavor{Name: "g2d-t-c2m1", RAM: 1024}
	volID, created, err := resolveBootVolume(nil, flavor, "img-abc", "", "testserver")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if volID != "" || created {
		t.Errorf("dedicated flavor should not create volume, got volID=%q created=%v", volID, created)
	}
}

func TestResolveBootVolume_ExistingVolume(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v2/{tenant}/volumes/vol-existing", func(w http.ResponseWriter, r *http.Request) {
		resp := `{"volume":{"id":"vol-existing","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	client := api.NewClient(ts.URL, "fake-token", "tenant-1")
	volumeAPI := api.NewVolumeAPI(client)
	flavor := &model.Flavor{Name: "g2l-t-c2m1", RAM: 1024}

	volID, created, err := resolveBootVolume(volumeAPI, flavor, "img-abc", "vol-existing", "testserver")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created {
		t.Error("expected created=false for existing volume")
	}
	if volID != "vol-existing" {
		t.Errorf("volID = %q, want %q", volID, "vol-existing")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/server/ -run TestResolveBootVolume -v`
Expected: FAIL — `resolveBootVolume` signature mismatch (missing `serverName` parameter)

- [ ] **Step 3: Update `resolveBootVolume` signature and logic**

Update the function in `cmd/server/create.go`:

```go
// resolveBootVolume determines the boot volume for server creation.
// Returns volumeID (empty if not needed), whether a new volume was created, and any error.
func resolveBootVolume(volumeAPI *api.VolumeAPI, flavor *model.Flavor, imageID string, flagVolumeID string, serverName string) (string, bool, error) {
	if !flavorNeedsVolume(flavor.Name) {
		return "", false, nil
	}

	// --volume flag specified
	if flagVolumeID != "" {
		vol, err := volumeAPI.GetVolume(flagVolumeID)
		if err != nil {
			return "", false, fmt.Errorf("volume %q not found: %w", flagVolumeID, err)
		}
		if vol.Status != "available" {
			return "", false, fmt.Errorf("volume %s is not available (status: %s)", flagVolumeID, vol.Status)
		}
		if maxGB := maxBootVolumeGB(flavor); vol.Size > maxGB {
			return "", false, fmt.Errorf("volume size %dGB exceeds maximum %dGB for flavor %s", vol.Size, maxGB, flavor.Name)
		}
		return flagVolumeID, false, nil
	}

	// Non-interactive: auto-create with defaults
	if !prompt.IsInteractive() {
		return createBootVolumeWithDefaults(volumeAPI, flavor, imageID, serverName)
	}

	// Interactive selection
	items := []prompt.SelectItem{
		{Label: "Create new volume", Value: "new"},
		{Label: "Use existing volume", Value: "existing"},
	}
	choice, err := prompt.Select("Boot volume", items)
	if err != nil {
		return "", false, err
	}

	if choice == "new" {
		return createBootVolume(volumeAPI, flavor, imageID)
	}
	return selectExistingVolume(volumeAPI)
}
```

Update the call site (line 139):

```go
volumeID, created, err := resolveBootVolume(volumeAPI, flavor, imageID, flagVolumeID, name)
```

Add import for `prompt` package if not already present (it is — used in the file already).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/server/ -run TestResolveBootVolume -v`
Expected: PASS (all 3 new tests)

- [ ] **Step 5: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/server/create.go cmd/server/create_test.go
git commit -m "Auto-create boot volume in non-interactive mode for server create (#66)"
```

---

### Task 4: Verify and lint

**Files:** None (verification only)

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -v`
Expected: All PASS

- [ ] **Step 2: Run linter**

Run: `golangci-lint run ./...`
Expected: No issues

- [ ] **Step 3: Manual verification**

Verify the fix by tracing the non-interactive path:
1. `--volume` not specified + non-TTY → `createBootVolumeWithDefaults` called
2. `--volume` specified → existing validation path (unchanged)
3. TTY without `--volume` → interactive prompt (unchanged)
4. Dedicated flavor → no volume (unchanged)
