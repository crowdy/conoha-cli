package app

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	"github.com/crowdy/conoha-cli/internal/model"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(deployCmd)
	deployCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	deployCmd.Flags().String("slot", "", "override slot ID (default: git short SHA or timestamp). Must match [a-z0-9][a-z0-9-]{0,63}. Reusing an existing slot removes its work dir before re-extracting; pending drain-teardowns for the same slot will auto-skip")
	AddModeFlags(deployCmd)
}

var deployCmd = &cobra.Command{
	Use:   "deploy <server>",
	Short: "Deploy the current directory via conoha-proxy blue/green",
	Long: `Archive the current directory, upload via SSH, start the web container
in a new compose slot on a dynamic port, then ask conoha-proxy to probe and
swap. The previous slot is torn down after the drain window.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeployDispatch(cmd, args[0])
	},
}

// runDeployDispatch resolves mode (flag override + server marker) and calls
// the proxy or no-proxy deploy path.
func runDeployDispatch(cmd *cobra.Command, serverID string) error {
	noProxyFlag, _ := cmd.Flags().GetBool("no-proxy")

	if noProxyFlag {
		appName, _ := cmd.Flags().GetString("app-name")
		if appName == "" {
			return fmt.Errorf("--app-name is required with --no-proxy")
		}
		if err := internalssh.ValidateAppName(appName); err != nil {
			return err
		}
		sshClient, s, ip, err := connectToServer(cmd, serverID)
		if err != nil {
			return err
		}
		defer func() { _ = sshClient.Close() }()
		mode, err := ResolveMode(cmd, sshClient, appName, serverID)
		if err != nil {
			if errors.Is(err, ErrNoMarker) {
				return notInitializedError(appName, serverID, ModeNoProxy)
			}
			return err
		}
		if mode != ModeNoProxy {
			return formatModeConflictError(appName, serverID, mode, ModeNoProxy)
		}
		return runNoProxyDeploy(cmd, sshClient, s, ip, appName)
	}

	return runProxyDeploy(cmd, serverID)
}

// runNoProxyDeploy uploads the working tree to /opt/conoha/<app>/ and runs
// 'docker compose -p <app> up -d --build'. No proxy upsert, no slot.
func runNoProxyDeploy(cmd *cobra.Command, sshClient *ssh.Client, s *model.Server, ip, appName string) error {
	fmt.Fprintf(os.Stderr, "==> Deploying %q to %s (%s) in no-proxy mode\n", appName, s.Name, ip)

	patterns, err := loadIgnorePatterns(".")
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := createTarGz(".", patterns, &buf); err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	workDir := "/opt/conoha/" + appName
	if err := runRemote(sshClient, buildNoProxyUploadCmd(workDir), &buf); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	pf := &config.ProjectFile{}
	composeFile, err := pf.ResolveComposeFile(".")
	if err != nil {
		return err
	}

	if err := runRemote(sshClient, buildNoProxyDeployCmd(workDir, appName, composeFile), nil); err != nil {
		return fmt.Errorf("compose up: %w", err)
	}
	fmt.Fprintln(os.Stderr, "Deploy complete.")
	return nil
}

// buildNoProxyUploadCmd extracts the incoming tar archive into the app work
// directory. It removes the previous deploy's merged .env (if any) before
// extracting so the tar becomes authoritative for repo-level .env content;
// the deploy command then overlays /opt/conoha/<app>.env.server on top.
// Other sibling files (e.g. named-volume binds) are preserved.
// Caller MUST pre-validate app via internalssh.ValidateAppName.
func buildNoProxyUploadCmd(workDir string) string {
	return fmt.Sprintf(
		"mkdir -p '%[1]s' && rm -f '%[1]s/.env' && tar xzf - -C '%[1]s'",
		workDir)
}

// buildNoProxyDeployCmd brings the flat-layout compose project up in place.
// The compose project name equals the app name (no slot suffix).
//
// Env merge (v0.2+, spec 2026-04-23-app-env-redesign.md §2.3):
// appends /opt/conoha/<app>/.env.server (new canonical location) to
// <workDir>/.env. When that file is absent but the v0.1.x path
// /opt/conoha/<app>.env.server exists, the legacy path is used and a
// deprecation warning is emitted to stderr. `app env migrate` moves
// users forward.
//
// Because buildNoProxyUploadCmd cleared any prior merged .env before tar
// extraction, each deploy starts from the repo's committed .env (if any)
// and re-overlays the current .env.server. `app env unset` therefore takes
// effect on the next deploy.
//
// Caller MUST pre-validate app via internalssh.ValidateAppName.
// composeFile is defensively single-quoted — today it comes from the
// ResolveComposeFile whitelist, but quoting hardens against future callers.
func buildNoProxyDeployCmd(workDir, app, composeFile string) string {
	newEnv := envFilePath(app)
	legacyEnv := legacyEnvFilePath(app)
	return fmt.Sprintf(
		"cd '%s' && { "+
			"touch .env; "+
			"if [ -s '%s' ]; then "+
			"    printf '\\n' >> .env && cat '%s' >> .env; "+
			"elif [ -s '%s' ]; then "+
			"    echo 'warning: merging legacy env file %s (run conoha app env migrate <server> to move it)' >&2; "+
			"    printf '\\n' >> .env && cat '%s' >> .env; "+
			"fi; "+
			"} && docker compose -p %s -f '%s' up -d --build",
		workDir, newEnv, newEnv, legacyEnv, legacyEnv, legacyEnv, app, composeFile)
}

func runProxyDeploy(cmd *cobra.Command, serverID string) error {
	pf, err := config.LoadProjectFile(config.ProjectFileName)
	if err != nil {
		return err
	}
	if err := pf.Validate(); err != nil {
		return err
	}
	composeFile, err := pf.ResolveComposeFile(".")
	if err != nil {
		return err
	}
	if err := pf.ValidateAgainstCompose(composeFile); err != nil {
		return err
	}

	sshClient, s, ip, err := connectToServer(cmd, serverID)
	if err != nil {
		return err
	}
	defer func() { _ = sshClient.Close() }()

	// Mode dispatch parity: reject if this app was initialized in no-proxy mode.
	// Absent marker falls through to the existing "service not found on proxy" path.
	mode, err := ResolveMode(cmd, sshClient, pf.Name, serverID)
	if err != nil && !errors.Is(err, ErrNoMarker) {
		return err
	}
	if mode == ModeNoProxy {
		return formatModeConflictError(pf.Name, serverID, mode, ModeProxy)
	}

	dataDir, _ := cmd.Flags().GetString("data-dir")
	admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: sshClient}, proxy.SocketPath(dataDir))
	ops := newSSHDeployOps(sshClient, admin)

	slotOverride, _ := cmd.Flags().GetString("slot")

	return runProxyDeployState(proxyDeployParams{
		ProjectFile:  pf,
		ComposeFile:  composeFile,
		ServerID:     serverID,
		ServerName:   s.Name,
		ServerIP:     ip,
		SlotOverride: slotOverride,
		Archive:      makeArchiveOfCwd,
	}, ops)
}

// proxyDeployParams is the input to the runProxyDeployState state machine.
// Pulled into a struct so tests can populate it without mocking the local
// filesystem / cobra.Command machinery.
type proxyDeployParams struct {
	ProjectFile  *config.ProjectFile
	ComposeFile  string
	ServerID     string
	ServerName   string
	ServerIP     string
	SlotOverride string
	// Archive builds the tar.gz of the deploy payload on demand. The
	// production code tarballs the cwd; tests pass a pre-made buffer.
	Archive func() (io.Reader, error)
}

// makeArchiveOfCwd is the production Archive func: load .dockerignore
// patterns and tar.gz the current directory.
func makeArchiveOfCwd() (io.Reader, error) {
	patterns, err := loadIgnorePatterns(".")
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := createTarGz(".", patterns, &buf); err != nil {
		return nil, fmt.Errorf("create archive: %w", err)
	}
	return &buf, nil
}

// runProxyDeployState drives the blue/green deploy state machine without
// any direct SSH or filesystem dependencies. Every side effect flows
// through ops, which tests substitute with a recording fake.
func runProxyDeployState(p proxyDeployParams, ops DeployOps) error {
	pf := p.ProjectFile

	// Service must exist — init registers it. Missing = user skipped init.
	if _, err := ops.Proxy().Get(pf.Name); err != nil {
		return fmt.Errorf("%w: %v", notInitializedError(pf.Name, p.ServerID, ModeProxy), err)
	}

	slot := p.SlotOverride
	if slot == "" {
		base, err := determineSlotID(".", IsGitRepo("."))
		if err != nil {
			return err
		}
		slot = suffixIfTaken(base, func(candidate string) bool {
			probe := buildComposeProjectExists(slotProjectName(pf.Name, candidate))
			code, _, _ := ops.Run(probe, nil)
			return code == 0
		})
	}
	if err := ValidateSlotID(slot); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "==> Deploying %q to %s (%s)\n", pf.Name, p.ServerName, p.ServerIP)
	fmt.Fprintf(os.Stderr, "==> Slot: %s (compose project: %s)\n", slot, slotProjectName(pf.Name, slot))

	// Upload archive to slot work dir.
	archive, err := p.Archive()
	if err != nil {
		return err
	}
	slotWork := fmt.Sprintf("/opt/conoha/%s/%s", pf.Name, slot)
	if err := runRemoteOps(ops, buildSlotUploadCmd(slotWork, ""), archive); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	// Write compose override into the slot dir.
	overrideContent := composeOverride(pf.Name, slot, pf.Web.Service, pf.Web.Port, len(pf.Accessories) > 0)
	overridePath := "conoha-override.yml"
	writeOverride := fmt.Sprintf("cat > '%s/%s' <<'EOF'\n%sEOF", slotWork, overridePath, overrideContent)
	if err := runRemoteOps(ops, writeOverride, nil); err != nil {
		return fmt.Errorf("write override: %w", err)
	}

	// First-run: bring up accessories (idempotent via existence probe).
	if len(pf.Accessories) > 0 {
		check := buildAccessoryExists(accessoryProjectName(pf.Name))
		code, _, _ := ops.Run(check, nil)
		if code != 0 {
			fmt.Fprintf(os.Stderr, "==> Starting accessories: %v\n", pf.Accessories)
			if err := runRemoteOps(ops, buildAccessoryUp(slotWork, accessoryProjectName(pf.Name), p.ComposeFile, pf.Accessories), nil); err != nil {
				return fmt.Errorf("accessory up: %w", err)
			}
		}
	}

	// Start the new slot's web service.
	fmt.Fprintf(os.Stderr, "==> Building and starting %s in new slot\n", pf.Web.Service)
	if err := runRemoteOps(ops, buildSlotComposeUp(slotWork, slotProjectName(pf.Name, slot), p.ComposeFile, overridePath, pf.Web.Service), nil); err != nil {
		return fmt.Errorf("compose up (slot): %w", err)
	}

	// Discover kernel-picked host port.
	containerName := fmt.Sprintf("%s-%s-%s", pf.Name, slot, pf.Web.Service)
	_, portOut, portErr := ops.Run(buildDockerPortCmd(containerName, pf.Web.Port), nil)
	if portErr != nil {
		tearDownSlotOps(ops, pf.Name, slot)
		return fmt.Errorf("docker port: %w", portErr)
	}
	hostPort, err := extractHostPort(string(portOut))
	if err != nil {
		tearDownSlotOps(ops, pf.Name, slot)
		return err
	}
	targetURL := fmt.Sprintf("http://127.0.0.1:%d", hostPort)
	fmt.Fprintf(os.Stderr, "==> Host port: %d. Calling proxy /deploy\n", hostPort)

	drainMs := 30000
	if pf.Deploy != nil && pf.Deploy.DrainMs > 0 {
		drainMs = pf.Deploy.DrainMs
	}

	// Call proxy /deploy. On 424 the proxy did not mutate state — tear down new slot.
	updated, err := ops.Proxy().Deploy(pf.Name, proxypkg.DeployRequest{TargetURL: targetURL, DrainMs: drainMs})
	if err != nil {
		tearDownSlotOps(ops, pf.Name, slot)
		return err
	}

	// Read old slot pointer (empty on first deploy), then update to current.
	ptrPath := fmt.Sprintf("/opt/conoha/%s/CURRENT_SLOT", pf.Name)
	_, ptrBytes, _ := ops.Run(fmt.Sprintf("cat '%s' 2>/dev/null || true", ptrPath), nil)
	oldSlot := strings.TrimSpace(string(ptrBytes))
	if oldSlot != "" {
		if err := ValidateSlotID(oldSlot); err != nil {
			fmt.Fprintf(os.Stderr, "warning: CURRENT_SLOT contained %q, ignoring: %v\n", oldSlot, err)
			oldSlot = ""
		}
	}

	if err := runRemoteOps(ops, fmt.Sprintf("printf %%s '%s' > '%s'", slot, ptrPath), nil); err != nil {
		fmt.Fprintf(os.Stderr, "warning: update CURRENT_SLOT pointer (MANUAL: write %q to %s): %v\n", slot, ptrPath, err)
	}

	if oldSlot != "" && oldSlot != slot {
		oldWork := fmt.Sprintf("/opt/conoha/%s/%s", pf.Name, oldSlot)
		schedule := buildScheduleDrainCmd(oldWork, slotProjectName(pf.Name, oldSlot), pf.Name, oldSlot, drainMs)
		if err := runRemoteOps(ops, schedule, nil); err != nil {
			fmt.Fprintf(os.Stderr, "warning: schedule drain teardown: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "==> Scheduled teardown of old slot %q in %dms\n", oldSlot, drainMs)
		}
	}

	active := "<unknown>"
	if updated.ActiveTarget != nil {
		active = updated.ActiveTarget.URL
	}
	fmt.Fprintf(os.Stderr, "Deploy complete. active=%s phase=%s\n", active, updated.Phase)
	return nil
}

// runRemote runs command on cli. When stdinData is non-nil it is streamed as stdin.
// Returns an error if the remote exit status is not zero.
func runRemote(cli *ssh.Client, command string, stdinData *bytes.Buffer) error {
	var code int
	var err error
	if stdinData != nil {
		code, err = internalssh.RunWithStdin(cli, command, stdinData, os.Stdout, os.Stderr)
	} else {
		code, err = internalssh.RunCommand(cli, command, os.Stdout, os.Stderr)
	}
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("remote exit %d", code)
	}
	return nil
}

// tearDownSlot brings down a slot's compose project and removes its work dir.
// Best-effort; the caller already has a more interesting error to return.
func tearDownSlot(cli *ssh.Client, app, slot string) {
	work := fmt.Sprintf("/opt/conoha/%s/%s", app, slot)
	cmd := fmt.Sprintf(
		"docker compose -p %s -f '%s/conoha-override.yml' down 2>/dev/null || true; rm -rf '%s' || true",
		slotProjectName(app, slot), work, work)
	_, _ = internalssh.RunCommand(cli, cmd, os.Stderr, os.Stderr)
}
