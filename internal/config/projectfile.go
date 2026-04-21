package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// ProjectFileName is the canonical project file name.
const ProjectFileName = "conoha.yml"

// ProjectFile is the parsed conoha.yml declaration that lives at the repo root.
type ProjectFile struct {
	Name        string      `yaml:"name"`
	Hosts       []string    `yaml:"hosts"`
	Web         WebSpec     `yaml:"web"`
	ComposeFile string      `yaml:"compose_file,omitempty"`
	Accessories []string    `yaml:"accessories,omitempty"`
	Health      *HealthSpec `yaml:"health,omitempty"`
	Deploy      *DeploySpec `yaml:"deploy,omitempty"`
}

// WebSpec declares which compose service is the blue/green target.
type WebSpec struct {
	Service string `yaml:"service"`
	Port    int    `yaml:"port"`
}

// HealthSpec mirrors proxy's health_policy object (all fields optional).
type HealthSpec struct {
	Path               string `yaml:"path,omitempty"`
	IntervalMs         int    `yaml:"interval_ms,omitempty"`
	TimeoutMs          int    `yaml:"timeout_ms,omitempty"`
	HealthyThreshold   int    `yaml:"healthy_threshold,omitempty"`
	UnhealthyThreshold int    `yaml:"unhealthy_threshold,omitempty"`
}

// DeploySpec carries deploy-time parameters (currently only drain_ms).
type DeploySpec struct {
	DrainMs int `yaml:"drain_ms,omitempty"`
}

// LoadProjectFile reads and YAML-decodes a project file. It does NOT validate.
func LoadProjectFile(path string) (*ProjectFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var pf ProjectFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return &pf, nil
}

var dnsLabelRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

// Validate enforces the schema rules documented in the spec.
func (p *ProjectFile) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if len(p.Name) > 63 || !dnsLabelRe.MatchString(p.Name) {
		return fmt.Errorf("name %q is not a valid DNS-1123 label (lowercase alphanumerics and hyphens, 1-63 chars)", p.Name)
	}
	if len(p.Hosts) == 0 {
		return fmt.Errorf("hosts must list at least one FQDN")
	}
	seen := make(map[string]struct{}, len(p.Hosts))
	for _, h := range p.Hosts {
		if h == "" {
			return fmt.Errorf("hosts contains empty entry")
		}
		if _, dup := seen[h]; dup {
			return fmt.Errorf("hosts contains duplicate %q", h)
		}
		seen[h] = struct{}{}
	}
	if p.Web.Service == "" {
		return fmt.Errorf("web.service is required")
	}
	if p.Web.Port < 1 || p.Web.Port > 65535 {
		return fmt.Errorf("web.port must be between 1 and 65535, got %d", p.Web.Port)
	}
	for _, a := range p.Accessories {
		if a == p.Web.Service {
			return fmt.Errorf("accessory %q conflicts with web.service", a)
		}
	}
	return nil
}

// ComposeFileCandidates is the auto-detect order when compose_file is unset.
// Mirrors cmd/app/deploy.go composeFileNames as of the conoha-proxy refactor.
var ComposeFileCandidates = []string{
	"conoha-docker-compose.yml",
	"conoha-docker-compose.yaml",
	"docker-compose.yml",
	"docker-compose.yaml",
	"compose.yml",
	"compose.yaml",
}

// ResolveComposeFile returns the compose file path relative to dir.
// If the project file specified compose_file explicitly it is returned (existence is verified).
// Otherwise the first existing candidate from ComposeFileCandidates is returned.
func (p *ProjectFile) ResolveComposeFile(dir string) (string, error) {
	if p.ComposeFile != "" {
		full := p.ComposeFile
		if _, err := os.Stat(filepathJoin(dir, full)); err != nil {
			return "", fmt.Errorf("compose_file %q not found", full)
		}
		return full, nil
	}
	for _, name := range ComposeFileCandidates {
		if _, err := os.Stat(filepathJoin(dir, name)); err == nil {
			return name, nil
		}
	}
	return "", fmt.Errorf("no compose file found (tried %v)", ComposeFileCandidates)
}

func filepathJoin(dir, name string) string {
	if dir == "" || dir == "." {
		return name
	}
	return dir + string(os.PathSeparator) + name
}

// composeFileShape is the subset of a docker-compose YAML we need for cross-validation.
type composeFileShape struct {
	Services map[string]interface{} `yaml:"services"`
}

// ValidateAgainstCompose reads the resolved compose file and verifies that
//   - web.service exists as a service
//   - every accessory exists as a service
//
// It returns a single error describing the first failure, if any.
func (p *ProjectFile) ValidateAgainstCompose(composePath string) error {
	data, err := os.ReadFile(composePath)
	if err != nil {
		return fmt.Errorf("read compose %s: %w", composePath, err)
	}
	var c composeFileShape
	if err := yaml.Unmarshal(data, &c); err != nil {
		return fmt.Errorf("parse compose %s: %w", composePath, err)
	}
	if _, ok := c.Services[p.Web.Service]; !ok {
		return fmt.Errorf("web.service %q not found in %s (available: %v)", p.Web.Service, composePath, composeServiceKeys(c.Services))
	}
	for _, a := range p.Accessories {
		if _, ok := c.Services[a]; !ok {
			return fmt.Errorf("accessory %q not found in %s (available: %v)", a, composePath, composeServiceKeys(c.Services))
		}
	}
	return nil
}

func composeServiceKeys(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
