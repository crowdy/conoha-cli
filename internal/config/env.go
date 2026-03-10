package config

import "os"

// Environment variable names
const (
	EnvProfile   = "CONOHA_PROFILE"
	EnvTenantID  = "CONOHA_TENANT_ID"
	EnvUsername  = "CONOHA_USERNAME"
	EnvPassword  = "CONOHA_PASSWORD"
	EnvToken     = "CONOHA_TOKEN"
	EnvFormat    = "CONOHA_FORMAT"
	EnvConfigDir = "CONOHA_CONFIG_DIR"
	EnvNoInput   = "CONOHA_NO_INPUT"
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
