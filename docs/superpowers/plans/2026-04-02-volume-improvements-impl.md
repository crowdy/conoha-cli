# Volume Improvements (Group A) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add volume rename command (#67), duplicate name warning on create (#68), and --image flag for bootable volumes (#69).

**Architecture:** Single-file extension of `cmd/volume/volume.go`. All required API methods (`UpdateVolume`, `ListVolumes`, `CreateVolume`) and model fields (`VolumeCreateRequest.Volume.ImageRef`) already exist. No API or model layer changes needed. Image resolution uses existing `ImageAPI.ListImages()`.

**Tech Stack:** Go, cobra, httptest

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| Modify | `cmd/volume/volume.go` | Add `renameCmd`, duplicate check in `createCmd`, `--image` flag |
| Create | `cmd/volume/volume_test.go` | Tests for rename, duplicate warning, --image |

---

### Task 1: Volume Rename — Resolve Volume by ID or Name

Add a `findVolume` helper function and the `renameCmd` to `cmd/volume/volume.go`.

**Files:**
- Modify: `cmd/volume/volume.go`
- Create: `cmd/volume/volume_test.go`

- [ ] **Step 1: Write the failing test for findVolume**

Create `cmd/volume/volume_test.go`:

```go
package volume

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crowdy/conoha-cli/internal/api"
)

func newTestVolumeAPI(ts *httptest.Server) *api.VolumeAPI {
	client := &api.Client{HTTP: ts.Client(), Token: "test-token", TenantID: "test-tenant"}
	return api.NewVolumeAPI(client)
}

func volumeListHandler(volumes []map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/volumes/detail") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"volumes": volumes})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

func TestFindVolume_ByUUID(t *testing.T) {
	vols := []map[string]any{
		{"id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "name": "my-vol", "status": "available", "size": 100},
	}
	ts := httptest.NewServer(volumeListHandler(vols))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	volumeAPI := newTestVolumeAPI(ts)
	vol, err := findVolume(volumeAPI, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatalf("findVolume() error: %v", err)
	}
	if vol.ID != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("expected ID match, got %q", vol.ID)
	}
}

func TestFindVolume_ByName(t *testing.T) {
	vols := []map[string]any{
		{"id": "vol-1", "name": "my-vol", "status": "available", "size": 100},
		{"id": "vol-2", "name": "other-vol", "status": "available", "size": 200},
	}
	ts := httptest.NewServer(volumeListHandler(vols))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	volumeAPI := newTestVolumeAPI(ts)
	vol, err := findVolume(volumeAPI, "my-vol")
	if err != nil {
		t.Fatalf("findVolume() error: %v", err)
	}
	if vol.ID != "vol-1" {
		t.Errorf("expected vol-1, got %q", vol.ID)
	}
}

func TestFindVolume_MultipleSameName(t *testing.T) {
	vols := []map[string]any{
		{"id": "vol-1", "name": "dup-vol", "status": "available", "size": 100},
		{"id": "vol-2", "name": "dup-vol", "status": "available", "size": 200},
	}
	ts := httptest.NewServer(volumeListHandler(vols))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	volumeAPI := newTestVolumeAPI(ts)
	_, err := findVolume(volumeAPI, "dup-vol")
	if err == nil {
		t.Fatal("expected error for duplicate names")
	}
	if !strings.Contains(err.Error(), "multiple volumes found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFindVolume_NotFound(t *testing.T) {
	vols := []map[string]any{
		{"id": "vol-1", "name": "my-vol", "status": "available", "size": 100},
	}
	ts := httptest.NewServer(volumeListHandler(vols))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	volumeAPI := newTestVolumeAPI(ts)
	_, err := findVolume(volumeAPI, "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/volume/ -run TestFindVolume -v`
Expected: FAIL — `findVolume` undefined

- [ ] **Step 3: Implement findVolume**

Add to `cmd/volume/volume.go`, after the imports and before the command definitions:

```go
// findVolume resolves a volume by UUID or name.
func findVolume(volumeAPI *api.VolumeAPI, idOrName string) (*model.Volume, error) {
	volumes, err := volumeAPI.ListVolumes()
	if err != nil {
		return nil, err
	}
	// Try exact ID match
	for i := range volumes {
		if volumes[i].ID == idOrName {
			return &volumes[i], nil
		}
	}
	// Try name match
	var matched []*model.Volume
	for i := range volumes {
		if volumes[i].Name == idOrName {
			matched = append(matched, &volumes[i])
		}
	}
	if len(matched) == 1 {
		return matched[0], nil
	}
	if len(matched) > 1 {
		ids := make([]string, len(matched))
		for i, v := range matched {
			ids[i] = v.ID
		}
		return nil, fmt.Errorf("multiple volumes found with name %q (%s), use UUID instead", idOrName, strings.Join(ids, ", "))
	}
	return nil, fmt.Errorf("volume %q not found", idOrName)
}
```

