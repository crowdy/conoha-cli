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
//
// legacyAppNameRegex is the pre-DNS-1123 rule: it allows uppercase and
// underscores, which some v0.1.x deployments used. It is only accepted
// on read/ops paths (logs, stop, restart, status, rollback, destroy) so
// pre-existing deployments can still be managed and cleaned up after the
// tightening. New init/deploy paths use the strict rule.
var (
	appNameRegex       = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)
	legacyAppNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	envKeyRegex        = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

const (
	appNameMaxLen       = 63
	legacyAppNameMaxLen = 64
)

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

// ValidateAppNameExisting is a legacy-tolerant variant used by commands that
// operate on an already-deployed app (logs, stop, restart, status, rollback,
// destroy). It accepts both the current DNS-1123 form and the older loose
// form that allowed uppercase and underscores, so users with v0.1.x-era app
// names can still tear down / inspect their deployments. New deployments are
// still gated on the strict ValidateAppName via init/deploy paths.
func ValidateAppNameExisting(name string) error {
	if name == "" {
		return fmt.Errorf("app name cannot be empty")
	}
	if len(name) > legacyAppNameMaxLen {
		return fmt.Errorf("app name too long (max %d characters)", legacyAppNameMaxLen)
	}
	if !legacyAppNameRegex.MatchString(name) {
		return fmt.Errorf("invalid app name %q: must match [a-zA-Z0-9][a-zA-Z0-9_-]*", name)
	}
	return nil
}

func ValidateEnvKey(key string) error {
	if !envKeyRegex.MatchString(key) {
		return fmt.Errorf("invalid env key %q: must match [A-Za-z_][A-Za-z0-9_]*", key)
	}
	return nil
}
