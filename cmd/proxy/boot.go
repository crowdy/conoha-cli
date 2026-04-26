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

var bootCmd = &cobra.Command{
	Use:   "boot <server>",
	Short: "Install and start conoha-proxy on the server",
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

		fmt.Fprintf(os.Stderr, "==> Booting conoha-proxy on %s (%s)\n", ctx.Server.Name, ctx.IP)
		script := proxypkg.BootScript(proxypkg.BootParams{
			Email: email, Image: image, DataDir: dataDir, Container: container,
		})
		code, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("boot script: %w", err)
		}
		if code != 0 {
			return fmt.Errorf("boot script exited with %d", code)
		}

		// #175: confirm the container actually came up healthy. `docker run -d`
		// returns 0 the instant containerd creates the container, well before
		// the entrypoint has bound any ports — a crash loop is indistinguishable
		// from a healthy container at this stage. Polling here means same-class
		// regressions (file caps, sysctl drift, image entrypoint changes) fail
		// loudly on first run rather than only during a manual log read.
		waitTimeout, _ := cmd.Flags().GetDuration("wait-timeout")
		if waitTimeout > 0 {
			fmt.Fprintf(os.Stderr, "==> Waiting for proxy to become healthy (up to %s)\n", waitTimeout)
			// Discard remote stderr noise from the polling curls; healthcheck
			// errors are already aggregated and surfaced by WaitForHealthy.
			exec := &proxypkg.SSHExecutor{Client: ctx.Client, Stderr: io.Discard}
			if hcErr := proxypkg.WaitForHealthy(exec, container, dataDir, waitTimeout, nil, proxypkg.HealthcheckOptions{}); hcErr != nil {
				return hcErr
			}
		}
		fmt.Fprintln(os.Stderr, "Boot complete.")
		return nil
	},
}

func init() {
	addSSHFlags(bootCmd)
	bootCmd.Flags().String("acme-email", "", "email for Let's Encrypt registration (required)")
	bootCmd.Flags().String("image", DefaultImage, "conoha-proxy docker image")
	bootCmd.Flags().String("data-dir", DefaultDataDir, "host data directory")
	bootCmd.Flags().String("container", DefaultContainer, "docker container name")
	bootCmd.Flags().Duration("wait-timeout", 30*time.Second, "max wait for container to report healthy (0 disables the check)")
	Cmd.AddCommand(bootCmd)
}
