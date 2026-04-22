package ssh

import (
	"fmt"
	"regexp"
)

// appNameRegex mirrors config.dnsLabelRe (RFC 1123 label: lowercase
// alphanumerics and hyphens, must start and end with alphanumeric).
// Kept consistent so app-name values flow without mismatch through
// docker compose project names, proxy service names, and the on-disk
// work dir at /opt/conoha/<name>.
var (
	appNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	envKeyRegex  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

const appNameMaxLen = 63

func ValidateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("app name cannot be empty")
	}
	if len(name) > appNameMaxLen {
		return fmt.Errorf("app name too long (max %d characters)", appNameMaxLen)
	}
	if !appNameRegex.MatchString(name) {
		return fmt.Errorf("invalid app name %q: must be a DNS-1123 label (lowercase alphanumerics and hyphens, must start and end with alphanumeric)", name)
	}
	return nil
}

func ValidateEnvKey(key string) error {
	if !envKeyRegex.MatchString(key) {
		return fmt.Errorf("invalid env key %q: must match [A-Za-z_][A-Za-z0-9_]*", key)
	}
	return nil
}
