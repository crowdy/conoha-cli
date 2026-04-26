package proxy

import "fmt"

// BootParams bundles the config needed by BootScript / RebootScript.
type BootParams struct {
	Email     string // --acme-email value (required)
	Image     string // docker image reference
	DataDir   string // host path mounted at /var/lib/conoha-proxy
	Container string // container name (e.g. "conoha-proxy")
}

// BootScript installs docker if missing, creates the data volume with the
// correct ownership, and runs the conoha-proxy container.
func BootScript(p BootParams) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

if ! command -v docker >/dev/null 2>&1; then
    echo "==> Installing Docker..."
    curl -fsSL https://get.docker.com | sh
fi

echo "==> Preparing data directory %[3]s"
mkdir -p %[3]s
chown 65532:65532 %[3]s

# #164: the proxy container runs as uid 65532 with --network host. Ubuntu's
# default net.ipv4.ip_unprivileged_port_start=1024 plus the bind()-time
# capability check means the proxy can't bind :80/:443 — it crash-loops
# with "permission denied" before admin.sock is created.
#
# --cap-add=NET_BIND_SERVICE alone doesn't help: the cap lands in the
# bounding/permitted set, but a non-root process needs it in the EFFECTIVE
# set, which requires either file capabilities on the binary or ambient
# caps. Neither is in this image.
#
# Per-container '--sysctl net.ipv4.ip_unprivileged_port_start=0' is also
# rejected because we use --network host (host-namespace sysctls aren't
# container-namespaced; runc errors with "not allowed in host network
# namespace").
#
# So set the sysctl at HOST level. Persistent via /etc/sysctl.d so it
# survives reboots; 'sysctl --system' applies immediately. Blast radius:
# any unprivileged user on this VPS can bind ports 80-1023, but this VPS
# is dedicated to the proxy. DinD masks the original bug with
# --privileged.
mkdir -p /etc/sysctl.d
echo 'net.ipv4.ip_unprivileged_port_start=0' > /etc/sysctl.d/99-conoha-proxy.conf
sysctl --system >/dev/null

# #165: stock Ubuntu cloud images run UFW with policy DROP and only SSH
# allowed, so external traffic to :80/:443 — including LE HTTP-01 challenge
# from the ACME servers — is silently dropped after 'proxy boot'. Open the
# two ports here. 'command -v ufw' guards images without UFW (the rule add
# is a no-op then). 'ufw allow' is idempotent.
#
# Placement (load-bearing): this snippet runs BEFORE the docker-inspect
# short-circuit below so a re-run of 'proxy boot' against a VPS where UFW
# state was flushed (manual reset, snapshot revert) still re-asserts the
# rules even when the container already exists. Don't "tidy" it past the
# early-exit.
#
# Errors are intentionally swallowed via '|| true': #165 is a best-effort
# firewall convenience, not a hard prerequisite. A future "ports closed"
# debug should run 'ufw status' directly rather than relying on this log.
if command -v ufw >/dev/null 2>&1; then
    ufw allow 80/tcp >/dev/null || true
    ufw allow 443/tcp >/dev/null || true
fi

if docker inspect %[4]s >/dev/null 2>&1; then
    echo "Container %[4]s already exists. Use 'conoha proxy reboot' to upgrade."
    exit 0
fi

echo "==> Starting %[4]s from %[2]s"
# --network host: CLI's app deploy probes slots at http://127.0.0.1:<slot-port>,
# which only resolves to the slot when the proxy shares the host loopback.
# Bridge-networked containers would see their own loopback and the probe
# would fail (spec 2026-04-20 §5 step 10). The :80/:443 bind permission
# question is solved by the host sysctl above (#164), not by --cap-add.
docker run -d --name %[4]s \
  --restart unless-stopped \
  --network host \
  -v %[3]s:%[3]s \
  %[2]s \
  run --acme-email=%[1]s

echo "==> Done. Admin socket: %[3]s/admin.sock"
`, p.Email, p.Image, p.DataDir, p.Container))
}

// RebootScript pulls the image then replaces the existing container, keeping the volume.
func RebootScript(p BootParams) []byte {
	return []byte(fmt.Sprintf(`#!/bin/bash
set -euo pipefail

echo "==> Pulling %[2]s"
docker pull %[2]s

# Re-assert host sysctl + UFW rules on the reboot path so an in-place
# upgrade against a VPS where /etc/sysctl.d or UFW state was reset
# (image rebuild, manual cleanup) re-establishes them. See #164/#165.
mkdir -p /etc/sysctl.d
echo 'net.ipv4.ip_unprivileged_port_start=0' > /etc/sysctl.d/99-conoha-proxy.conf
sysctl --system >/dev/null
if command -v ufw >/dev/null 2>&1; then
    ufw allow 80/tcp >/dev/null || true
    ufw allow 443/tcp >/dev/null || true
fi

if docker inspect %[4]s >/dev/null 2>&1; then
    echo "==> Stopping %[4]s"
    docker stop %[4]s >/dev/null
    docker rm %[4]s >/dev/null
fi

echo "==> Starting new %[4]s from %[2]s"
# See BootScript for why --network host is required and why the bind
# permission is solved by the host sysctl above (#164), not by --cap-add.
docker run -d --name %[4]s \
  --restart unless-stopped \
  --network host \
  -v %[3]s:%[3]s \
  %[2]s \
  run --acme-email=%[1]s
`, p.Email, p.Image, p.DataDir, p.Container))
}

// StartScript / StopScript / RestartScript are trivial wrappers.
func StartScript(container string) []byte {
	return []byte(fmt.Sprintf("#!/bin/bash\nset -e\ndocker start %s\n", container))
}

func StopScript(container string) []byte {
	return []byte(fmt.Sprintf("#!/bin/bash\nset -e\ndocker stop %s\n", container))
}

func RestartScript(container string) []byte {
	return []byte(fmt.Sprintf("#!/bin/bash\nset -e\ndocker restart %s\n", container))
}

// RemoveScript removes the container. When purge=true, the host data dir is also deleted.
func RemoveScript(container, dataDir string, purge bool) []byte {
	script := fmt.Sprintf("#!/bin/bash\nset -e\ndocker rm -f %s 2>/dev/null || true\n", container)
	if purge {
		script += fmt.Sprintf("rm -rf %s\n", dataDir)
	}
	return []byte(script)
}

// LogsScript returns `docker logs` with optional follow/tail flags.
func LogsScript(container string, follow bool, lines int) []byte {
	cmd := "docker logs"
	if follow {
		cmd += " -f"
	}
	if lines > 0 {
		cmd += fmt.Sprintf(" --tail %d", lines)
	}
	cmd += " " + container
	return []byte("#!/bin/bash\nset -e\n" + cmd + "\n")
}
