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

func TestProjectFile_Validate_Expose(t *testing.T) {
	base := func() ProjectFile {
		return ProjectFile{
			Name:  "myapp",
			Hosts: []string{"app.example.com"},
			Web:   WebSpec{Service: "web", Port: 8080},
			Expose: []ExposeBlock{
				{Label: "dex", Host: "dex.example.com", Service: "dex", Port: 5556},
			},
		}
	}

	t.Run("single valid block", func(t *testing.T) {
		p := base()
		if err := p.Validate(); err != nil {
			t.Fatalf("valid expose should pass: %v", err)
		}
	})

	t.Run("multiple valid blocks", func(t *testing.T) {
		p := base()
		p.Expose = append(p.Expose, ExposeBlock{Label: "api", Host: "api.example.com", Service: "api", Port: 8000})
		if err := p.Validate(); err != nil {
			t.Fatalf("two blocks should pass: %v", err)
		}
	})

	cases := []struct {
		name string
		mod  func(*ProjectFile)
		want string
	}{
		{"empty label", func(p *ProjectFile) { p.Expose[0].Label = "" }, "label"},
		{"bad label", func(p *ProjectFile) { p.Expose[0].Label = "Dex_1" }, "DNS-1123"},
		{"label too long with name",
			func(p *ProjectFile) {
				p.Name = "a-very-long-app-name-that-consumes-most-of-the-63-char-budget"
				p.Expose[0].Label = "equally-long-label"
			},
			"exceeds 63"},
		{"duplicate label",
			func(p *ProjectFile) {
				p.Expose = append(p.Expose, ExposeBlock{Label: "dex", Host: "other.example.com", Service: "other", Port: 81})
			},
			"duplicated"},
		{"empty host", func(p *ProjectFile) { p.Expose[0].Host = "" }, "host"},
		{"host not fqdn", func(p *ProjectFile) { p.Expose[0].Host = "not a host" }, "FQDN"},
		{"host duplicates hosts[]",
			func(p *ProjectFile) { p.Expose[0].Host = "app.example.com" },
			"duplicates"},
		{"duplicate host across expose",
			func(p *ProjectFile) {
				p.Expose = append(p.Expose, ExposeBlock{Label: "api", Host: "dex.example.com", Service: "api", Port: 81})
			},
			"duplicated"},
		{"service equals web.service",
			func(p *ProjectFile) { p.Expose[0].Service = "web" },
			"web.service"},
		{"service in accessories",
			func(p *ProjectFile) {
				p.Accessories = []string{"dex"}
				p.Expose[0].Service = "dex"
			},
			"accessory"},
		{"duplicate service across expose",
			func(p *ProjectFile) {
				p.Expose = append(p.Expose, ExposeBlock{Label: "api", Host: "api.example.com", Service: "dex", Port: 81})
			},
			"duplicated"},
		{"port zero", func(p *ProjectFile) { p.Expose[0].Port = 0 }, "port"},
		{"port too high", func(p *ProjectFile) { p.Expose[0].Port = 70000 }, "port"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := base()
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

func TestValidateAgainstCompose_Expose(t *testing.T) {
	dir := t.TempDir()
	compose := filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(compose, []byte("services:\n  web:\n    image: nginx\n  dex:\n    image: dex\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &ProjectFile{
		Name:  "myapp",
		Hosts: []string{"a.example.com"},
		Web:   WebSpec{Service: "web", Port: 80},
		Expose: []ExposeBlock{
			{Label: "dex", Host: "dex.example.com", Service: "dex", Port: 5556},
		},
	}
	if err := p.ValidateAgainstCompose(compose); err != nil {
		t.Errorf("known expose service should pass: %v", err)
	}

	p.Expose[0].Service = "nonexistent"
	err := p.ValidateAgainstCompose(compose)
	if err == nil {
		t.Fatal("missing expose.service should fail")
	}
	if !contains(err.Error(), "expose[0].service") {
		t.Errorf("error should cite expose[0], got %q", err.Error())
	}
}

func TestLoadProjectFile_Expose(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conoha.yml")
	data := []byte(`name: gitea
hosts: [gitea.example.com]
web:
  service: gitea
  port: 3000
expose:
  - label: dex
    host: dex.example.com
    service: dex
    port: 5556
    blue_green: false
    health:
      path: /healthz
      interval_ms: 1000
`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	pf, err := LoadProjectFile(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(pf.Expose) != 1 {
		t.Fatalf("Expose len = %d, want 1", len(pf.Expose))
	}
	e := pf.Expose[0]
	if e.Label != "dex" || e.Host != "dex.example.com" || e.Service != "dex" || e.Port != 5556 {
		t.Errorf("Expose[0] = %+v", e)
	}
	if e.BlueGreen == nil || *e.BlueGreen != false {
		t.Errorf("BlueGreen = %v, want *bool -> false", e.BlueGreen)
	}
	if e.Health == nil || e.Health.Path != "/healthz" {
		t.Errorf("Health = %+v", e.Health)
	}
	if err := pf.Validate(); err != nil {
		t.Errorf("loaded fixture Validate: %v", err)
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

func TestValidateAgainstCompose(t *testing.T) {
	dir := t.TempDir()
	compose := filepath.Join(dir, "compose.yml")
	if err := os.WriteFile(compose, []byte("services:\n  web:\n    image: nginx\n  db:\n    image: postgres\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := &ProjectFile{
		Name:  "myapp",
		Hosts: []string{"a.example.com"},
		Web:   WebSpec{Service: "web", Port: 80},
	}
	if err := p.ValidateAgainstCompose(compose); err != nil {
		t.Errorf("ok case: %v", err)
	}

	p.Accessories = []string{"db"}
	if err := p.ValidateAgainstCompose(compose); err != nil {
		t.Errorf("db accessory: %v", err)
	}

	p.Web.Service = "nonexistent"
	if err := p.ValidateAgainstCompose(compose); err == nil {
		t.Error("missing web.service should fail")
	}

	p.Web.Service = "web"
	p.Accessories = []string{"cache"}
	if err := p.ValidateAgainstCompose(compose); err == nil {
		t.Error("missing accessory should fail")
	}

	if err := p.ValidateAgainstCompose(filepath.Join(dir, "nope.yml")); err == nil {
		t.Error("missing file should fail")
	}
}
