package ssh

import (
	"fmt"
	"regexp"
)

var (
	appNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)
	envKeyRegex  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

func ValidateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("app name cannot be empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("app name too long (max 64 characters)")
	}
	if !appNameRegex.MatchString(name) {
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
