package server

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/api"
	"github.com/crowdy/conoha-cli/internal/model"
)

func newTestCmd(flags map[string]string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("user-data", "", "")
	cmd.Flags().String("user-data-raw", "", "")
	cmd.Flags().String("user-data-url", "", "")
	for k, v := range flags {
		_ = cmd.Flags().Set(k, v)
	}
	return cmd
}

func TestResolveUserData_None(t *testing.T) {
	cmd := newTestCmd(nil)
	got, err := resolveUserData(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestResolveUserData_Raw(t *testing.T) {
	cmd := newTestCmd(map[string]string{"user-data-raw": "#!/bin/bash\necho hello"})
	got, err := resolveUserData(cmd)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != "#!/bin/bash\necho hello" {
		t.Errorf("decoded = %q", string(decoded))
	}
}

func TestResolveUserData_URL(t *testing.T) {
	cmd := newTestCmd(map[string]string{"user-data-url": "https://example.com/setup.sh"})
	got, err := resolveUserData(cmd)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	want := "#include\nhttps://example.com/setup.sh\n"
	if string(decoded) != want {
		t.Errorf("decoded = %q, want %q", string(decoded), want)
	}
}

func TestResolveUserData_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "startup.sh")
	content := "#!/bin/bash\napt update"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCmd(map[string]string{"user-data": path})
	got, err := resolveUserData(cmd)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(got)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != content {
		t.Errorf("decoded = %q, want %q", string(decoded), content)
	}
}

func TestResolveUserData_FileNotFound(t *testing.T) {
	cmd := newTestCmd(map[string]string{"user-data": "/nonexistent/file.sh"})
	_, err := resolveUserData(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveUserData_TooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.sh")
	data := strings.Repeat("x", userDataMaxSize+1)
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cmd := newTestCmd(map[string]string{"user-data": path})
	_, err := resolveUserData(cmd)
	if err == nil {
		t.Fatal("expected error for oversized file")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Errorf("expected 'too large' error, got: %v", err)
	}
}

func TestMaxBootVolumeGB(t *testing.T) {
	tests := []struct {
		name   string
		ram    int
		wantGB int
	}{
		{"512MB plan", 512, 30},
		{"1GB plan", 1024, 100},
		{"2GB plan", 2048, 100},
		{"4GB plan", 4096, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &model.Flavor{RAM: tt.ram}
			if got := maxBootVolumeGB(f); got != tt.wantGB {
				t.Errorf("maxBootVolumeGB(RAM=%d) = %d, want %d", tt.ram, got, tt.wantGB)
			}
		})
	}
}

func TestBootVolumeSizes(t *testing.T) {
	// 512MB plan should only offer 30GB
	f512 := &model.Flavor{RAM: 512}
	sizes := bootVolumeSizes(f512)
	if len(sizes) != 1 || sizes[0].Value != "30" {
		t.Errorf("512MB plan: expected [30GB], got %v", sizes)
	}

	// 1GB+ plan should offer 100GB
	f1g := &model.Flavor{RAM: 1024}
	sizes = bootVolumeSizes(f1g)
	if len(sizes) != 1 || sizes[0].Value != "100" {
		t.Errorf("1GB plan: expected [100GB], got %v", sizes)
	}
}

func TestIsUsableFlavor(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"g2l-t-c1m512", true},
		{"g2l-t-c2m1", true},
		{"g2l-t-c4m4", true},
		{"g2l-p-c1m512", false}, // prepaid
		{"g2w-t-c2m1", false},   // Windows
		{"g2d-t-c2m1", false},   // dedicated
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isUsableFlavor(tt.name); got != tt.want {
				t.Errorf("isUsableFlavor(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestResolveUserData_MutualExclusion(t *testing.T) {
	cmd := newTestCmd(map[string]string{
		"user-data-raw": "echo hi",
		"user-data-url": "https://example.com/x.sh",
	})
	_, err := resolveUserData(cmd)
	if err == nil {
		t.Fatal("expected error for multiple flags")
	}
	if !strings.Contains(err.Error(), "only one") {
		t.Errorf("expected 'only one' error, got: %v", err)
	}
}

func TestResolveBootVolume_NonInteractive_AutoCreates(t *testing.T) {
	t.Setenv("CONOHA_NO_INPUT", "1")

	mux := http.NewServeMux()
	var gotReq model.VolumeCreateRequest
	mux.HandleFunc("POST /v3/{tenant}/volumes", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotReq)
		resp := `{"volume":{"id":"vol-auto-1","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(resp))
	})
	mux.HandleFunc("GET /v3/{tenant}/volumes/vol-auto-1", func(w http.ResponseWriter, r *http.Request) {
		resp := `{"volume":{"id":"vol-auto-1","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
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
	mux.HandleFunc("GET /v3/{tenant}/volumes/vol-existing", func(w http.ResponseWriter, r *http.Request) {
		resp := `{"volume":{"id":"vol-existing","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
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

func TestCreateBootVolume_WithDefaults(t *testing.T) {
	mux := http.NewServeMux()

	var gotReq model.VolumeCreateRequest
	mux.HandleFunc("POST /v3/{tenant}/volumes", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotReq)
		resp := `{"volume":{"id":"vol-new-123","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(resp))
	})
	mux.HandleFunc("GET /v3/{tenant}/volumes/vol-new-123", func(w http.ResponseWriter, r *http.Request) {
		resp := `{"volume":{"id":"vol-new-123","status":"available","size":100}}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
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

func TestCreateBootVolume_512MBPlan(t *testing.T) {
	mux := http.NewServeMux()

	var gotReq model.VolumeCreateRequest
	mux.HandleFunc("POST /v3/{tenant}/volumes", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotReq)
		resp := `{"volume":{"id":"vol-small-1","status":"available","size":30}}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(resp))
	})
	mux.HandleFunc("GET /v3/{tenant}/volumes/vol-small-1", func(w http.ResponseWriter, r *http.Request) {
		resp := `{"volume":{"id":"vol-small-1","status":"available","size":30}}`
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(resp))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()
	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	client := &api.Client{HTTP: ts.Client(), Token: "fake-token", TenantID: "tenant-1"}
	volumeAPI := api.NewVolumeAPI(client)
	flavor := &model.Flavor{Name: "g2l-t-c1m512", RAM: 512}

	_, _, err := createBootVolumeWithDefaults(volumeAPI, flavor, "img-abc", "small-server")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotReq.Volume.Size != 30 {
		t.Errorf("volume size = %d, want 30 for 512MB plan", gotReq.Volume.Size)
	}
	if gotReq.Volume.Name != "small-server-boot" {
		t.Errorf("volume name = %q, want %q", gotReq.Volume.Name, "small-server-boot")
	}
}