Add `"strings"` to the imports if not already present.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/volume/ -run TestFindVolume -v`
Expected: All 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/volume/volume.go cmd/volume/volume_test.go
git commit -m "Add findVolume helper for volume name/UUID resolution"
```

---

### Task 2: Volume Rename Command

Add the `renameCmd` subcommand with `--name` and `--description` flags.

**Files:**
- Modify: `cmd/volume/volume.go`
- Modify: `cmd/volume/volume_test.go`

- [ ] **Step 1: Write the failing test for rename**

Add to `cmd/volume/volume_test.go`:

```go
func TestRenameCmd_Success(t *testing.T) {
	var updateBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/volumes/detail") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"volumes": []map[string]any{
					{"id": "vol-1", "name": "old-name", "status": "available", "size": 100},
				},
			})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/volumes/vol-1") && r.Method == http.MethodPut {
			json.NewDecoder(r.Body).Decode(&updateBody)
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/volumes/vol-1") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"volume": map[string]any{"id": "vol-1", "name": "new-name", "status": "available", "size": 100},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)
	t.Setenv("CONOHA_TOKEN", "test-token")
	t.Setenv("CONOHA_TENANT_ID", "test-tenant")

	cmd := Cmd
	cmd.SetArgs([]string{"rename", "old-name", "--name", "new-name"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("rename command failed: %v", err)
	}
	if updateBody["name"] != "new-name" {
		t.Errorf("expected name 'new-name' in update body, got %v", updateBody["name"])
	}
}

func TestRenameCmd_NoFlags(t *testing.T) {
	cmd := Cmd
	cmd.SetArgs([]string{"rename", "some-vol"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no flags provided")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/volume/ -run TestRenameCmd -v`
Expected: FAIL — rename subcommand doesn't exist

- [ ] **Step 3: Implement renameCmd**

Add to `cmd/volume/volume.go`:

In the first `init()` function, after `Cmd.AddCommand(backupCmd)`:

```go
	Cmd.AddCommand(renameCmd)

	renameCmd.Flags().String("name", "", "new volume name")
	renameCmd.Flags().String("description", "", "new volume description")
```

Add the command definition after `deleteCmd`:

```go
var renameCmd = &cobra.Command{
	Use:   "rename <id|name>",
	Short: "Rename a volume",
	Args:  cmdutil.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		newName, _ := cmd.Flags().GetString("name")
		newDesc, _ := cmd.Flags().GetString("description")

		if newName == "" && newDesc == "" {
			return fmt.Errorf("at least one of --name or --description is required")
		}

		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		volumeAPI := api.NewVolumeAPI(client)
		vol, err := findVolume(volumeAPI, args[0])
		if err != nil {
			return err
		}

		body := map[string]any{}
		if newName != "" {
			body["name"] = newName
		}
		if newDesc != "" {
			body["description"] = newDesc
		}
		if err := volumeAPI.UpdateVolume(vol.ID, body); err != nil {
			return err
		}

		updated, err := volumeAPI.GetVolume(vol.ID)
		if err != nil {
			return err
		}
		return cmdutil.FormatOutput(cmd, updated)
	},
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/volume/ -run TestRenameCmd -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/volume/volume.go cmd/volume/volume_test.go
git commit -m "Add volume rename command (#67)"
```

---

### Task 3: Volume Create Duplicate Name Warning

Add duplicate name checking at the start of `volume create`.

**Files:**
- Modify: `cmd/volume/volume.go`
- Modify: `cmd/volume/volume_test.go`

- [ ] **Step 1: Write the failing test for duplicate detection**

Add to `cmd/volume/volume_test.go`:

