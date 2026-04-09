package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/internal/prompt"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(resetCmd)
	resetCmd.Flags().Bool("yes", false, "skip confirmation prompt")
}

var resetCmd = &cobra.Command{
	Use:   "reset <id|name>",
	Short: "Destroy and redeploy an app from scratch",
	Long:  "Equivalent to running destroy + init + deploy in sequence. Stops containers, removes all app data, re-initializes the environment, and deploys the current directory.\n\nNote: server-side environment variables (set via 'app env set') will be lost.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			ok, err := prompt.Confirm(fmt.Sprintf("Reset app %q on %s? This will destroy all data and redeploy.", ctx.AppName, ctx.Server.Name))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
		}

		// Step 0: Stop all apps on the server
		fmt.Fprintln(os.Stderr, "==> Stopping all apps on server...")
		stopAllScript := generateStopAllScript()
		exitCode, err := internalssh.RunScript(ctx.Client, stopAllScript, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("stop all apps failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("stop all apps exited with code %d", exitCode)
		}

		// Step 1: Destroy
		fmt.Fprintln(os.Stderr, "==> Destroying app...")
		script := generateDestroyScript(ctx.AppName)
		exitCode, err = internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("destroy failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("destroy exited with code %d", exitCode)
		}

		// Step 2: Init
		fmt.Fprintln(os.Stderr, "==> Re-initializing app...")
		script = generateInitScript(ctx.AppName)
		exitCode, err = internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("init failed: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("init exited with code %d", exitCode)
		}

		// Step 3: Deploy
		fmt.Fprintln(os.Stderr, "==> Deploying app...")
		if err := deployApp(ctx); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "App %q reset complete.\n", ctx.AppName)
		return nil
	},
}

func generateStopAllScript() []byte {
	return []byte(`#!/bin/bash
set -euo pipefail

echo "==> Stopping all app containers on this server..."
for dir in /opt/conoha/*/; do
    [ -d "$dir" ] || continue
    # skip .git bare repos
    case "$dir" in *.git/) continue;; esac
    if [ -f "$dir/docker-compose.yml" ] || [ -f "$dir/docker-compose.yaml" ] || \
       [ -f "$dir/compose.yml" ] || [ -f "$dir/compose.yaml" ] || \
       [ -f "$dir/conoha-docker-compose.yml" ] || [ -f "$dir/conoha-docker-compose.yaml" ]; then
        echo "Stopping containers in $dir..."
        cd "$dir"
        docker compose down --remove-orphans 2>/dev/null || true
    fi
done
echo "==> All apps stopped."
`)
}
