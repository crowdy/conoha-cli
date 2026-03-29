package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/prompt"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(destroyCmd)
}

var destroyCmd = &cobra.Command{
	Use:   "destroy <id|name>",
	Short: "Destroy an app and all its data",
	Long:  "Stop containers, remove work directory, git repository, and environment file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		ok, err := prompt.Confirm(fmt.Sprintf("Destroy app %q on %s? All data will be deleted.", ctx.AppName, ctx.Server.Name))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return nil
		}

		script := generateDestroyScript(ctx.AppName)
		exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("destroy failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("destroy exited with code %d", exitCode)
		}

		fmt.Fprintf(os.Stderr, "App %q destroyed.\n", ctx.AppName)
		return nil
	},
}

func generateDestroyScript(appName string) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

APP_NAME="%s"
WORK_DIR="/opt/conoha/${APP_NAME}"
REPO_DIR="/opt/conoha/${APP_NAME}.git"
ENV_FILE="/opt/conoha/${APP_NAME}.env.server"

echo "==> Stopping containers..."
if [ -d "$WORK_DIR" ]; then
    cd "$WORK_DIR"
    docker compose down --remove-orphans 2>/dev/null || true
fi

echo "==> Removing work directory..."
rm -rf "$WORK_DIR"

echo "==> Removing git repository..."
rm -rf "$REPO_DIR"

echo "==> Removing environment file..."
rm -f "$ENV_FILE"

echo "==> Done."
`, appName))
}
