package config

import "os"

// Environment variable names
const (
	EnvProfile      = "CONOHA_PROFILE"
	EnvTenantID     = "CONOHA_TENANT_ID"
	EnvUsername     = "CONOHA_USERNAME"
	EnvPassword     = "CONOHA_PASSWORD"
	EnvToken        = "CONOHA_TOKEN"
	EnvFormat       = "CONOHA_FORMAT"
	EnvConfigDir    = "CONOHA_CONFIG_DIR"
	EnvNoInput      = "CONOHA_NO_INPUT"
	EnvEndpoint     = "CONOHA_ENDPOINT"
	EnvEndpointMode = "CONOHA_ENDPOINT_MODE"
	EnvDebug        = "CONOHA_DEBUG"
	EnvYes          = "CONOHA_YES"
	EnvNoColor      = "CONOHA_NO_COLOR"
	EnvSSHInsecure  = "CONOHA_SSH_INSECURE"
)

// EnvOr returns the environment variable value if set, otherwise the fallback.
func EnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// IsNoInput returns true if non-interactive mode is requested.
func IsNoInput() bool {
	return os.Getenv(EnvNoInput) == "1" || os.Getenv(EnvNoInput) == "true"
}

// IsYes returns true if confirmation prompts should be auto-confirmed.
func IsYes() bool {
	return os.Getenv(EnvYes) == "1" || os.Getenv(EnvYes) == "true"
}

// IsNoColor returns true if color output should be disabled.
// Supports both CONOHA_NO_COLOR and the standard NO_COLOR env var.
func IsNoColor() bool {
	if v := os.Getenv(EnvNoColor); v == "1" || v == "true" {
		return true
	}
	_, noColor := os.LookupEnv("NO_COLOR")
	return noColor
}

// IsSSHInsecure returns true when SSH host-key verification should be
// disabled (InsecureIgnoreHostKey). Set via --insecure flag or the env var.
// Default false — real known_hosts verification with TOFU fallback.
func IsSSHInsecure() bool {
	return os.Getenv(EnvSSHInsecure) == "1" || os.Getenv(EnvSSHInsecure) == "true"
}
