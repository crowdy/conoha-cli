package app

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

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
