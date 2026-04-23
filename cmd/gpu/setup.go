package gpu

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	"github.com/crowdy/conoha-cli/cmd/cmdutil"
	"github.com/crowdy/conoha-cli/internal/api"
	cerrors "github.com/crowdy/conoha-cli/internal/errors"
	internalssh "github.com/crowdy/conoha-cli/internal/ssh"
)

func init() {
	setupCmd.Flags().StringP("user", "l", "root", "SSH user")
	setupCmd.Flags().StringP("port", "p", "22", "SSH port")
	setupCmd.Flags().StringP("identity", "i", "", "SSH private key path")
	setupCmd.Flags().Bool("skip-reboot", false, "run the install but don't reboot at the end (default: reboot and wait for nvidia-smi)")
	setupCmd.Flags().Duration("reboot-timeout", 5*time.Minute, "maximum wait for server ACTIVE + SSH after reboot")
}

var setupCmd = &cobra.Command{
	Use:   "setup <server>",
	Short: "Install NVIDIA driver + Container Toolkit on a GPU server",
	Long: `Run the end-to-end post-boot provisioning that turns a fresh GPU VPS
into a Docker host ready to schedule CUDA workloads:

  1. Wait for apt locks (unattended-upgrades often holds them right after first boot).
  2. Install the NVIDIA Container Toolkit (apt repo + nvidia-ctk configure + docker restart).
  3. Install the NVIDIA datacenter driver via ubuntu-drivers install --gpgpu.
  4. Reboot the server (use --skip-reboot to keep running with the current driver).
  5. Wait for the server to come back, then install nvidia-utils and verify with nvidia-smi.

Requires: Ubuntu 22.04+ or 24.04, Docker already installed. See the ConoHa VMI
image vmi-docker-* for a ready-made base.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := cmdutil.NewClient(cmd)
		if err != nil {
			return err
		}
		compute := api.NewComputeAPI(client)

		s, err := compute.FindServer(args[0])
		if err != nil {
			return err
		}
		ip, err := internalssh.ServerIP(s)
		if err != nil {
			return err
		}

		user, _ := cmd.Flags().GetString("user")
		port, _ := cmd.Flags().GetString("port")
		identity, _ := cmd.Flags().GetString("identity")
		skipReboot, _ := cmd.Flags().GetBool("skip-reboot")
		rebootTimeout, _ := cmd.Flags().GetDuration("reboot-timeout")

		if identity == "" {
			identity = internalssh.ResolveKeyPath(s.KeyName)
		}
		if identity == "" {
			return fmt.Errorf("no SSH key found; specify --identity or ensure ~/.ssh/conoha_<keyname> exists")
		}

		cfg := internalssh.ConnectConfig{Host: ip, Port: port, User: user, KeyPath: identity}
		sshClient, err := internalssh.Connect(cfg)
		if err != nil {
			return fmt.Errorf("SSH connect: %w", err)
		}
		fmt.Fprintf(os.Stderr, "==> Connected to %s (%s)\n", s.Name, ip)

		// Phase 1: install toolkit + driver.
		code, err := internalssh.RunScript(sshClient, gpuInstallScript(), nil, os.Stdout, os.Stderr)
		sshClient.Close()
		if err != nil {
			return fmt.Errorf("install script: %w", err)
		}
		if code != 0 {
			return fmt.Errorf("install script exited with code %d", code)
		}

		if skipReboot {
			fmt.Fprintf(os.Stderr, "==> --skip-reboot set; driver load deferred. Run 'conoha server reboot %s --wait' to load the new driver.\n", s.Name)
			return nil
		}

		// Phase 2: reboot + wait for ACTIVE.
		fmt.Fprintf(os.Stderr, "==> Rebooting server %s to load the NVIDIA driver\n", s.Name)
		if err := compute.RebootServer(s.ID, false); err != nil {
			return fmt.Errorf("reboot: %w", err)
		}
		wc := &cmdutil.WaitConfig{Resource: "server " + s.Name, Timeout: rebootTimeout}
		if err := waitForReboot(compute, s.ID, wc); err != nil {
			return err
		}

		// Phase 3: wait for SSH (sshd takes a moment after ACTIVE).
		sshClient, err = waitForSSH(cfg, rebootTimeout)
		if err != nil {
			return err
		}
		defer sshClient.Close()

		// Phase 4: install nvidia-utils + verify.
		code, err = internalssh.RunScript(sshClient, gpuVerifyScript(), nil, os.Stdout, os.Stderr)
		if err != nil {
			return fmt.Errorf("verify script: %w", err)
		}
		if code != 0 {
			return fmt.Errorf("verify script exited with code %d", code)
		}

		fmt.Fprintln(os.Stderr, "==> GPU setup complete.")
		return nil
	},
}

// waitForReboot mirrors cmd/server's two-phase "leave ACTIVE, then return to
// ACTIVE" poll, adapted for the gpu package to avoid cross-package import
// of unexported helpers.
func waitForReboot(compute *api.ComputeAPI, id string, wc *cmdutil.WaitConfig) error {
	// Phase 1: wait for the server to leave ACTIVE.
	_ = cmdutil.WaitFor(cmdutil.WaitConfig{
		Resource: wc.Resource,
		Timeout:  30 * time.Second,
		Interval: 2 * time.Second,
	}, func() (bool, string, error) {
		srv, err := compute.GetServer(id)
		if err != nil {
			return false, "", err
		}
		if srv.Status != "ACTIVE" {
			return true, srv.Status, nil
		}
		return false, srv.Status, nil
	})
	// Phase 2: wait for ACTIVE.
	return cmdutil.WaitFor(cmdutil.WaitConfig{Resource: wc.Resource, Timeout: wc.Timeout}, func() (bool, string, error) {
		srv, err := compute.GetServer(id)
		if err != nil {
			// NotFound during reboot is transient; keep polling briefly.
			var nfe *cerrors.NotFoundError
			if errors.As(err, &nfe) {
				return false, "missing?", nil
			}
			return false, "", err
		}
		if srv.Status == "ACTIVE" {
			return true, "", nil
		}
		return false, srv.Status, nil
	})
}

// waitForSSH polls SSH connect until it succeeds or timeout elapses. The
// server may be ACTIVE before sshd binds — especially on first boot after a
// kernel module change. Uses cmdutil.WaitFor for progress output and Ctrl-C
// handling.
func waitForSSH(cfg internalssh.ConnectConfig, timeout time.Duration) (*ssh.Client, error) {
	var cli *ssh.Client
	err := cmdutil.WaitFor(cmdutil.WaitConfig{
		Resource: "SSH " + cfg.Host,
		Timeout:  timeout,
		Interval: 5 * time.Second,
	}, func() (bool, string, error) {
		c, e := internalssh.Connect(cfg)
		if e == nil {
			cli = c
			return true, "", nil
		}
		return false, "dialing", nil
	})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// gpuInstallScript returns the bash script that installs the NVIDIA Container
// Toolkit and the datacenter driver. Idempotent: safe to re-run; each step
// short-circuits when already applied.
func gpuInstallScript() []byte {
	var b bytes.Buffer
	b.WriteString(`#!/bin/bash
set -euo pipefail

echo "==> Waiting for any apt lock to release..."
for i in $(seq 1 60); do
    if ! fuser /var/lib/dpkg/lock-frontend >/dev/null 2>&1 \
       && ! fuser /var/lib/dpkg/lock         >/dev/null 2>&1 \
       && ! fuser /var/lib/apt/lists/lock    >/dev/null 2>&1; then
        break
    fi
    sleep 5
done

export DEBIAN_FRONTEND=noninteractive

echo "==> Adding NVIDIA Container Toolkit apt repo..."
if [ ! -f /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg ]; then
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey \
        | gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
fi
if [ ! -f /etc/apt/sources.list.d/nvidia-container-toolkit.list ]; then
    curl -fsSL https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list \
        | sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' \
        > /etc/apt/sources.list.d/nvidia-container-toolkit.list
fi

apt-get update -y
apt-get install -y nvidia-container-toolkit

echo "==> Configuring docker runtime..."
if ! grep -q '"nvidia"' /etc/docker/daemon.json 2>/dev/null; then
    nvidia-ctk runtime configure --runtime=docker
    systemctl restart docker
else
    echo "    nvidia runtime already configured; skipping docker restart"
fi

echo "==> Installing NVIDIA datacenter driver via ubuntu-drivers..."
apt-get install -y ubuntu-drivers-common
ubuntu-drivers install --gpgpu

echo "==> Install phase complete (reboot required for driver load)."
`)
	return b.Bytes()
}

// gpuVerifyScript installs the user-space nvidia-smi tool and runs it to
// confirm the GPU is visible. Run after the reboot.
func gpuVerifyScript() []byte {
	var b bytes.Buffer
	b.WriteString(`#!/bin/bash
set -euo pipefail

export DEBIAN_FRONTEND=noninteractive

echo "==> Installing nvidia-utils for nvidia-smi..."
if ! command -v nvidia-smi >/dev/null 2>&1; then
    apt-get install -y nvidia-utils-535-server || apt-get install -y nvidia-utils-550-server || apt-get install -y nvidia-utils
fi

echo "==> nvidia-smi output:"
nvidia-smi

echo "==> Verification complete. To smoke-test docker --gpus manually, run:"
echo "    docker run --rm --gpus all nvidia/cuda:12.4.0-base-ubuntu22.04 nvidia-smi"
`)
	return b.Bytes()
}
