package app

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/crowdy/conoha-cli/cmd/proxy"
	"github.com/crowdy/conoha-cli/internal/config"
	proxypkg "github.com/crowdy/conoha-cli/internal/proxy"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	addAppFlags(deployCmd)
	deployCmd.Flags().String("data-dir", proxy.DefaultDataDir, "proxy data directory on the server")
	deployCmd.Flags().String("slot", "", "override slot ID (default: git short SHA or timestamp)")
}

var deployCmd = &cobra.Command{
	Use:   "deploy <server>",
	Short: "Deploy the current directory via conoha-proxy blue/green",
	Long: `Archive the current directory, upload via SSH, start the web container
in a new compose slot on a dynamic port, then ask conoha-proxy to probe and
swap. The previous slot is torn down after the drain window.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDeploy(cmd, args[0])
	},
}

func runDeploy(cmd *cobra.Command, serverID string) error {
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

	dataDir, _ := cmd.Flags().GetString("data-dir")
	admin := proxypkg.NewClient(&proxypkg.SSHExecutor{Client: sshClient}, proxy.SocketPath(dataDir))

	// Service must exist — init registers it. Missing = user skipped init.
	if _, err := admin.Get(pf.Name); err != nil {
		return fmt.Errorf("service %q not found on proxy — run 'conoha app init %s' first: %w", pf.Name, serverID, err)
	}

	slotOverride, _ := cmd.Flags().GetString("slot")
	slot := slotOverride
	if slot == "" {
		base, err := determineSlotID(".", IsGitRepo("."))
		if err != nil {
			return err
		}
		slot = suffixIfTaken(base, func(candidate string) bool {
			probe := buildComposeProjectExists(slotProjectName(pf.Name, candidate))
			code, _ := internalssh.RunCommand(sshClient, probe, os.Stderr, os.Stderr)
			return code == 0
		})
	}
	if err := ValidateSlotID(slot); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "==> Deploying %q to %s (%s)\n", pf.Name, s.Name, ip)
	fmt.Fprintf(os.Stderr, "==> Slot: %s (compose project: %s)\n", slot, slotProjectName(pf.Name, slot))

	// Upload archive to slot work dir.
	patterns, err := loadIgnorePatterns(".")
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := createTarGz(".", patterns, &buf); err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	slotWork := fmt.Sprintf("/opt/conoha/%s/%s", pf.Name, slot)
	if err := runRemote(sshClient, buildSlotUploadCmd(slotWork, ""), &buf); err != nil {
		return fmt.Errorf("upload: %w", err)
	}

	// Write compose override into the slot dir.
	overrideContent := composeOverride(pf.Name, slot, pf.Web.Service, pf.Web.Port, len(pf.Accessories) > 0)
	overridePath := "conoha-override.yml"
	writeOverride := fmt.Sprintf("cat > '%s/%s' <<'EOF'\n%sEOF", slotWork, overridePath, overrideContent)
	if err := runRemote(sshClient, writeOverride, nil); err != nil {
		return fmt.Errorf("write override: %w", err)
	}

	// First-run: bring up accessories (idempotent via existence probe).
	if len(pf.Accessories) > 0 {
		check := buildAccessoryExists(accessoryProjectName(pf.Name))
		code, _ := internalssh.RunCommand(sshClient, check, os.Stderr, os.Stderr)
		if code != 0 {
			fmt.Fprintf(os.Stderr, "==> Starting accessories: %v\n", pf.Accessories)
			if err := runRemote(sshClient, buildAccessoryUp(slotWork, accessoryProjectName(pf.Name), composeFile, pf.Accessories), nil); err != nil {
				return fmt.Errorf("accessory up: %w", err)
			}
		}
	}

	// Start the new slot's web service.
	fmt.Fprintf(os.Stderr, "==> Building and starting %s in new slot\n", pf.Web.Service)
	if err := runRemote(sshClient, buildSlotComposeUp(slotWork, slotProjectName(pf.Name, slot), composeFile, overridePath, pf.Web.Service), nil); err != nil {
		return fmt.Errorf("compose up (slot): %w", err)
	}

	// Discover kernel-picked host port.
	containerName := fmt.Sprintf("%s-%s-%s", pf.Name, slot, pf.Web.Service)
	var portOut bytes.Buffer
	if _, err := internalssh.RunCommand(sshClient, buildDockerPortCmd(containerName, pf.Web.Port), &portOut, os.Stderr); err != nil {
		tearDownSlot(sshClient, pf.Name, slot)
		return fmt.Errorf("docker port: %w", err)
	}
	hostPort, err := extractHostPort(portOut.String())
	if err != nil {
		tearDownSlot(sshClient, pf.Name, slot)
		return err
	}
	targetURL := fmt.Sprintf("http://127.0.0.1:%d", hostPort)
	fmt.Fprintf(os.Stderr, "==> Host port: %d. Calling proxy /deploy\n", hostPort)

	drainMs := 30000
	if pf.Deploy != nil && pf.Deploy.DrainMs > 0 {
		drainMs = pf.Deploy.DrainMs
	}

	// Call proxy /deploy. On 424 the proxy did not mutate state — tear down new slot.
	updated, err := admin.Deploy(pf.Name, proxypkg.DeployRequest{TargetURL: targetURL, DrainMs: drainMs})
	if err != nil {
		tearDownSlot(sshClient, pf.Name, slot)
		return err
	}

	// Read old slot pointer (empty on first deploy), then update to current.
	ptrPath := fmt.Sprintf("/opt/conoha/%s/CURRENT_SLOT", pf.Name)
	var ptrBuf bytes.Buffer
	_, _ = internalssh.RunCommand(sshClient, fmt.Sprintf("cat '%s' 2>/dev/null || true", ptrPath), &ptrBuf, os.Stderr)
	oldSlot := strings.TrimSpace(ptrBuf.String())
	if oldSlot != "" {
		if err := ValidateSlotID(oldSlot); err != nil {
			fmt.Fprintf(os.Stderr, "warning: CURRENT_SLOT contained %q, ignoring: %v\n", oldSlot, err)
			oldSlot = ""
		}
	}

	if err := runRemote(sshClient, fmt.Sprintf("printf %%s '%s' > '%s'", slot, ptrPath), nil); err != nil {
		fmt.Fprintf(os.Stderr, "warning: update CURRENT_SLOT pointer (MANUAL: write %q to %s): %v\n", slot, ptrPath, err)
	}

	if oldSlot != "" && oldSlot != slot {
		oldWork := fmt.Sprintf("/opt/conoha/%s/%s", pf.Name, oldSlot)
		schedule := buildScheduleDrainCmd(oldWork, slotProjectName(pf.Name, oldSlot), pf.Name, oldSlot, drainMs)
		if err := runRemote(sshClient, schedule, nil); err != nil {
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
