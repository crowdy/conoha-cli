package app

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	"github.com/crowdy/conoha-cli/internal/prompt"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(resetCmd)
	resetCmd.Flags().Bool("yes", false, "skip confirmation prompt")
	resetCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	resetCmd.Flags().String("slot", "", "override slot ID for the fresh deploy (proxy mode)")
	AddModeFlags(resetCmd)
}

var resetCmd = &cobra.Command{
	Use:   "reset <server>",
	Short: "Destroy and re-deploy an app from a clean state",
	Long: `Reset performs destroy → init → deploy in one command. Use it when you
want to discard the current deployment state (slots, work dirs, env files)
and re-apply the current conoha.yml + repo from scratch.

Proxy mode order is DELETE /v1/services/<name> first, then compose
down for every slot + accessory, then rm -rf work dir. Dropping the
proxy registration before killing containers means in-flight requests
see 404 (no such service) for a moment rather than 502 (upstream dead)
for the duration of the teardown.

No-proxy mode: compose down, rm -rf work dir, re-init the marker,
then re-deploy.

No slot rollback window survives reset — the previous slot's drain
deadline is discarded as part of the destroy phase.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverID := args[0]
		ctx, err := connectToApp(cmd, args)
		if err != nil {
			return err
		}
		defer func() { _ = ctx.Client.Close() }()

		mode, legacyProxy, modeErr := ResolveAppModeWithLegacyFallback(cmd, ctx)
		if errors.Is(modeErr, ErrNoMarker) {
			return notInitializedError(ctx.AppName, serverID, "")
		}
		if modeErr != nil {
			return modeErr
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			ok, err := prompt.Confirm(resetConfirmationMessage(ctx.AppName, ctx.Server.Name, mode, legacyProxy))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
		}

		// --- Phase 1: destroy ---
		fmt.Fprintln(os.Stderr, "==> Phase 1/3: destroying current deployment")
		if err := resetDestroy(cmd, ctx, mode, legacyProxy); err != nil {
			return fmt.Errorf("destroy phase: %w", err)
		}

		// --- Phase 2: re-init ---
		fmt.Fprintln(os.Stderr, "==> Phase 2/3: re-initializing")
		if err := resetInit(cmd, ctx, mode, serverID); err != nil {
			// Phase 1 already removed containers + proxy registration. The app
			// is recoverable manually, but users won't know how. Spell it out.
			return fmt.Errorf("init phase: %w\n\nPhase 1 completed (containers destroyed, proxy deregistered). To recover, run:\n    conoha app init %s && conoha app deploy %s", err, serverID, serverID)
		}

		// --- Phase 3: deploy ---
		fmt.Fprintln(os.Stderr, "==> Phase 3/3: deploying")
		// runProxyDeploy and runNoProxyDeploy open their own SSH connections
		// via connectToServer; close ours first so we don't hold two sessions.
		_ = ctx.Client.Close()
		switch mode {
		case ModeProxy:
			if err := runProxyDeploy(cmd, serverID); err != nil {
				return fmt.Errorf("deploy phase: %w\n\nPhases 1-2 completed (service re-registered). To retry the deploy only, run:\n    conoha app deploy %s", err, serverID)
			}
		case ModeNoProxy:
			// runNoProxyDeploy requires an SSH client from connectToServer;
			// replicate the no-proxy dispatch path with this cmd.
			if err := runResetNoProxyDeploy(cmd, serverID, ctx.AppName); err != nil {
				return fmt.Errorf("deploy phase: %w\n\nPhases 1-2 completed. To retry the deploy only, run:\n    conoha app deploy --no-proxy --app-name %s %s", err, ctx.AppName, serverID)
			}
		default:
			return fmt.Errorf("unreachable: mode %q after init", mode)
		}

		fmt.Fprintf(os.Stderr, "App %q reset complete.\n", ctx.AppName)
		return nil
	},
}

// resetConfirmationMessage returns the prompt shown before a reset. It spells
// out the mode, the proxy-dereg side effect, and the loss of any open rollback
// window, so users don't fat-finger a destructive command expecting the
// rollback-window safety net they'd get from a plain deploy.
func resetConfirmationMessage(app, serverName string, mode Mode, legacyProxy bool) string {
	modeLabel := string(mode)
	if legacyProxy {
		modeLabel = "proxy (legacy, no marker)"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Reset app %q on %s (mode=%s)?", app, serverName, modeLabel)
	b.WriteString("\n  - all slots will be torn down (compose down + rm -rf /opt/conoha/")
	b.WriteString(app)
	b.WriteString(")")
	b.WriteString("\n  - env files (.env.server) will be deleted")
	if mode == ModeProxy || legacyProxy {
		b.WriteString("\n  - proxy registration will be dropped")
		b.WriteString("\n  - any pending rollback window will be discarded")
	}
	return b.String()
}

// resetDestroy runs the same cleanup steps as 'app destroy' without the
// prompt (the reset prompt already covered it) and without tearing down the
// caller's SSH client.
//
// Phase order: for proxy mode, DELETE /v1/services FIRST so the proxy stops
// routing to the about-to-be-killed containers; then tear down containers +
// work dir. Destroying containers first would leave the proxy pointing at
// dead upstreams (502s) for the duration of the SSH script.
func resetDestroy(cmd *cobra.Command, ctx *appContext, mode Mode, legacyProxy bool) error {
	if mode == ModeProxy || legacyProxy {
		dataDir, _ := cmd.Flags().GetString("data-dir")
		if dataDir == "" {
			dataDir = proxy.DefaultDataDir
		}
		admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: ctx.Client}, proxy.SocketPath(dataDir))
		pf, pfErr := config.LoadProjectFile(config.ProjectFileName)
		if pfErr == nil && pf.Validate() == nil {
			if err := admin.Delete(pf.Name); err != nil && !errors.Is(err, proxypkg.ErrNotFound) {
				fmt.Fprintf(os.Stderr, "warning: proxy delete %s: %v (continuing with teardown)\n", pf.Name, err)
			} else if err == nil {
				fmt.Fprintf(os.Stderr, "==> Deregistered %q from proxy (traffic now 404s)\n", pf.Name)
			}
		}
	}

	script := generateDestroyScript(ctx.AppName)
	exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("destroy script: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("destroy script exited with code %d", exitCode)
	}
	return nil
}

// resetInit re-runs the per-mode init body against the existing ctx.Client.
// Does not re-verify docker (the destroy phase already succeeded there).
func resetInit(cmd *cobra.Command, ctx *appContext, mode Mode, serverID string) error {
	switch mode {
	case ModeProxy:
		pf, err := config.LoadProjectFile(config.ProjectFileName)
		if err != nil {
			return err
		}
		if err := pf.Validate(); err != nil {
			return err
		}
		dataDir, _ := cmd.Flags().GetString("data-dir")
		if dataDir == "" {
			dataDir = proxy.DefaultDataDir
		}
		admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: ctx.Client}, proxy.SocketPath(dataDir))
		fmt.Fprintf(os.Stderr, "==> Re-registering service %q\n", pf.Name)
		if _, err := admin.Upsert(proxypkg.UpsertRequest{
			Name:         pf.Name,
			Hosts:        pf.Hosts,
			HealthPolicy: mapHealth(pf.Health),
		}); err != nil {
			return err
		}
		if err := WriteMarker(ctx.Client, pf.Name, ModeProxy); err != nil {
			return fmt.Errorf("write mode marker: %w", err)
		}
		return nil

	case ModeNoProxy:
		if err := WriteMarker(ctx.Client, ctx.AppName, ModeNoProxy); err != nil {
			return err
		}
		return nil

	default:
		return fmt.Errorf("unsupported mode %q", mode)
	}
}

// runResetNoProxyDeploy replicates the no-proxy half of runDeployDispatch
// without going through flag parsing (--no-proxy is not set on resetCmd by
// default; reset infers mode from the marker instead).
func runResetNoProxyDeploy(cmd *cobra.Command, serverID, appName string) error {
	sshClient, s, ip, err := connectToServer(cmd, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sshClient.Close() }()
	return runNoProxyDeploy(cmd, sshClient, s, ip, appName)
}
