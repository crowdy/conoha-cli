package skill

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	cerrors "github.com/crowdy/conoha-cli/internal/errors"
	"github.com/crowdy/conoha-cli/internal/prompt"
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
	Cmd.AddCommand(updateCmd)
	Cmd.AddCommand(removeCmd)
}

func defaultSkillBase() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".claude", "skills"), nil
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

func runUpdate(baseDir string) error {
	skillDir := filepath.Join(baseDir, skillName)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return &cerrors.ValidationError{Message: "not installed, use 'conoha skill install'"}
	}

	gitDir := filepath.Join(skillDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return &cerrors.ValidationError{Message: "not a git repository, remove and reinstall"}
	}

	cmd := exec.Command("git", "-C", skillDir, "pull")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return &cerrors.NetworkError{Err: fmt.Errorf("git pull failed: %w", err)}
	}

	fmt.Fprintln(os.Stderr, "Updated conoha-cli-skill successfully.")
	return nil
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update conoha-cli-skill to latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		base, err := defaultSkillBase()
		if err != nil {
			return err
		}
		return runUpdate(base)
	},
}

func runRemove(baseDir string) error {
	skillDir := filepath.Join(baseDir, skillName)
	if _, err := os.Stat(skillDir); os.IsNotExist(err) {
		return &cerrors.ValidationError{Message: "not installed"}
	}

	ok, err := prompt.Confirm("Remove conoha-cli-skill?")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "Cancelled.")
		return nil
	}

	if err := os.RemoveAll(skillDir); err != nil {
		return fmt.Errorf("failed to remove: %w", err)
	}

	fmt.Fprintln(os.Stderr, "Removed conoha-cli-skill successfully.")
	return nil
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove conoha-cli-skill",
	RunE: func(cmd *cobra.Command, args []string) error {
		base, err := defaultSkillBase()
		if err != nil {
			return err
		}
		return runRemove(base)
	},
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install conoha-cli-skill for Claude Code",
	RunE: func(cmd *cobra.Command, args []string) error {
		base, err := defaultSkillBase()
		if err != nil {
			return err
		}
		return runInstall(base)
	},
}
