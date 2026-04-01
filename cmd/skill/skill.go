package skill

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	cerrors "github.com/crowdy/conoha-cli/internal/errors"
)

const (
	skillRepo = "https://github.com/crowdy/conoha-cli-skill.git"
	skillName = "conoha-cli-skill"
)

// Cmd is the parent command for skill management.
var Cmd = &cobra.Command{
	Use:   "skill",
	Short: "Manage Claude Code skills",
}

func init() {
	Cmd.AddCommand(installCmd)
}

func defaultSkillBase() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude", "skills")
}

func runInstall(baseDir string) error {
	if _, err := exec.LookPath("git"); err != nil {
		return &cerrors.ValidationError{Message: "git is required to install skills"}
	}

	skillDir := filepath.Join(baseDir, skillName)
	if _, err := os.Stat(skillDir); err == nil {
		return &cerrors.ValidationError{Message: "already installed, use 'conoha skill update'"}
	}

	cmd := exec.Command("git", "clone", skillRepo, skillDir)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return &cerrors.NetworkError{Err: fmt.Errorf("git clone failed: %w", err)}
	}

	fmt.Fprintln(os.Stderr, "Installed conoha-cli-skill successfully.")
	return nil
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install conoha-cli-skill for Claude Code",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInstall(defaultSkillBase())
	},
}
