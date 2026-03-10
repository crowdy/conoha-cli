package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("CONOHA_CONFIG_DIR", dir)
	return dir
}

func TestDefaultConfig(t *testing.T) {
	setupTestDir(t)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.ActiveProfile != "default" {
		t.Errorf("expected 'default', got %q", cfg.ActiveProfile)
	}
	if cfg.Defaults.Format != "table" {
		t.Errorf("expected 'table', got %q", cfg.Defaults.Format)
	}
}

func TestConfigSaveLoad(t *testing.T) {
	dir := setupTestDir(t)
	cfg := &Config{
		Version:       1,
		ActiveProfile: "test",
		Defaults:      Defaults{Format: "json"},
		Profiles: map[string]Profile{
			"test": {TenantID: "t1", Username: "u1", Region: "c3j1"},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Stat() error: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected 0600, got %o", perm)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.ActiveProfile != "test" {
		t.Errorf("expected 'test', got %q", loaded.ActiveProfile)
	}
	p := loaded.Profiles["test"]
	if p.TenantID != "t1" || p.Username != "u1" {
		t.Errorf("profile mismatch: %+v", p)
	}
}

func TestActiveProfileConfig(t *testing.T) {
	setupTestDir(t)
	cfg := &Config{
		ActiveProfile: "prod",
		Profiles: map[string]Profile{
			"prod": {Region: "tyo1"},
		},
	}
	p := cfg.ActiveProfileConfig()
	if p.Region != "tyo1" {
		t.Errorf("expected 'tyo1', got %q", p.Region)
	}

	cfg.ActiveProfile = "nonexistent"
	p = cfg.ActiveProfileConfig()
	if p.Region != DefaultRegion {
		t.Errorf("expected default region %q, got %q", DefaultRegion, p.Region)
	}
}

func TestCredentialsSaveLoad(t *testing.T) {
	setupTestDir(t)
	store := &CredentialsStore{
		Profiles: map[string]Credentials{
			"default": {Password: "secret"},
		},
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := LoadCredentials()
	if err != nil {
		t.Fatalf("LoadCredentials() error: %v", err)
	}
	cred, ok := loaded.Get("default")
	if !ok {
		t.Fatal("expected credential for 'default'")
	}
	if cred.Password != "secret" {
		t.Errorf("expected 'secret', got %q", cred.Password)
	}

	loaded.Delete("default")
	if _, ok := loaded.Get("default"); ok {
		t.Error("expected credential to be deleted")
	}
}

func TestTokenStore(t *testing.T) {
	setupTestDir(t)
	store := &TokenStore{
		Profiles: map[string]TokenEntry{
			"default": {Token: "tok123", ExpiresAt: time.Now().Add(1 * time.Hour)},
			"expired": {Token: "old", ExpiresAt: time.Now().Add(-1 * time.Hour)},
		},
	}
	if err := store.Save(); err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	loaded, err := LoadTokens()
	if err != nil {
		t.Fatalf("LoadTokens() error: %v", err)
	}

	if !loaded.IsValid("default") {
		t.Error("expected 'default' token to be valid")
	}
	if loaded.IsValid("expired") {
		t.Error("expected 'expired' token to be invalid")
	}
	if loaded.IsValid("nonexistent") {
		t.Error("expected nonexistent to be invalid")
	}

	entry, ok := loaded.Get("default")
	if !ok || entry.Token != "tok123" {
		t.Errorf("unexpected token entry: %+v", entry)
	}

	loaded.Delete("default")
	if loaded.IsValid("default") {
		t.Error("expected deleted token to be invalid")
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("TEST_VAR", "hello")
	if v := EnvOr("TEST_VAR", "fallback"); v != "hello" {
		t.Errorf("expected 'hello', got %q", v)
	}
	if v := EnvOr("NONEXISTENT_VAR", "fallback"); v != "fallback" {
		t.Errorf("expected 'fallback', got %q", v)
	}
}

func TestIsNoInput(t *testing.T) {
	t.Setenv("CONOHA_NO_INPUT", "1")
	if !IsNoInput() {
		t.Error("expected true")
	}
	t.Setenv("CONOHA_NO_INPUT", "false")
	if IsNoInput() {
		t.Error("expected false")
	}
}
