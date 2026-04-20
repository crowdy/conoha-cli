package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(statusCmd)
	statusCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
}

var statusCmd = &cobra.Command{
	Use:   "status <id|name>",
	Short: "Show app container status and proxy phase",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		// Print docker compose state across all slot projects for this app.
		psCmd := fmt.Sprintf(
			`for p in $(docker compose ls -a --format '{{.Name}}' 2>/dev/null | grep -E "^%[1]s(-|$)" || true); do `+
				`echo "--- compose project: ${p} ---"; `+
				`docker compose -p "${p}" ps; `+
				`done`,
			ctx.AppName)
		if _, err := internalssh.RunCommand(ctx.Client, psCmd, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "warning: compose ps: %v\n", err)
		}

		// Enrich with proxy service state if conoha.yml is present.
		pf, pfErr := config.LoadProjectFile(config.ProjectFileName)
		if pfErr == nil && pf.Validate() == nil {
			dataDir, _ := cmd.Flags().GetString("data-dir")
			if dataDir == "" {
				dataDir = proxy.DefaultDataDir
			}
			admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: ctx.Client}, proxy.SocketPath(dataDir))
			if svc, err := admin.Get(pf.Name); err == nil {
				fmt.Fprintf(os.Stderr, "\n==> Proxy service %q: phase=%s tls=%s\n", svc.Name, svc.Phase, svc.TLSStatus)
				if svc.ActiveTarget != nil {
					fmt.Fprintf(os.Stderr, "    active:   %s\n", svc.ActiveTarget.URL)
				}
				if svc.DrainingTarget != nil {
					fmt.Fprintf(os.Stderr, "    draining: %s\n", svc.DrainingTarget.URL)
				}
				if svc.DrainDeadline != nil {
					fmt.Fprintf(os.Stderr, "    drain deadline: %s\n", svc.DrainDeadline.Format("2006-01-02 15:04:05 MST"))
				}
			} else {
				fmt.Fprintf(os.Stderr, "\n==> Proxy service %q: (error: %v)\n", pf.Name, err)
			}
		}
		return nil
	},
}
