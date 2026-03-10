package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const credentialsFile = "credentials.yaml"

type CredentialsStore struct {
	Profiles map[string]Credentials `yaml:"profiles"`
}

type Credentials struct {
	Password string `yaml:"password"`
}

func LoadCredentials() (*CredentialsStore, error) {
	path := filepath.Join(DefaultConfigDir(), credentialsFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &CredentialsStore{Profiles: map[string]Credentials{}}, nil
		}
		return nil, fmt.Errorf("reading credentials: %w", err)
	}

	var store CredentialsStore
	if err := yaml.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	if store.Profiles == nil {
		store.Profiles = map[string]Credentials{}
	}
	return &store, nil
}

func (s *CredentialsStore) Save() error {
	dir := DefaultConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, credentialsFile), data, 0600)
}

func (s *CredentialsStore) Get(profile string) (Credentials, bool) {
	c, ok := s.Profiles[profile]
	return c, ok
}

func (s *CredentialsStore) Set(profile string, cred Credentials) {
	s.Profiles[profile] = cred
}

func (s *CredentialsStore) Delete(profile string) {
	delete(s.Profiles, profile)
}
