package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetAccountInfo(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/AUTH_test-tenant") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("X-Account-Container-Count", "5")
		w.Header().Set("X-Account-Object-Count", "42")
		w.Header().Set("X-Account-Bytes-Used", "1073741824")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	info, err := api.GetAccountInfo()
	if err != nil {
		t.Fatalf("GetAccountInfo() error: %v", err)
	}
	if info.ContainerCount != 5 {
		t.Errorf("expected ContainerCount 5, got %d", info.ContainerCount)
	}
	if info.ObjectCount != 42 {
		t.Errorf("expected ObjectCount 42, got %d", info.ObjectCount)
	}
	if info.BytesUsed != 1073741824 {
		t.Errorf("expected BytesUsed 1073741824, got %d", info.BytesUsed)
	}
}

func TestListContainers(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/AUTH_test-tenant") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.RawQuery != "format=json" {
			t.Errorf("expected query 'format=json', got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "container-a", "count": 10, "bytes": 512},
			{"name": "container-b", "count": 5, "bytes": 1024},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	containers, err := api.ListContainers()
	if err != nil {
		t.Fatalf("ListContainers() error: %v", err)
	}
	if len(containers) != 2 {
		t.Fatalf("expected 2 containers, got %d", len(containers))
	}
	if containers[0].Name != "container-a" {
		t.Errorf("expected name 'container-a', got %q", containers[0].Name)
	}
	if containers[0].Count != 10 {
		t.Errorf("expected count 10, got %d", containers[0].Count)
	}
	if containers[1].Bytes != 1024 {
		t.Errorf("expected bytes 1024, got %d", containers[1].Bytes)
	}
}

func TestCreateContainer(t *testing.T) {
	const containerName = "my-new-container"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/AUTH_test-tenant/"+containerName) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	if err := api.CreateContainer(containerName); err != nil {
		t.Fatalf("CreateContainer() error: %v", err)
	}
}

func TestDeleteContainer(t *testing.T) {
	const containerName = "old-container"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/AUTH_test-tenant/"+containerName) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	if err := api.DeleteContainer(containerName); err != nil {
		t.Fatalf("DeleteContainer() error: %v", err)
	}
}

func TestListObjects(t *testing.T) {
	const containerName = "my-container"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/AUTH_test-tenant/"+containerName) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.RawQuery != "format=json" {
			t.Errorf("expected query 'format=json', got %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "file1.txt", "content_type": "text/plain", "bytes": 100, "last_modified": "2026-01-01T00:00:00Z", "hash": "abc123"},
			{"name": "image.png", "content_type": "image/png", "bytes": 2048, "last_modified": "2026-01-02T00:00:00Z", "hash": "def456"},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	objects, err := api.ListObjects(containerName)
	if err != nil {
		t.Fatalf("ListObjects() error: %v", err)
	}
	if len(objects) != 2 {
		t.Fatalf("expected 2 objects, got %d", len(objects))
	}
	if objects[0].Name != "file1.txt" {
		t.Errorf("expected name 'file1.txt', got %q", objects[0].Name)
	}
	if objects[1].Bytes != 2048 {
		t.Errorf("expected bytes 2048, got %d", objects[1].Bytes)
	}
}

func TestListObjectsWithPrefix(t *testing.T) {
	const containerName = "my-container"
	const prefix = "logs/"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/AUTH_test-tenant/"+containerName) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("format") != "json" {
			t.Errorf("expected format=json, got %q", q.Get("format"))
		}
		if q.Get("prefix") != prefix {
			t.Errorf("expected prefix=%q, got %q", prefix, q.Get("prefix"))
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"name": "logs/app.log", "content_type": "text/plain", "bytes": 512, "last_modified": "2026-01-01T00:00:00Z", "hash": "aaa"},
		})
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	objects, err := api.ListObjectsWithPrefix(containerName, prefix)
	if err != nil {
		t.Fatalf("ListObjectsWithPrefix() error: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("expected 1 object, got %d", len(objects))
	}
	if objects[0].Name != "logs/app.log" {
		t.Errorf("expected name 'logs/app.log', got %q", objects[0].Name)
	}
}

func TestUploadObject(t *testing.T) {
	const containerName = "uploads"
	const objectName = "hello.txt"
	const fileContent = "hello world"

	// Create a temp file to upload
	dir := t.TempDir()
	localPath := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(localPath, []byte(fileContent), 0600); err != nil {
		t.Fatalf("creating temp file: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		expectedPath := "/v1/AUTH_test-tenant/" + containerName + "/" + objectName
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	if err := api.UploadObject(containerName, objectName, localPath); err != nil {
		t.Fatalf("UploadObject() error: %v", err)
	}
}

func TestDownloadObject(t *testing.T) {
	const containerName = "downloads"
	const objectName = "data.txt"
	const fileContent = "downloaded content"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		expectedPath := "/v1/AUTH_test-tenant/" + containerName + "/" + objectName
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fileContent)) //nolint:errcheck
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	dir := t.TempDir()
	localPath := filepath.Join(dir, "data.txt")

	api := NewObjectStorageAPI(newTestClient(ts))
	if err := api.DownloadObject(containerName, objectName, localPath); err != nil {
		t.Fatalf("DownloadObject() error: %v", err)
	}

	got, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("reading downloaded file: %v", err)
	}
	if string(got) != fileContent {
		t.Errorf("expected file content %q, got %q", fileContent, string(got))
	}
}

func TestDeleteObject(t *testing.T) {
	const containerName = "my-container"
	const objectName = "old-file.txt"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		expectedPath := "/v1/AUTH_test-tenant/" + containerName + "/" + objectName
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	if err := api.DeleteObject(containerName, objectName); err != nil {
		t.Fatalf("DeleteObject() error: %v", err)
	}
}

func TestPublishContainer(t *testing.T) {
	const containerName = "public-container"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/AUTH_test-tenant/"+containerName) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Container-Read") != ".r:*" {
			t.Errorf("expected X-Container-Read '.r:*', got %q", r.Header.Get("X-Container-Read"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	if err := api.PublishContainer(containerName); err != nil {
		t.Fatalf("PublishContainer() error: %v", err)
	}
}

func TestUnpublishContainer(t *testing.T) {
	const containerName = "public-container"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/v1/AUTH_test-tenant/"+containerName) {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("X-Container-Read") != "" {
			t.Errorf("expected empty X-Container-Read, got %q", r.Header.Get("X-Container-Read"))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	api := NewObjectStorageAPI(newTestClient(ts))
	if err := api.UnpublishContainer(containerName); err != nil {
		t.Fatalf("UnpublishContainer() error: %v", err)
	}
}
