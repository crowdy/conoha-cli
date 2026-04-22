package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

func init() {
	addAppFlags(rollbackCmd)
	rollbackCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	rollbackCmd.Flags().Int("drain-ms", 0, "drain window for the swapped-back target (0 = proxy default)")
	AddModeFlags(rollbackCmd)
}

func noProxyRollbackError(app string) error {
	return fmt.Errorf(
		"rollback is not supported in no-proxy mode. Deploy a previous revision instead: "+
			"git checkout <rev> && conoha app deploy --no-proxy --app-name %s <server>", app)
}

var rollbackCmd = &cobra.Command{
	Use:   "rollback <server>",
	Short: "Swap back to the previous target (within the drain window)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		noProxyFlag, _ := cmd.Flags().GetBool("no-proxy")
		if noProxyFlag {
			appName, _ := cmd.Flags().GetString("app-name")
			if appName == "" {
				return fmt.Errorf("--app-name is required with --no-proxy")
			}
			return noProxyRollbackError(appName)
		}
		pf, err := config.LoadProjectFile(config.ProjectFileName)
		if err != nil {
			return err
		}
		if err := pf.Validate(); err != nil {
			return err
		}
		sshClient, s, ip, err := connectToServer(cmd, args[0])
		if err != nil {
			return err
		}
		defer func() { _ = sshClient.Close() }()

		mode, err := ResolveMode(cmd, sshClient, pf.Name, args[0])
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return notInitializedError(pf.Name, args[0], ModeProxy)
			}
			return err
		}
		if mode == ModeNoProxy {
			return noProxyRollbackError(pf.Name)
		}

		dataDir, _ := cmd.Flags().GetString("data-dir")
		admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: sshClient}, proxy.SocketPath(dataDir))
		drainMs, _ := cmd.Flags().GetInt("drain-ms")
		fmt.Fprintf(os.Stderr, "==> Rolling back %q on %s (%s)\n", pf.Name, s.Name, ip)
		updated, err := admin.Rollback(pf.Name, drainMs)
		if err != nil {
			if errors.Is(err, proxypkg.ErrNoDrainTarget) {
				return fmt.Errorf("drain window has closed — redeploy the previous slot (git SHA) instead")
			}
			return err
		}
		active := "<unknown>"
		if updated.ActiveTarget != nil {
			active = updated.ActiveTarget.URL
		}
		fmt.Fprintf(os.Stderr, "Rollback complete. active=%s phase=%s\n", active, updated.Phase)
		return nil
	},
}
