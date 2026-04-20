package config

import (
	"fmt"
	"os"

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
