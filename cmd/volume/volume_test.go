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
