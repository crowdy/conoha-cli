package proxy

import (
	"fmt"
	"os"

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
	Cmd.AddCommand(bootCmd)
}
