package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/crowdy/conoha-cli/internal/model"
)

func TestListVolumes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/test-tenant/volumes/detail") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"volumes": []map[string]any{
				{"id": "vol-1", "name": "my-volume", "status": "available", "size": 100},
				{"id": "vol-2", "name": "data-volume", "status": "in-use", "size": 200},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewVolumeAPI(newTestClient(ts))
	volumes, err := api.ListVolumes()
	if err != nil {
		t.Fatalf("ListVolumes() error: %v", err)
	}
	if len(volumes) != 2 {
		t.Fatalf("expected 2 volumes, got %d", len(volumes))
	}
	if volumes[0].ID != "vol-1" {
		t.Errorf("expected volume ID 'vol-1', got %q", volumes[0].ID)
	}
	if volumes[0].Name != "my-volume" {
		t.Errorf("expected volume name 'my-volume', got %q", volumes[0].Name)
	}
	if volumes[1].Size != 200 {
		t.Errorf("expected size 200, got %d", volumes[1].Size)
	}
}

func TestGetVolume(t *testing.T) {
	const volumeID = "vol-abc-123"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/test-tenant/volumes/"+volumeID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"volume": map[string]any{
				"id":     volumeID,
				"name":   "boot-volume",
				"status": "available",
				"size":   100,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewVolumeAPI(newTestClient(ts))
	volume, err := api.GetVolume(volumeID)
	if err != nil {
		t.Fatalf("GetVolume() error: %v", err)
	}
	if volume.ID != volumeID {
		t.Errorf("expected volume ID %q, got %q", volumeID, volume.ID)
	}
	if volume.Name != "boot-volume" {
		t.Errorf("expected volume name 'boot-volume', got %q", volume.Name)
	}
}

func TestCreateVolume(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/test-tenant/volumes") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		vol, ok := body["volume"].(map[string]any)
		if !ok {
			t.Errorf("expected 'volume' key in body")
		} else {
			if vol["name"] != "new-volume" {
				t.Errorf("expected name 'new-volume', got %v", vol["name"])
			}
			if vol["size"].(float64) != 100 {
				t.Errorf("expected size 100, got %v", vol["size"])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"volume": map[string]any{
				"id":     "new-vol-id",
				"name":   "new-volume",
				"status": "creating",
				"size":   100,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewVolumeAPI(newTestClient(ts))
	req := &model.VolumeCreateRequest{}
	req.Volume.Name = "new-volume"
	req.Volume.Size = 100
	volume, err := api.CreateVolume(req)
	if err != nil {
		t.Fatalf("CreateVolume() error: %v", err)
	}
	if volume.ID != "new-vol-id" {
		t.Errorf("expected volume ID 'new-vol-id', got %q", volume.ID)
	}
	if volume.Status != "creating" {
		t.Errorf("expected status 'creating', got %q", volume.Status)
	}
}

func TestUpdateVolume(t *testing.T) {
	const volumeID = "upd-vol-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/test-tenant/volumes/"+volumeID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "renamed-volume" {
			t.Errorf("expected name 'renamed-volume', got %v", body["name"])
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewVolumeAPI(newTestClient(ts))
	err := api.UpdateVolume(volumeID, map[string]any{"name": "renamed-volume"})
	if err != nil {
		t.Fatalf("UpdateVolume() error: %v", err)
	}
}

func TestDeleteVolume(t *testing.T) {
	const volumeID = "del-vol-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/test-tenant/volumes/"+volumeID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewVolumeAPI(newTestClient(ts))
	if err := api.DeleteVolume(volumeID); err != nil {
		t.Fatalf("DeleteVolume() error: %v", err)
	}
}

func TestListVolumeTypes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/test-tenant/types") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"volume_types": []map[string]any{
				{"id": "type-1", "name": "c3j1-ds02-boot"},
				{"id": "type-2", "name": "c3j1-ds03-boot"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewVolumeAPI(newTestClient(ts))
	types, err := api.ListVolumeTypes()
	if err != nil {
		t.Fatalf("ListVolumeTypes() error: %v", err)
	}
	if len(types) != 2 {
		t.Fatalf("expected 2 volume types, got %d", len(types))
	}
	if types[0].ID != "type-1" {
		t.Errorf("expected type ID 'type-1', got %q", types[0].ID)
	}
	if types[0].Name != "c3j1-ds02-boot" {
		t.Errorf("expected type name 'c3j1-ds02-boot', got %q", types[0].Name)
	}
}

func TestListBackups(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/test-tenant/backups/detail") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"backups": []map[string]any{
				{"id": "bak-1", "name": "backup-one", "status": "available", "volume_id": "vol-1", "size": 100},
				{"id": "bak-2", "name": "backup-two", "status": "available", "volume_id": "vol-2", "size": 200},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewVolumeAPI(newTestClient(ts))
	backups, err := api.ListBackups()
	if err != nil {
		t.Fatalf("ListBackups() error: %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("expected 2 backups, got %d", len(backups))
	}
	if backups[0].ID != "bak-1" {
		t.Errorf("expected backup ID 'bak-1', got %q", backups[0].ID)
	}
	if backups[0].VolumeID != "vol-1" {
		t.Errorf("expected volume_id 'vol-1', got %q", backups[0].VolumeID)
	}
}

func TestGetBackup(t *testing.T) {
	const backupID = "bak-abc-456"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v3/test-tenant/backups/"+backupID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"backup": map[string]any{
				"id":        backupID,
				"name":      "my-backup",
				"status":    "available",
				"volume_id": "vol-xyz",
				"size":      100,
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewVolumeAPI(newTestClient(ts))
	backup, err := api.GetBackup(backupID)
	if err != nil {
		t.Fatalf("GetBackup() error: %v", err)
	}
	if backup.ID != backupID {
		t.Errorf("expected backup ID %q, got %q", backupID, backup.ID)
	}
	if backup.Name != "my-backup" {
		t.Errorf("expected backup name 'my-backup', got %q", backup.Name)
	}
	if backup.VolumeID != "vol-xyz" {
		t.Errorf("expected volume_id 'vol-xyz', got %q", backup.VolumeID)
	}
}

func TestRestoreBackup(t *testing.T) {
	const backupID = "bak-restore-id"
	const volumeID = "vol-target-id"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		expectedPath := "/v3/test-tenant/backups/" + backupID + "/restore"
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		restore, ok := body["restore"].(map[string]any)
		if !ok {
			t.Errorf("expected 'restore' key in body")
		} else if restore["volume_id"] != volumeID {
			t.Errorf("expected volume_id %q, got %v", volumeID, restore["volume_id"])
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewVolumeAPI(newTestClient(ts))
	if err := api.RestoreBackup(backupID, volumeID); err != nil {
		t.Fatalf("RestoreBackup() error: %v", err)
	}
}
