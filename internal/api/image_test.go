package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageList(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2/images") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"images": []map[string]any{
				{"id": "img-1", "name": "ubuntu-22.04", "status": "active", "disk_format": "qcow2", "container_format": "bare"},
				{"id": "img-2", "name": "centos-9", "status": "active", "disk_format": "raw", "container_format": "bare"},
			},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewImageAPI(newTestClient(ts))
	images, err := api.ListImages()
	if err != nil {
		t.Fatalf("ListImages() error: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("expected 2 images, got %d", len(images))
	}
	if images[0].ID != "img-1" {
		t.Errorf("expected image ID 'img-1', got %q", images[0].ID)
	}
	if images[0].Name != "ubuntu-22.04" {
		t.Errorf("expected image name 'ubuntu-22.04', got %q", images[0].Name)
	}
	if images[1].DiskFormat != "raw" {
		t.Errorf("expected disk_format 'raw', got %q", images[1].DiskFormat)
	}
}

func TestImageGet(t *testing.T) {
	const imageID = "img-abc-123"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2/images/"+imageID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":               imageID,
			"name":             "my-image",
			"status":           "active",
			"disk_format":      "qcow2",
			"container_format": "bare",
			"visibility":       "private",
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewImageAPI(newTestClient(ts))
	img, err := api.GetImage(imageID)
	if err != nil {
		t.Fatalf("GetImage() error: %v", err)
	}
	if img.ID != imageID {
		t.Errorf("expected image ID %q, got %q", imageID, img.ID)
	}
	if img.Name != "my-image" {
		t.Errorf("expected image name 'my-image', got %q", img.Name)
	}
	if img.Visibility != "private" {
		t.Errorf("expected visibility 'private', got %q", img.Visibility)
	}
}

func TestImageDelete(t *testing.T) {
	const imageID = "img-del-456"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2/images/"+imageID) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewImageAPI(newTestClient(ts))
	if err := api.DeleteImage(imageID); err != nil {
		t.Fatalf("DeleteImage() error: %v", err)
	}
}

func TestImageCreate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v2/images") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["name"] != "custom-image" {
			t.Errorf("expected name 'custom-image', got %v", body["name"])
		}
		if body["disk_format"] != "qcow2" {
			t.Errorf("expected disk_format 'qcow2', got %v", body["disk_format"])
		}
		if body["container_format"] != "bare" {
			t.Errorf("expected container_format 'bare', got %v", body["container_format"])
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id":               "new-img-id",
			"name":             "custom-image",
			"status":           "queued",
			"disk_format":      "qcow2",
			"container_format": "bare",
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewImageAPI(newTestClient(ts))
	img, err := api.CreateImage("custom-image", "qcow2", "bare")
	if err != nil {
		t.Fatalf("CreateImage() error: %v", err)
	}
	if img.ID != "new-img-id" {
		t.Errorf("expected image ID 'new-img-id', got %q", img.ID)
	}
	if img.Name != "custom-image" {
		t.Errorf("expected image name 'custom-image', got %q", img.Name)
	}
	if img.Status != "queued" {
		t.Errorf("expected status 'queued', got %q", img.Status)
	}
}
