package app

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	"github.com/crowdy/conoha-cli/internal/prompt"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(destroyCmd)
	destroyCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	destroyCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	AddModeFlags(destroyCmd)
}

var destroyCmd = &cobra.Command{
	Use:   "destroy <id|name>",
	Short: "Destroy an app and deregister it from conoha-proxy",
	Long:  "Stop every slot's containers, remove the app work directory, remove accessories, and deregister the service from conoha-proxy.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			ok, err := prompt.Confirm(fmt.Sprintf("Destroy app %q on %s? All data will be deleted.", ctx.AppName, ctx.Server.Name))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
		}

		// Resolve mode BEFORE running the destroy script, because the
		// script removes the .conoha-mode marker as part of rm -rf.
		mode, modeErr := ResolveMode(cmd, ctx.Client, ctx.AppName)
		if modeErr != nil && !errors.Is(modeErr, ErrNoMarker) {
			return modeErr
		}

		script := generateDestroyScript(ctx.AppName)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("destroy failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("destroy exited with code %d", exitCode)
		}

		if mode == ModeProxy {
			dataDir, _ := cmd.Flags().GetString("data-dir")
			if dataDir == "" {
				dataDir = proxy.DefaultDataDir
			}
			admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: ctx.Client}, proxy.SocketPath(dataDir))
			pf, pfErr := config.LoadProjectFile(config.ProjectFileName)
			if pfErr == nil && pf.Validate() == nil {
				if err := admin.Delete(pf.Name); err != nil && !errors.Is(err, proxypkg.ErrNotFound) {
					fmt.Fprintf(os.Stderr, "warning: proxy delete %s: %v\n", pf.Name, err)
				} else if err == nil {
					fmt.Fprintf(os.Stderr, "==> Deregistered %q from proxy\n", pf.Name)
				}
			}
		}

		fmt.Fprintf(os.Stderr, "App %q destroyed.\n", ctx.AppName)
		return nil
	},
}

func generateDestroyScript(appName string) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

APP_NAME="%s"
APP_DIR="/opt/conoha/${APP_NAME}"

echo "==> Stopping all compose projects for ${APP_NAME}..."
for project in $(docker compose ls -a --format '{{.Name}}' 2>/dev/null | grep -E "^${APP_NAME}(-|$)" || true); do
    echo "    - ${project}"
    docker compose -p "${project}" down --remove-orphans 2>/dev/null || true
done

echo "==> Removing app directory..."
rm -rf "${APP_DIR}"

# Legacy cleanup from v0.1.x (safe to attempt):
rm -rf "/opt/conoha/${APP_NAME}.git" 2>/dev/null || true
rm -f  "/opt/conoha/${APP_NAME}.env.server" 2>/dev/null || true

echo "==> Done."
`, appName))
}
