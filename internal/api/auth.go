package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/crowdy/conoha-cli/internal/config"
	cerrors "github.com/crowdy/conoha-cli/internal/errors"
)

// AuthResponse represents the token response from the Identity API.
type AuthResponse struct {
	Token struct {
		ExpiresAt string `json:"expires_at"`
	} `json:"token"`
}

// TokenResult contains the token string and expiration.
type TokenResult struct {
	Token     string
	ExpiresAt time.Time
}

// Authenticate obtains a new token from the Identity API.
func Authenticate(region, tenantID, username, password string) (*TokenResult, error) {
	baseURL := fmt.Sprintf("https://identity.%s.conoha.io", region)
	if ep := os.Getenv(config.EnvEndpoint); ep != "" {
		baseURL = ep
	}
	url := baseURL + "/v3/auth/tokens"

	body := map[string]any{
		"auth": map[string]any{
			"identity": map[string]any{
				"methods": []string{"password"},
				"password": map[string]any{
					"user": map[string]any{
						"name":     username,
						"password": password,
						"domain": map[string]any{
							"id": "default",
						},
					},
				},
			},
			"scope": map[string]any{
				"project": map[string]any{
					"id": tenantID,
				},
			},
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling auth request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	debugLogRequest(req, jsonBody)

	client := &http.Client{Timeout: 30 * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, &cerrors.NetworkError{Err: err}
	}
	defer resp.Body.Close()

	// Debug log response
	elapsed := time.Since(start)
	if debugLevel >= DebugAPI {
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		debugLogResponse(resp, elapsed, respBody)
	} else {
		debugLogResponse(resp, elapsed, nil)
	}

	if resp.StatusCode != http.StatusCreated {
		return nil, &cerrors.AuthError{
			Message: fmt.Sprintf("authentication failed with status %d", resp.StatusCode),
		}
	}

	token := resp.Header.Get("X-Subject-Token")
	if token == "" {
		return nil, &cerrors.AuthError{Message: "no token in response"}
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, fmt.Errorf("decoding auth response: %w", err)
	}

	expiresAt, err := time.Parse(time.RFC3339, authResp.Token.ExpiresAt)
	if err != nil {
		// Default to 24 hours from now
		expiresAt = time.Now().Add(24 * time.Hour)
	}

	return &TokenResult{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

// EnsureToken returns a valid token, refreshing if necessary.
// Priority: env var > cached token > re-authenticate.
func EnsureToken(profile string, cfg *config.Config, creds *config.CredentialsStore, tokens *config.TokenStore) (string, error) {
	// 1. Check environment variable
	if t := config.EnvOr(config.EnvToken, ""); t != "" {
		return t, nil
	}

	// 2. Check cached token
	if tokens.IsValid(profile) {
		entry, _ := tokens.Get(profile)
		return entry.Token, nil
	}

	// 3. Re-authenticate
	p, ok := cfg.Profiles[profile]
	if !ok {
		return "", &cerrors.ConfigError{Message: fmt.Sprintf("profile %q not found", profile)}
	}

	cred, ok := creds.Get(profile)
	if !ok {
		return "", &cerrors.AuthError{Message: fmt.Sprintf("no credentials for profile %q, run 'conoha auth login'", profile)}
	}

	tenantID := config.EnvOr(config.EnvTenantID, p.TenantID)
	username := config.EnvOr(config.EnvUsername, p.Username)
	password := config.EnvOr(config.EnvPassword, cred.Password)
	region := p.Region
	if region == "" {
		region = config.DefaultRegion
	}

	result, err := Authenticate(region, tenantID, username, password)
	if err != nil {
		return "", err
	}

	// Cache the token
	tokens.Set(profile, config.TokenEntry{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
	})
	// Non-fatal: token works even if cache save fails
	_ = tokens.Save()

	return result.Token, nil
}
