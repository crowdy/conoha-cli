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

func TestProjectFile_Validate(t *testing.T) {
	good := ProjectFile{
		Name:  "myapp",
		Hosts: []string{"app.example.com"},
		Web:   WebSpec{Service: "web", Port: 8080},
	}
	if err := good.Validate(); err != nil {
		t.Errorf("good Validate: %v", err)
	}

	cases := []struct {
		name string
		mod  func(*ProjectFile)
		want string // substring of error
	}{
		{"empty name", func(p *ProjectFile) { p.Name = "" }, "name"},
		{"bad name", func(p *ProjectFile) { p.Name = "My-App_1" }, "name"},
		{"too long name", func(p *ProjectFile) { p.Name = "a" + string(make([]byte, 63)) }, "name"},
		{"no hosts", func(p *ProjectFile) { p.Hosts = nil }, "hosts"},
		{"empty host", func(p *ProjectFile) { p.Hosts = []string{""} }, "hosts"},
		{"dup hosts", func(p *ProjectFile) { p.Hosts = []string{"a.com", "a.com"} }, "duplicate"},
		{"no web service", func(p *ProjectFile) { p.Web.Service = "" }, "web.service"},
		{"no port", func(p *ProjectFile) { p.Web.Port = 0 }, "web.port"},
		{"bad port high", func(p *ProjectFile) { p.Web.Port = 70000 }, "web.port"},
		{"accessory = web", func(p *ProjectFile) { p.Accessories = []string{"web"} }, "accessor"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := good
			tc.mod(&p)
			err := p.Validate()
			if err == nil {
				t.Fatalf("want error containing %q, got nil", tc.want)
			}
			if !contains(err.Error(), tc.want) {
				t.Errorf("error = %q, want substring %q", err.Error(), tc.want)
			}
		})
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestResolveComposeFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "compose.yml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &ProjectFile{}
	got, err := p.ResolveComposeFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "compose.yml" {
		t.Errorf("got %q, want compose.yml", got)
	}

	p.ComposeFile = "custom.yml"
	if _, err := p.ResolveComposeFile(dir); err == nil {
		t.Error("want error for missing compose_file")
	}
	if err := os.WriteFile(filepath.Join(dir, "custom.yml"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err = p.ResolveComposeFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "custom.yml" {
		t.Errorf("got %q", got)
	}
}
