package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectFile_Minimal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conoha.yml")
	data := []byte("name: myapp\nhosts: [app.example.com]\nweb:\n  service: web\n  port: 8080\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	pf, err := LoadProjectFile(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if pf.Name != "myapp" {
		t.Errorf("Name = %q, want %q", pf.Name, "myapp")
	}
	if len(pf.Hosts) != 1 || pf.Hosts[0] != "app.example.com" {
		t.Errorf("Hosts = %v", pf.Hosts)
	}
	if pf.Web.Service != "web" || pf.Web.Port != 8080 {
		t.Errorf("Web = %+v", pf.Web)
	}
}

func TestLoadProjectFile_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conoha.yml")
	data := []byte(`name: myapp
hosts: [a.example.com, b.example.com]
web:
  service: web
  port: 3000
compose_file: docker-compose.yml
accessories: [db, redis]
health:
  path: /healthz
  interval_ms: 1000
  timeout_ms: 500
  healthy_threshold: 2
  unhealthy_threshold: 5
deploy:
  drain_ms: 15000
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	pf, err := LoadProjectFile(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if pf.ComposeFile != "docker-compose.yml" {
		t.Errorf("ComposeFile = %q", pf.ComposeFile)
	}
	if len(pf.Accessories) != 2 {
		t.Errorf("Accessories = %v", pf.Accessories)
	}
	if pf.Health == nil || pf.Health.Path != "/healthz" || pf.Health.IntervalMs != 1000 {
		t.Errorf("Health = %+v", pf.Health)
	}
	if pf.Deploy == nil || pf.Deploy.DrainMs != 15000 {
		t.Errorf("Deploy = %+v", pf.Deploy)
	}
}

func TestLoadProjectFile_Missing(t *testing.T) {
	_, err := LoadProjectFile(filepath.Join(t.TempDir(), "no-such.yml"))
	if err == nil {
		t.Fatal("want error for missing file")
	}
}

func TestLoadProjectFile_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conoha.yml")
	if err := os.WriteFile(path, []byte("name: [unterminated"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadProjectFile(path)
	if err == nil {
		t.Fatal("want error for invalid YAML")
	}
}
