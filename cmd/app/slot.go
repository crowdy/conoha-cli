package app

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// slotIDRe permits lowercase alphanumerics and hyphens, 1-64 chars.
// Chosen to match git short-SHA (7 hex), timestamps ("YYYYMMDDHHMMSS"),
// and collision suffixes like "abc1234-2". Strict enough to be safe for
// inclusion in shell commands and compose project names.
var slotIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)

// ValidateSlotID rejects anything that could be unsafe in shell commands
// or compose project names. Applied to --slot and to CURRENT_SLOT content.
func ValidateSlotID(slot string) error {
	if !slotIDRe.MatchString(slot) {
		return fmt.Errorf("invalid slot ID %q: must match %s", slot, slotIDRe)
	}
	return nil
}

// determineSlotID returns either the repo's 7-char HEAD short SHA (when useGit)
// or a timestamp "YYYYMMDDHHMMSS" otherwise. useGit=false is used when the caller
// knows or has decided git isn't appropriate.
func determineSlotID(dir string, useGit bool) (string, error) {
	if useGit {
		cmd := exec.Command("git", "-C", dir, "rev-parse", "--short=7", "HEAD")
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git rev-parse: %w", err)
		}
		sha := strings.TrimSpace(string(out))
		if len(sha) != 7 {
			return "", fmt.Errorf("unexpected short SHA %q", sha)
		}
		return sha, nil
	}
	return time.Now().UTC().Format("20060102150405"), nil
}

// suffixIfTaken returns base unchanged when taken(base) is false.
// Otherwise it appends -2, -3, ... until an unused name is found.
func suffixIfTaken(base string, taken func(string) bool) string {
	if !taken(base) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !taken(candidate) {
			return candidate
		}
	}
}

// IsGitRepo reports whether dir is inside a git work tree.
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--is-inside-work-tree")
	return cmd.Run() == nil
}
