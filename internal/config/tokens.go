package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const tokensFile = "tokens.yaml"

type TokenStore struct {
	Profiles map[string]TokenEntry `yaml:"profiles"`
}

type TokenEntry struct {
	Token     string    `yaml:"token"`
	ExpiresAt time.Time `yaml:"expires_at"`
}

func LoadTokens() (*TokenStore, error) {
	path := filepath.Join(DefaultConfigDir(), tokensFile)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &TokenStore{Profiles: map[string]TokenEntry{}}, nil
		}
		return nil, fmt.Errorf("reading tokens: %w", err)
	}

	var store TokenStore
	if err := yaml.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("parsing tokens: %w", err)
	}
	if store.Profiles == nil {
		store.Profiles = map[string]TokenEntry{}
	}
	return &store, nil
}

func (s *TokenStore) Save() error {
	dir := DefaultConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshaling tokens: %w", err)
	}
	return os.WriteFile(filepath.Join(dir, tokensFile), data, 0600)
}

func (s *TokenStore) Get(profile string) (TokenEntry, bool) {
	t, ok := s.Profiles[profile]
	return t, ok
}

func (s *TokenStore) IsValid(profile string) bool {
	t, ok := s.Profiles[profile]
	if !ok {
		return false
	}
	// Valid if more than 5 minutes remaining
	return time.Until(t.ExpiresAt) > 5*time.Minute
}

func (s *TokenStore) Set(profile string, entry TokenEntry) {
	s.Profiles[profile] = entry
}

func (s *TokenStore) Delete(profile string) {
	delete(s.Profiles, profile)
}