```go
func TestCreateCmd_DuplicateNameWarning(t *testing.T) {
	createCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/volumes/detail") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"volumes": []map[string]any{
					{"id": "existing-vol", "name": "my-volume", "status": "available", "size": 100},
				},
			})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodPost {
			createCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{
				"volume": map[string]any{"id": "new-vol", "name": "my-volume", "status": "creating", "size": 50},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)
	t.Setenv("CONOHA_TOKEN", "test-token")
	t.Setenv("CONOHA_TENANT_ID", "test-tenant")
	t.Setenv("CONOHA_NO_INPUT", "true")

	cmd := Cmd
	cmd.SetArgs([]string{"create", "--name", "my-volume", "--size", "50"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error in no-input mode when duplicate name exists")
	}
	if createCalled {
		t.Error("CreateVolume should not have been called when duplicate detected in no-input mode")
	}
}

func TestCreateCmd_NoDuplicate(t *testing.T) {
	createCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/volumes/detail") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"volumes": []map[string]any{}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodPost {
			createCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{
				"volume": map[string]any{"id": "new-vol", "name": "unique-vol", "status": "creating", "size": 50},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)
	t.Setenv("CONOHA_TOKEN", "test-token")
	t.Setenv("CONOHA_TENANT_ID", "test-tenant")

	cmd := Cmd
	cmd.SetArgs([]string{"create", "--name", "unique-vol", "--size", "50"})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("create should succeed with no duplicates: %v", err)
	}
	if !createCalled {
		t.Error("CreateVolume should have been called")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/volume/ -run TestCreateCmd_Duplicate -v`
Expected: FAIL — duplicate check not implemented (create proceeds despite duplicate)

- [ ] **Step 3: Implement duplicate name check in createCmd**

Modify the `createCmd.RunE` in `cmd/volume/volume.go`. Insert the duplicate check after getting the `name` flag and before building the request. The new `createCmd.RunE` body becomes:

```go
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		name, _ := cmd.Flags().GetString("name")
		size, _ := cmd.Flags().GetInt("size")
		volType, _ := cmd.Flags().GetString("type")
		desc, _ := cmd.Flags().GetString("description")

		// Check for duplicate volume names
		volumeAPI := api.NewVolumeAPI(client)
		volumes, err := volumeAPI.ListVolumes()
		if err != nil {
			return err
		}
		for _, v := range volumes {
			if v.Name == name {
				fmt.Fprintf(os.Stderr, "Warning: volume with name %q already exists (ID: %s)\n", name, v.ID)
				ok, err := prompt.Confirm("Create anyway?")
				if err != nil {
					return err
				}
				if !ok {
					fmt.Fprintln(os.Stderr, "Cancelled.")
					return nil
				}
				break
			}
		}

		req := &model.VolumeCreateRequest{}
		req.Volume.Name = name
		req.Volume.Size = size
		req.Volume.VolumeType = volType
		req.Volume.Description = desc

		vol, err := volumeAPI.CreateVolume(req)
		if err != nil {
			return err
		}

		if wc := cmdutil.GetWaitConfig(cmd, "volume "+name); wc != nil {
			fmt.Fprintf(os.Stderr, "Waiting for volume %s to become available...\n", name)
			if err := waitForVolumeAvailable(volumeAPI, vol.ID, wc); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Volume %s is available.\n", name)
		}

		return cmdutil.FormatOutput(cmd, vol)
	},
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/volume/ -run "TestCreateCmd_Duplicate|TestCreateCmd_NoDuplicate" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/volume/volume.go cmd/volume/volume_test.go
git commit -m "Add duplicate name warning to volume create (#68)"
```

---

### Task 4: Volume Create --image Flag

Add `--image` flag to `volume create` for bootable volume creation.

**Files:**
- Modify: `cmd/volume/volume.go`
- Modify: `cmd/volume/volume_test.go`

- [ ] **Step 1: Write the failing test for --image flag**

Add to `cmd/volume/volume_test.go`:

```go
func newTestImageAPI(ts *httptest.Server) *api.ImageAPI {
	client := &api.Client{HTTP: ts.Client(), Token: "test-token", TenantID: "test-tenant"}
	return api.NewImageAPI(client)
}

func TestResolveImageID_UUID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not make API call for UUID")
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	imageAPI := newTestImageAPI(ts)
	id, err := resolveImageID(imageAPI, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	if err != nil {
		t.Fatalf("resolveImageID() error: %v", err)
	}
	if id != "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Errorf("expected UUID passthrough, got %q", id)
	}
}

func TestResolveImageID_ByName(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/images") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"images": []map[string]any{
					{"id": "img-123", "name": "vmi-ubuntu-24.04-amd64", "status": "active"},
					{"id": "img-456", "name": "vmi-centos-9-amd64", "status": "active"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	imageAPI := newTestImageAPI(ts)
	id, err := resolveImageID(imageAPI, "vmi-ubuntu-24.04-amd64")
	if err != nil {
		t.Fatalf("resolveImageID() error: %v", err)
	}
	if id != "img-123" {
		t.Errorf("expected img-123, got %q", id)
	}
}

func TestResolveImageID_NotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/images") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"images": []map[string]any{}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	imageAPI := newTestImageAPI(ts)
	_, err := resolveImageID(imageAPI, "nonexistent-image")
	if err == nil {
		t.Fatal("expected error for image not found")
	}
	if !strings.Contains(err.Error(), "image not found") {
		t.Errorf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/volume/ -run TestResolveImageID -v`
