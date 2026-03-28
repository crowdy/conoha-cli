package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/crowdy/conoha-cli/internal/config"
)

func TestAuthenticate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v3/auth/tokens" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("X-Subject-Token", "new-token-123")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"token":{"expires_at":"2099-12-31T23:59:59Z"}}`)
	}))
	defer ts.Close()

	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	result, err := Authenticate("c3j1", "tenant-id", "user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token != "new-token-123" {
		t.Errorf("expected new-token-123, got %s", result.Token)
	}
	if result.ExpiresAt.Year() != 2099 {
		t.Errorf("expected expires_at year 2099, got %d", result.ExpiresAt.Year())
	}
}

func TestAuthenticateNon201Error(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"message":"unauthorized"}}`)
	}))
	defer ts.Close()

	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	_, err := Authenticate("c3j1", "tenant-id", "bad-user", "bad-pass")
	if err == nil {
		t.Fatal("expected error for non-201 response")
	}
}

func TestAuthenticateNoTokenHeader(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"token":{"expires_at":"2099-12-31T23:59:59Z"}}`)
		// No X-Subject-Token header
	}))
	defer ts.Close()

	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	_, err := Authenticate("c3j1", "tenant-id", "user", "pass")
	if err == nil {
		t.Fatal("expected error when X-Subject-Token header is missing")
	}
}

func TestAuthenticateExpiresAtFallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Subject-Token", "fallback-token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		// expires_at is not a valid RFC3339 timestamp — should fall back to 24h from now
		fmt.Fprint(w, `{"token":{"expires_at":"not-a-date"}}`)
	}))
	defer ts.Close()

	t.Setenv("CONOHA_ENDPOINT", ts.URL)

	before := time.Now()
	result, err := Authenticate("c3j1", "tenant-id", "user", "pass")
	if err != nil {
		t.Fatal(err)
	}
	if result.Token != "fallback-token" {
		t.Errorf("expected fallback-token, got %s", result.Token)
	}
	// ExpiresAt should be roughly 24 hours from now
	expected := before.Add(24 * time.Hour)
	diff := result.ExpiresAt.Sub(expected)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("expected ExpiresAt near %v, got %v (diff %v)", expected, result.ExpiresAt, diff)
	}
}

func TestEnsureTokenFromEnv(t *testing.T) {
	t.Setenv("CONOHA_TOKEN", "env-token")

	token, err := EnsureToken("default", nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "env-token" {
		t.Errorf("expected env-token, got %s", token)
	}
}

func TestEnsureTokenCached(t *testing.T) {
	// Use a temp dir so TokenStore.Save() doesn't touch the real config dir
	t.Setenv("CONOHA_CONFIG_DIR", t.TempDir())
	// Ensure CONOHA_TOKEN is unset so the env path is not taken
	t.Setenv("CONOHA_TOKEN", "")

	tokens := &config.TokenStore{
		Profiles: map[string]config.TokenEntry{
			"default": {
				Token:     "cached-token",
				ExpiresAt: time.Now().Add(2 * time.Hour), // well within the 5-minute threshold
			},
		},
	}

	token, err := EnsureToken("default", nil, nil, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if token != "cached-token" {
		t.Errorf("expected cached-token, got %s", token)
	}
}

func TestEnsureTokenReauthenticate(t *testing.T) {
	// Set up a test server to handle re-authentication
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/v3/auth/tokens" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("X-Subject-Token", "reauth-token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"token":{"expires_at":"2099-12-31T23:59:59Z"}}`)
	}))
	defer ts.Close()

	t.Setenv("CONOHA_ENDPOINT", ts.URL)
	t.Setenv("CONOHA_TOKEN", "")
	t.Setenv("CONOHA_CONFIG_DIR", t.TempDir())

	// Token store with an expired token
	tokens := &config.TokenStore{
		Profiles: map[string]config.TokenEntry{
			"myprofile": {
				Token:     "expired-token",
				ExpiresAt: time.Now().Add(-1 * time.Hour), // already expired
			},
		},
	}

	cfg := &config.Config{
		Profiles: map[string]config.Profile{
			"myprofile": {
				TenantID: "tenant-id",
				Username: "user",
				Region:   "c3j1",
			},
		},
	}

	creds := &config.CredentialsStore{
		Profiles: map[string]config.Credentials{
			"myprofile": {Password: "pass"},
		},
	}

	token, err := EnsureToken("myprofile", cfg, creds, tokens)
	if err != nil {
		t.Fatal(err)
	}
	if token != "reauth-token" {
		t.Errorf("expected reauth-token, got %s", token)
	}

	// Verify the token was cached in the store
	entry, ok := tokens.Get("myprofile")
	if !ok {
		t.Fatal("expected token to be cached in store after re-authentication")
	}
	if entry.Token != "reauth-token" {
		t.Errorf("expected cached token reauth-token, got %s", entry.Token)
	}
}

func TestEnsureTokenProfileNotFound(t *testing.T) {
	t.Setenv("CONOHA_TOKEN", "")

	tokens := &config.TokenStore{
		Profiles: map[string]config.TokenEntry{},
	}

	cfg := &config.Config{
		Profiles: map[string]config.Profile{}, // empty — profile not found
	}

	creds := &config.CredentialsStore{
		Profiles: map[string]config.Credentials{},
	}

	_, err := EnsureToken("nonexistent", cfg, creds, tokens)
	if err == nil {
		t.Fatal("expected error when profile not found")
	}
}
