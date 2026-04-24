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
	Name        string         `yaml:"name"`
	Hosts       []string       `yaml:"hosts"`
	Web         WebSpec        `yaml:"web"`
	ComposeFile string         `yaml:"compose_file,omitempty"`
	Accessories []string       `yaml:"accessories,omitempty"`
	Health      *HealthSpec    `yaml:"health,omitempty"`
	Deploy      *DeploySpec    `yaml:"deploy,omitempty"`
	Expose      []ExposeBlock  `yaml:"expose,omitempty"`
}

// WebSpec declares which compose service is the blue/green target.
type WebSpec struct {
	Service string `yaml:"service"`
	Port    int    `yaml:"port"`
}

// ExposeBlock declares an additional public (host, service, port) that should
// be routed by proxy and, if BlueGreen is nil or true, participate in slot
// rotation alongside web. The proxy service registered for a block is named
// "<ProjectFile.Name>-<Label>" (see spec §3.1, §7 Q-naming).
type ExposeBlock struct {
	Label     string      `yaml:"label"`
	Host      string      `yaml:"host"`
	Service   string      `yaml:"service"`
	Port      int         `yaml:"port"`
	BlueGreen *bool       `yaml:"blue_green,omitempty"`
	Health    *HealthSpec `yaml:"health,omitempty"`
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

// fqdnRe is a pragmatic FQDN check: lowercase DNS labels separated by dots,
// at least two labels. Not a full RFC 1035 parser — catches typos (spaces,
// uppercase, empty labels) without rejecting reasonable hostnames.
var fqdnRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$`)

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
	if err := p.validateExpose(); err != nil {
		return err
	}
	return nil
}

// validateExpose applies the rules from the subdomain-split RFC §3.1:
//   - label is a DNS-1123 label
//   - <name>-<label> fits in 63 chars (proxy service name constraint)
//   - label is unique across expose blocks
//   - host is a valid FQDN, not equal to any entry in hosts[], and unique
//     across expose blocks
//   - service is not equal to web.service, not in accessories, and unique
//     across expose blocks
//   - port is 1..65535
func (p *ProjectFile) validateExpose() error {
	if len(p.Expose) == 0 {
		return nil
	}
	hostSet := make(map[string]struct{}, len(p.Hosts))
	for _, h := range p.Hosts {
		hostSet[h] = struct{}{}
	}
	accSet := make(map[string]struct{}, len(p.Accessories))
	for _, a := range p.Accessories {
		accSet[a] = struct{}{}
	}
	seenLabel := make(map[string]struct{}, len(p.Expose))
	seenHost := make(map[string]struct{}, len(p.Expose))
	seenService := make(map[string]struct{}, len(p.Expose))
	for i, e := range p.Expose {
		loc := fmt.Sprintf("expose[%d]", i)
		if e.Label == "" {
			return fmt.Errorf("%s.label is required", loc)
		}
		if len(e.Label) > 63 || !dnsLabelRe.MatchString(e.Label) {
			return fmt.Errorf("%s.label %q is not a valid DNS-1123 label", loc, e.Label)
		}
		// Proxy service name is "<name>-<label>"; enforce 63-char total.
		if n := len(p.Name) + 1 + len(e.Label); n > 63 {
			return fmt.Errorf("%s: proxy service name %q-%q is %d chars, exceeds 63", loc, p.Name, e.Label, n)
		}
		if _, dup := seenLabel[e.Label]; dup {
			return fmt.Errorf("%s.label %q is duplicated across expose blocks", loc, e.Label)
		}
		seenLabel[e.Label] = struct{}{}

		if e.Host == "" {
			return fmt.Errorf("%s.host is required", loc)
		}
		if !fqdnRe.MatchString(e.Host) {
			return fmt.Errorf("%s.host %q is not a valid FQDN", loc, e.Host)
		}
		if _, clash := hostSet[e.Host]; clash {
			return fmt.Errorf("%s.host %q duplicates an entry in hosts[]", loc, e.Host)
		}
		if _, dup := seenHost[e.Host]; dup {
			return fmt.Errorf("%s.host %q is duplicated across expose blocks", loc, e.Host)
		}
		seenHost[e.Host] = struct{}{}

		if e.Service == "" {
			return fmt.Errorf("%s.service is required", loc)
		}
		if e.Service == p.Web.Service {
			return fmt.Errorf("%s.service %q conflicts with web.service", loc, e.Service)
		}
		if _, clash := accSet[e.Service]; clash {
			return fmt.Errorf("%s.service %q conflicts with an accessory", loc, e.Service)
		}
		if _, dup := seenService[e.Service]; dup {
			return fmt.Errorf("%s.service %q is duplicated across expose blocks", loc, e.Service)
		}
		seenService[e.Service] = struct{}{}

		if e.Port < 1 || e.Port > 65535 {
			return fmt.Errorf("%s.port must be between 1 and 65535, got %d", loc, e.Port)
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
	for i, e := range p.Expose {
		if _, ok := c.Services[e.Service]; !ok {
			return fmt.Errorf("expose[%d].service %q not found in %s (available: %v)", i, e.Service, composePath, composeServiceKeys(c.Services))
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
