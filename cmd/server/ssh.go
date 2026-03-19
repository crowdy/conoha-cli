package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/model"
)

func init() {
	sshCmd.Flags().StringP("user", "l", "root", "SSH user")
	sshCmd.Flags().StringP("port", "p", "22", "SSH port")
	sshCmd.Flags().StringP("identity", "i", "", "SSH private key path (overrides auto-detection)")
}

var sshCmd = &cobra.Command{
	Use:   "ssh <id|name> [command...]",
	Short: "SSH into a server",
	Long:  "Connect to a server via SSH. Remaining arguments are passed as the remote command.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		compute, err := getComputeAPI(cmd)
		if err != nil {
			return err
		}

		s, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}

		ip, err := getServerIP(s)
		if err != nil {
			return err
		}

		user, _ := cmd.Flags().GetString("user")
		port, _ := cmd.Flags().GetString("port")
		identity, _ := cmd.Flags().GetString("identity")

		if identity == "" {
			identity = resolveKeyPath(s.KeyName)
		}

		sshArgs := []string{"ssh"}
		sshArgs = append(sshArgs, "-l", user)
		if port != "22" {
			sshArgs = append(sshArgs, "-p", port)
		}
		if identity != "" {
			sshArgs = append(sshArgs, "-i", identity)
		}
		sshArgs = append(sshArgs, ip)
		sshArgs = append(sshArgs, args[1:]...)

		sshPath, err := exec.LookPath("ssh")
		if err != nil {
			return fmt.Errorf("ssh not found in PATH: %w", err)
		}

		// NOTE: syscall.Exec replaces the process (Unix only).
		// Windows is not supported for this command.
		return syscall.Exec(sshPath, sshArgs, os.Environ())
	},
}

// getServerIP extracts the best IP address from server addresses.
// Prefers floating IPv4 over fixed IPv4.
func getServerIP(s *model.Server) (string, error) {
	var fixedIP, floatingIP string

	for _, addrs := range s.Addresses {
		for _, a := range addrs {
			if a.Version != 4 {
				continue
			}
			switch a.Type {
			case "floating":
				floatingIP = a.Addr
			case "fixed":
				if fixedIP == "" {
					fixedIP = a.Addr
				}
			}
		}
	}

	if floatingIP != "" {
		return floatingIP, nil
	}
	if fixedIP != "" {
		return fixedIP, nil
	}
	return "", fmt.Errorf("no IPv4 address found for server %s", s.Name)
}

// resolveKeyPath returns the SSH private key path for a server's key name.
// Returns empty string if no key is associated or the file doesn't exist.
func resolveKeyPath(keyName string) string {
	if keyName == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	keyPath := filepath.Join(home, ".ssh", "conoha_"+keyName)
	if _, err := os.Stat(keyPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: key file %s not found, connecting without key\n", keyPath)
		return ""
	}
	return keyPath
}
