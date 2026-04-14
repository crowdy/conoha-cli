package volume

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"

	"github.com/crowdy/conoha-cli/internal/api"
)

// setupTestConfig creates a temporary config directory with a minimal config.yaml
// so that cmdutil.NewClient() can find a profile in tests.
func setupTestConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	cfg := []byte("version: 1\nactive_profile: default\nprofiles:\n  default:\n    tenant_id: test-tenant\n    region: c3j1\n")
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), cfg, 0600); err != nil {
		t.Fatal(err)
	}
	creds := []byte("version: 1\ncredentials:\n  default:\n    username: test\n    password: test\n")
	if err := os.WriteFile(filepath.Join(dir, "credentials.yaml"), creds, 0600); err != nil {
		t.Fatal(err)
	}
	tokens := []byte("version: 1\ntokens:\n  default:\n    token: test-token\n    expires: \"2099-01-01T00:00:00Z\"\n")
	if err := os.WriteFile(filepath.Join(dir, "tokens.yaml"), tokens, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONOHA_CONFIG_DIR", dir)
}

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
	listCalled := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/volumes/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee") && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"volume": map[string]any{"id": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "name": "my-vol", "status": "available", "size": 100},
			})
			return
		}
		if strings.HasSuffix(r.URL.Path, "/volumes/detail") && r.Method == http.MethodGet {
			listCalled = true
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
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
	if listCalled {
		t.Error("ListVolumes should not be called when UUID resolves directly")
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

func TestRenameCmd_Success(t *testing.T) {
	setupTestConfig(t)
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
	// Reset Changed state from previous tests
	renameCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = false })

	cmd := Cmd
	cmd.SetArgs([]string{"rename", "some-vol"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no flags provided")
	}
	if !strings.Contains(err.Error(), "at least one of --name or --description is required") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

func TestCreateCmd_DuplicateNameWarning(t *testing.T) {
	setupTestConfig(t)
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
	cmd.SetArgs([]string{"create", "--name", "my-volume", "--size", "50", "--image", ""})
	err := cmd.Execute()
	// --no-input implies --yes: duplicate-name warning is auto-confirmed and volume is created
	if err != nil {
		t.Fatalf("expected no error in no-input mode (auto-confirm duplicate): %v", err)
	}
	if !createCalled {
		t.Error("CreateVolume should have been called (--no-input auto-confirms duplicate warning)")
	}
}

func TestCreateCmd_NoDuplicate(t *testing.T) {
	setupTestConfig(t)
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
	cmd.SetArgs([]string{"create", "--name", "unique-vol", "--size", "50", "--image", ""})
	err := cmd.Execute()
	if err != nil {
		t.Fatalf("create should succeed with no duplicates: %v", err)
	}
	if !createCalled {
		t.Error("CreateVolume should have been called")
	}
}

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
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCreateCmd_WithImage(t *testing.T) {
	setupTestConfig(t)
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
