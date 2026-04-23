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

Proxy mode: compose down for every slot + accessory, rm -rf work dir,
DELETE /v1/services/<name>, re-upsert the service, then run a fresh
blue/green deploy.

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

		mode, modeErr := ResolveModeFromCtx(cmd, ctx)
		if modeErr != nil && !errors.Is(modeErr, ErrNoMarker) {
			return modeErr
		}

		// If no marker but conoha.yml is present, treat as legacy proxy app
		// (mirrors destroy.go's fallback so we don't leak proxy registrations).
		legacyProxy := false
		if errors.Is(modeErr, ErrNoMarker) {
			if pf, pfErr := config.LoadProjectFile(config.ProjectFileName); pfErr == nil && pf.Validate() == nil {
				legacyProxy = true
				fmt.Fprintln(os.Stderr, "==> No mode marker; treating as legacy proxy deployment")
				mode = ModeProxy
			} else {
				return notInitializedError(ctx.AppName, serverID, "")
			}
		}

		yes, _ := cmd.Flags().GetBool("yes")
		if !yes {
			ok, err := prompt.Confirm(fmt.Sprintf("Reset app %q on %s? All slot data + env files will be deleted before the fresh deploy.", ctx.AppName, ctx.Server.Name))
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
			return fmt.Errorf("init phase: %w", err)
		}

		// --- Phase 3: deploy ---
		fmt.Fprintln(os.Stderr, "==> Phase 3/3: deploying")
		// runProxyDeploy and runNoProxyDeploy open their own SSH connections
		// via connectToServer; close ours first so we don't hold two sessions.
		_ = ctx.Client.Close()
		switch mode {
		case ModeProxy:
			if err := runProxyDeploy(cmd, serverID); err != nil {
				return fmt.Errorf("deploy phase: %w", err)
			}
		case ModeNoProxy:
			// runNoProxyDeploy requires an SSH client from connectToServer;
			// replicate the no-proxy dispatch path with this cmd.
			if err := runResetNoProxyDeploy(cmd, serverID, ctx.AppName); err != nil {
				return fmt.Errorf("deploy phase: %w", err)
			}
		default:
			return fmt.Errorf("unreachable: mode %q after init", mode)
		}

		fmt.Fprintf(os.Stderr, "App %q reset complete.\n", ctx.AppName)
		return nil
	},
}

// resetDestroy runs the same cleanup steps as 'app destroy' without the
// prompt (the reset prompt already covered it) and without tearing down the
// caller's SSH client.
func resetDestroy(cmd *cobra.Command, ctx *appContext, mode Mode, legacyProxy bool) error {
	script := generateDestroyScript(ctx.AppName)
	exitCode, err := internalssh.RunScript(ctx.Client, script, nil, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("destroy script: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("destroy script exited with code %d", exitCode)
	}

	if mode != ModeProxy && !legacyProxy {
		return nil
	}

	dataDir, _ := cmd.Flags().GetString("data-dir")
	if dataDir == "" {
		dataDir = proxy.DefaultDataDir
	}
	admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: ctx.Client}, proxy.SocketPath(dataDir))
	pf, pfErr := config.LoadProjectFile(config.ProjectFileName)
	if pfErr != nil || pf.Validate() != nil {
		// Without conoha.yml we can't name the service. Log + continue —
		// the fresh init in Phase 2 will fail anyway if conoha.yml is missing.
		return nil
	}
	if err := admin.Delete(pf.Name); err != nil && !errors.Is(err, proxypkg.ErrNotFound) {
		fmt.Fprintf(os.Stderr, "warning: proxy delete %s: %v\n", pf.Name, err)
	} else if err == nil {
		fmt.Fprintf(os.Stderr, "==> Deregistered %q from proxy\n", pf.Name)
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
