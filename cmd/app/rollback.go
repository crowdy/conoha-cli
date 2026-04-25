package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
)

func init() {
	addAppFlags(rollbackCmd)
	rollbackCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	rollbackCmd.Flags().Int("drain-ms", 0, "drain window for the swapped-back target (0 = proxy default)")
	rollbackCmd.Flags().String("target", "", "rollback only a single block: 'web' or an expose label (default: all blocks)")
	AddModeFlags(rollbackCmd)
}

// rollbackClient is the minimal proxy API surface used by rollback paths.
type rollbackClient interface {
	Rollback(name string, drainMs int) (*proxypkg.Service, error)
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
		target, _ := cmd.Flags().GetString("target")
		if err := validateRollbackTarget(pf, target); err != nil {
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

		if target != "" {
			return runRollbackSingle(admin, pf, target, drainMs, os.Stderr)
		}
		return runRollbackAll(admin, pf, drainMs, os.Stderr)
	},
}

// validateRollbackTarget enforces that --target, when set, names either
// `web` (root) or a declared expose label.
func validateRollbackTarget(pf *config.ProjectFile, target string) error {
	if target == "" || target == "web" {
		return nil
	}
	for i := range pf.Expose {
		if pf.Expose[i].Label == target {
			return nil
		}
	}
	valid := make([]string, 0, 1+len(pf.Expose))
	valid = append(valid, "web")
	for i := range pf.Expose {
		valid = append(valid, pf.Expose[i].Label)
	}
	return fmt.Errorf("unknown --target %q (valid: %s)", target, strings.Join(valid, ", "))
}

// runRollbackSingle rolls back exactly one block. 409 no_drain_target is
// surfaced as a typed user-facing error because the user explicitly asked
// for this one target; degrading to a warning would hide the failure.
func runRollbackSingle(admin rollbackClient, pf *config.ProjectFile, target string, drainMs int, w io.Writer) error {
	name := rollbackServiceName(pf, target)
	updated, err := admin.Rollback(name, drainMs)
	if err != nil {
		if errors.Is(err, proxypkg.ErrNoDrainTarget) {
			return fmt.Errorf("drain window has closed for %s — redeploy the previous slot (git SHA) instead", name)
		}
		return err
	}
	active := "<unknown>"
	if updated.ActiveTarget != nil {
		active = updated.ActiveTarget.URL
	}
	_, _ = fmt.Fprintf(w, "Rollback complete for %s. active=%s phase=%s\n", name, active, updated.Phase)
	return nil
}

// runRollbackAll rolls back root first, then expose blocks in reverse
// declaration order — the inverse of the phase-3 deploy order (spec §3.3).
// 409 no_drain_target on any one service degrades to a stderr warning and
// the loop continues, so a drained-out sub-host does not block rolling the
// rest of the app. Non-409 errors are also logged but recorded as the
// final return value, so scripts checking $? still see a non-zero exit.
func runRollbackAll(admin rollbackClient, pf *config.ProjectFile, drainMs int, w io.Writer) error {
	order := rollbackServiceOrder(pf)
	var firstErr error
	for _, name := range order {
		svc, err := admin.Rollback(name, drainMs)
		if err != nil {
			if errors.Is(err, proxypkg.ErrNoDrainTarget) {
				_, _ = fmt.Fprintf(w, "warning: drain window expired for %s; skipping\n", name)
				continue
			}
			_, _ = fmt.Fprintf(w, "warning: rollback %s: %v\n", name, err)
			if firstErr == nil {
				firstErr = fmt.Errorf("rollback %s: %w", name, err)
			}
			continue
		}
		active := "<unknown>"
		if svc.ActiveTarget != nil {
			active = svc.ActiveTarget.URL
		}
		_, _ = fmt.Fprintf(w, "Rollback complete for %s. active=%s phase=%s\n", name, active, svc.Phase)
	}
	return firstErr
}

// rollbackServiceName returns the proxy service name for a single-target
// rollback. `web` maps to the root service; any other string is treated as
// an expose label. Caller must have already validated the label.
func rollbackServiceName(pf *config.ProjectFile, target string) string {
	if target == "web" {
		return pf.Name
	}
	return exposeServiceName(pf.Name, target)
}

// rollbackServiceOrder returns the reverse-of-deploy order: root first,
// then expose blocks in reverse declaration order. Only blue/green
// participants are included — blue_green:false expose blocks are accessory-
// style (they never rotate with a slot) and have no previous target to
// swap back to.
func rollbackServiceOrder(pf *config.ProjectFile) []string {
	out := []string{pf.Name}
	for i := len(pf.Expose) - 1; i >= 0; i-- {
		b := &pf.Expose[i]
		if b.BlueGreen != nil && !*b.BlueGreen {
			continue
		}
		out = append(out, exposeServiceName(pf.Name, b.Label))
	}
	return out
}
