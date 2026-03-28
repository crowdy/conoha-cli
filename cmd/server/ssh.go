package server

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/spf13/cobra"

	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
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

		ip, err := internalssh.ServerIP(s)
		if err != nil {
			return err
		}

		user, _ := cmd.Flags().GetString("user")
		port, _ := cmd.Flags().GetString("port")
		identity, _ := cmd.Flags().GetString("identity")

		if identity == "" {
			identity = internalssh.ResolveKeyPath(s.KeyName)
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
