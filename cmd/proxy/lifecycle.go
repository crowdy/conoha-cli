package proxy

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

// runScriptOnServer is the common shape of every lifecycle subcommand:
// connect → run script → close.
func runScriptOnServer(cmd *cobra.Command, args []string, desc string, scriptFn func(container string) []byte) error {
	container, _ := cmd.Flags().GetString("container")
	ctx, err := connect(cmd, args)
	if err != nil {
		return err
	}
	defer func() { _ = ctx.Client.Close() }()
	fmt.Fprintf(os.Stderr, "==> %s on %s (%s)\n", desc, ctx.Server.Name, ctx.IP)
	code, err := internalssh.RunScript(ctx.Client, scriptFn(container), nil, os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("script exited with %d", code)
	}
	return nil
}

var rebootCmd = &cobra.Command{
	Use:   "reboot <server>",
	Short: "Pull the latest image and restart conoha-proxy",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		email, _ := cmd.Flags().GetString("acme-email")
		if email == "" {
			return fmt.Errorf("--acme-email is required")
		}
		image, _ := cmd.Flags().GetString("image")
		dataDir, _ := cmd.Flags().GetString("data-dir")
		container, _ := cmd.Flags().GetString("container")
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		fmt.Fprintf(os.Stderr, "==> Rebooting conoha-proxy on %s (%s)\n", ctx.Server.Name, ctx.IP)
		script := proxypkg.RebootScript(proxypkg.BootParams{
			Email: email, Image: image, DataDir: dataDir, Container: container,
		})
		code, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("reboot script exited with %d", code)
		}
		// Same healthy-gate as boot (#175): a reboot's docker run inherits
		// every gotcha that boot does, so the same check applies. If
		// anything, the gate matters MORE on reboot — the previous version
		// was working, and a silent unhealthy upgrade would mask a real
		// regression. --wait-timeout=0 is honored but discouraged here.
		waitTimeout, _ := cmd.Flags().GetDuration("wait-timeout")
		if waitTimeout > 0 {
			fmt.Fprintf(os.Stderr, "==> Waiting for proxy to become healthy (up to %s)\n", waitTimeout)
			exec := &proxypkg.SSHExecutor{Client: ctx.Client, Stderr: io.Discard}
			if hcErr := proxypkg.WaitForHealthy(exec, container, dataDir, waitTimeout, nil, proxypkg.HealthcheckOptions{}); hcErr != nil {
				return hcErr
			}
		}
		fmt.Fprintln(os.Stderr, "Reboot complete.")
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start <server>",
	Short: "Start the conoha-proxy container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScriptOnServer(cmd, args, "Starting conoha-proxy", proxypkg.StartScript)
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop <server>",
	Short: "Stop the conoha-proxy container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScriptOnServer(cmd, args, "Stopping conoha-proxy", proxypkg.StopScript)
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart <server>",
	Short: "Restart the conoha-proxy container",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScriptOnServer(cmd, args, "Restarting conoha-proxy", proxypkg.RestartScript)
	},
}

var removeCmd = &cobra.Command{
	Use:   "remove <server>",
	Short: "Remove the conoha-proxy container (volume is kept unless --purge)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		container, _ := cmd.Flags().GetString("container")
		dataDir, _ := cmd.Flags().GetString("data-dir")
		purge, _ := cmd.Flags().GetBool("purge")
		ctx, err := connect(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()
		fmt.Fprintf(os.Stderr, "==> Removing conoha-proxy on %s (purge=%v)\n", ctx.Server.Name, purge)
		code, err := internalssh.RunScript(ctx.Client, proxypkg.RemoveScript(container, dataDir, purge), nil, os.Stdout, os.Stderr)
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("remove script exited with %d", code)
		}
		return nil
	},
}

func init() {
	addSSHFlags(rebootCmd)
	rebootCmd.Flags().String("acme-email", "", "email for Let's Encrypt registration (required)")
	rebootCmd.Flags().String("image", DefaultImage, "conoha-proxy docker image")
	rebootCmd.Flags().String("data-dir", DefaultDataDir, "host data directory")
	rebootCmd.Flags().String("container", DefaultContainer, "docker container name")
	rebootCmd.Flags().Duration("wait-timeout", 30*time.Second, "max wait for container to report healthy (0 disables — discouraged: previous version was working)")

	for _, c := range []*cobra.Command{startCmd, stopCmd, restartCmd} {
		addSSHFlags(c)
		c.Flags().String("container", DefaultContainer, "docker container name")
	}

	addSSHFlags(removeCmd)
	removeCmd.Flags().String("container", DefaultContainer, "docker container name")
	removeCmd.Flags().String("data-dir", DefaultDataDir, "host data directory")
	removeCmd.Flags().Bool("purge", false, "also delete the host data directory")

	Cmd.AddCommand(rebootCmd, startCmd, stopCmd, restartCmd, removeCmd)
}