Expected: FAIL — `resolveImageID` undefined

- [ ] **Step 3: Implement resolveImageID and --image flag**

Add to `cmd/volume/volume.go`:

1. Add the `resolveImageID` function after `findVolume`:

```go
// resolveImageID returns the image UUID. If the input looks like a UUID (36 chars with
// dashes at positions 8, 13, 18, 23), it is returned as-is. Otherwise, ListImages()
// is called to find the image by name.
func resolveImageID(imageAPI *api.ImageAPI, idOrName string) (string, error) {
	if len(idOrName) == 36 && idOrName[8] == '-' && idOrName[13] == '-' && idOrName[18] == '-' && idOrName[23] == '-' {
		return idOrName, nil
	}
	images, err := imageAPI.ListImages()
	if err != nil {
		return "", err
	}
	for _, img := range images {
		if img.Name == idOrName {
			return img.ID, nil
		}
	}
	return "", fmt.Errorf("image not found: %s", idOrName)
}
```

2. Add the `--image` flag in the first `init()` — after the existing `createCmd.Flags().String("description", ...)` line:

```go
	createCmd.Flags().String("image", "", "source image ID or name (creates bootable volume)")
```

3. In `createCmd.RunE`, after building the `req` and before calling `volumeAPI.CreateVolume(req)`, add:

```go
		image, _ := cmd.Flags().GetString("image")
		if image != "" {
			imageAPI := api.NewImageAPI(client)
			imageID, err := resolveImageID(imageAPI, image)
			if err != nil {
				return err
			}
			req.Volume.ImageRef = imageID
		}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/volume/ -run TestResolveImageID -v`
Expected: All 3 tests PASS

- [ ] **Step 5: Write integration test for create with --image**

Add to `cmd/volume/volume_test.go`:

```go
func TestCreateCmd_WithImage(t *testing.T) {
	var createBody map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/volumes/detail") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"volumes": []map[string]any{}})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/images") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"images": []map[string]any{
					{"id": "img-ubuntu-id", "name": "vmi-ubuntu-24.04-amd64", "status": "active"},
				},
			})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/volumes") && r.Method == http.MethodPost {
			json.NewDecoder(r.Body).Decode(&createBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]any{
				"volume": map[string]any{"id": "new-vol", "name": "boot-vol", "status": "creating", "size": 30},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)
	t.Setenv("CONOHA_TOKEN", "test-token")
	t.Setenv("CONOHA_TENANT_ID", "test-tenant")

	cmd := Cmd
	cmd.SetArgs([]string{"create", "--name", "boot-vol", "--size", "30", "--image", "vmi-ubuntu-24.04-amd64"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("create with --image failed: %v", err)
	}
	vol, ok := createBody["volume"].(map[string]any)
	if !ok {
		t.Fatal("expected 'volume' key in create body")
	}
	if vol["imageRef"] != "img-ubuntu-id" {
		t.Errorf("expected imageRef 'img-ubuntu-id', got %v", vol["imageRef"])
	}
}
```

- [ ] **Step 6: Run all tests**

Run: `go test ./cmd/volume/ -v`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/volume/volume.go cmd/volume/volume_test.go
git commit -m "Add --image flag to volume create for bootable volumes (#69)"
```

---

### Task 5: Final Verification

Run full test suite and lint.

**Files:** None (verification only)

- [ ] **Step 1: Run full test suite**

Run: `make test`
Expected: All tests PASS

- [ ] **Step 2: Run linter**

Run: `make lint`
Expected: No lint errors

- [ ] **Step 3: Manual smoke test of help output**

Run: `go run . volume rename --help`
Expected output includes: `--name`, `--description` flags

Run: `go run . volume create --help`
Expected output includes: `--image` flag alongside existing flags

- [ ] **Step 4: Commit any lint fixes if needed**

```bash
git add -u
git commit -m "Fix lint issues in volume commands"
```
