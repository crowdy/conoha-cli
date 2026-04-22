package app

import (
	"errors"
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
	AddModeFlags(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status <id|name>",
	Short: "Show app container status",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		mode, err := ResolveModeFromCtx(cmd, ctx)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return notInitializedError(ctx.AppName, ctx.ServerID, "")
			}
			return err
		}

		var psCmd string
		if mode == ModeProxy {
			psCmd = buildStatusCmdForProxy(ctx.AppName)
		} else {
			psCmd = buildStatusCmdForNoProxy(ctx.AppName)
		}
		// A non-nil err from RunCommand is an SSH-layer failure (auth, transport,
		// channel close) — not something `compose ps` could produce by having no
		// containers to show. Surface it as a non-zero exit so scripts that check
		// $? don't silently get wrong information. Remote non-zero exit codes are
		// downgraded to a warning because an empty project is a normal state.
		code, err := internalssh.RunCommand(ctx.Client, psCmd, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("status (compose ps via SSH): %w", err)
		}
		if code != 0 {
			fmt.Fprintf(os.Stderr, "warning: compose ps exited with code %d\n", code)
		}

		if mode != ModeProxy {
			return nil
		}

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

func buildStatusCmdForProxy(app string) string {
	// Enumerate slot projects via container labels rather than
	// 'docker compose ls --format "{{.Name}}"', which fails silently on
	// Docker Compose v5 hosts and would produce an empty listing (#114).
	return fmt.Sprintf(
		`for p in $(%[1]s); do `+
			`echo "--- compose project: ${p} ---"; `+
			`docker compose -p "${p}" ps; `+
			`done`,
		composeProjectEnumPipeline(app))
}

func buildStatusCmdForNoProxy(app string) string {
	return fmt.Sprintf("cd /opt/conoha/%s && docker compose ps", app)
}
