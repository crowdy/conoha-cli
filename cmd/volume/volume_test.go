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
	cmd.SetArgs([]string{"rename", "some-vol", "--name", "", "--description", ""})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when no flags provided")
	}
	if !strings.Contains(err.Error(), "at least one of --name or --description is required") {
		t.Errorf("expected validation error, got: %v", err)
	}
}

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
