#!/bin/bash
# tests/e2e/run.sh — Phase 1 harness: proxy boot → app init → app deploy.
#
# Scope (spec §8 Phase 1 / scenarios §3 #1-3):
#   1. `conoha proxy boot`  — admin socket up, proxy container running.
#   2. `conoha app init`    — service upserted on the proxy's Admin API.
#   3. `conoha app deploy`  — blue/green first cycle: slot up, active target
#                             set, `GET /` returns 200 via the proxy.
#
# Runs inside a DinD target (Ubuntu + dockerd + sshd) built from
# tests/e2e/Dockerfile.target. A tiny compute-API stub (tests/e2e/stub)
# fakes out the ConoHa control plane; CONOHA_TOKEN bypasses identity.
set -euo pipefail

cd "$(dirname "$0")/../.."

TARGET_NAME="conoha-e2e-target"
STUB_NAME="conoha-e2e-stub"
IMAGE_TAG="conoha-e2e-target:latest"
HOST_SSH_PORT="${HOST_SSH_PORT:-22022}"
HOST_HTTP_PORT="${HOST_HTTP_PORT:-28080}"
STUB_PORT="${STUB_PORT:-28790}"
CONOHA_YML_HOST="${CONOHA_YML_HOST:-e2e.local}"

WORKDIR="$(mktemp -d -t conoha-e2e-XXXXXX)"
trap cleanup EXIT

cleanup() {
  local rc=$?
  set +e
  echo "==> Cleaning up (exit=$rc)"
  if [ -n "${STUB_PID:-}" ]; then
    kill "$STUB_PID" 2>/dev/null || true
    wait "$STUB_PID" 2>/dev/null || true
  fi
  docker rm -f "$TARGET_NAME" >/dev/null 2>&1 || true
  docker volume rm "${TARGET_NAME}-docker-data" >/dev/null 2>&1 || true
  if [ -z "${E2E_KEEP_IMAGE:-}" ]; then
    docker rmi "$IMAGE_TAG" >/dev/null 2>&1 || true
  fi
  rm -rf "$WORKDIR"
  return $rc
}

log() { printf '\n==> %s\n' "$*"; }

log "Workspace: $WORKDIR"

log "Build conoha CLI + stub"
go build -o "$WORKDIR/bin/conoha" ./
go build -o "$WORKDIR/bin/e2e-stub" ./tests/e2e/stub

log "Generate throwaway SSH key"
ssh-keygen -t ed25519 -N '' -f "$WORKDIR/id_ed25519" -C e2e@localhost >/dev/null

log "Build target image ($IMAGE_TAG)"
# --progress=plain keeps build output visible so a CI failure shows the
# offending RUN step rather than a silent `--quiet` dump.
docker build --progress=plain -t "$IMAGE_TAG" -f tests/e2e/Dockerfile.target tests/e2e

log "Start target container"
docker rm -f "$TARGET_NAME" >/dev/null 2>&1 || true
# A named volume for /var/lib/docker sidesteps the overlay-on-overlay issue
# when the outer runner uses the overlay2 storage driver (spec §4.4). The
# inner dockerd mounts a real filesystem and can use overlay2 normally.
DOCKER_DATA_VOL="$TARGET_NAME-docker-data"
docker volume rm "$DOCKER_DATA_VOL" >/dev/null 2>&1 || true
docker volume create "$DOCKER_DATA_VOL" >/dev/null
docker run -d --rm \
  --name "$TARGET_NAME" \
  --privileged --cgroupns=host \
  -p "127.0.0.1:${HOST_SSH_PORT}:22" \
  -p "127.0.0.1:${HOST_HTTP_PORT}:80" \
  -v "$WORKDIR/id_ed25519.pub:/authorized_keys:ro" \
  -v "$DOCKER_DATA_VOL:/var/lib/docker" \
  "$IMAGE_TAG" >/dev/null

log "Wait for sshd on 127.0.0.1:${HOST_SSH_PORT}"
for i in $(seq 1 60); do
  if ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
         -o ConnectTimeout=2 -o BatchMode=yes \
         -p "$HOST_SSH_PORT" -i "$WORKDIR/id_ed25519" root@127.0.0.1 true 2>/dev/null; then
    break
  fi
  if [ "$i" -eq 60 ]; then
    docker logs "$TARGET_NAME" || true
    echo "sshd never came up" >&2
    exit 1
  fi
  sleep 1
done

log "Wait for dockerd inside target"
for i in $(seq 1 60); do
  if docker exec "$TARGET_NAME" docker info >/dev/null 2>&1; then
    break
  fi
  if [ "$i" -eq 60 ]; then
    docker exec "$TARGET_NAME" cat /var/log/dockerd.log 2>/dev/null || true
    echo "dockerd never came up" >&2
    exit 1
  fi
  sleep 1
done

log "Start compute-API stub on 127.0.0.1:${STUB_PORT}"
"$WORKDIR/bin/e2e-stub" \
  --addr "127.0.0.1:${STUB_PORT}" \
  --server-name e2e-target \
  --server-ip 127.0.0.1 \
  >"$WORKDIR/stub.log" 2>&1 &
STUB_PID=$!
for _ in $(seq 1 30); do
  if curl -sf "http://127.0.0.1:${STUB_PORT}/healthz" >/dev/null; then break; fi
  sleep 0.2
done

log "Stage fixture project"
PROJECT="$WORKDIR/project"
mkdir -p "$PROJECT"
cp tests/e2e/fixtures/conoha.yml "$PROJECT/conoha.yml"
cp tests/e2e/fixtures/docker-compose.yml "$PROJECT/docker-compose.yml"

log "Write CLI config + env"
export CONOHA_CONFIG_DIR="$WORKDIR/cfg"
export CONOHA_TOKEN="e2e-stub-token"
export CONOHA_ENDPOINT="http://127.0.0.1:${STUB_PORT}"
export CONOHA_ENDPOINT_MODE="int"
export CONOHA_SSH_INSECURE="1"
export CONOHA_NO_INPUT="1"
export CONOHA_YES="1"
mkdir -p "$CONOHA_CONFIG_DIR"
cat >"$CONOHA_CONFIG_DIR/config.yaml" <<'YAML'
version: 1
active_profile: default
defaults:
  format: json
profiles:
  default:
    tenant_id: e2e-tenant
    username: e2e
    region: e2e
YAML

CONOHA="$WORKDIR/bin/conoha"
SSH_FLAGS=(-l root -p "$HOST_SSH_PORT" -i "$WORKDIR/id_ed25519")

log "Step 1: conoha proxy boot"
"$CONOHA" proxy boot "${SSH_FLAGS[@]}" --acme-email=e2e@example.local e2e-target

log "  verify admin socket responds"
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -p "$HOST_SSH_PORT" -i "$WORKDIR/id_ed25519" root@127.0.0.1 \
    'for i in $(seq 1 30); do [ -S /var/lib/conoha-proxy/admin.sock ] && exit 0; sleep 1; done; exit 1'
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -p "$HOST_SSH_PORT" -i "$WORKDIR/id_ed25519" root@127.0.0.1 \
    'curl -sf --unix-socket /var/lib/conoha-proxy/admin.sock http://admin/healthz || curl -sf --unix-socket /var/lib/conoha-proxy/admin.sock http://admin/readyz'

log "Step 2: conoha app init"
( cd "$PROJECT" && "$CONOHA" app init "${SSH_FLAGS[@]}" e2e-target )

log "  verify service registered on proxy"
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -p "$HOST_SSH_PORT" -i "$WORKDIR/id_ed25519" root@127.0.0.1 \
    'curl -sf --unix-socket /var/lib/conoha-proxy/admin.sock http://admin/v1/services/e2e-app | grep -q e2e-app'

log "Step 3: conoha app deploy (first cycle)"
( cd "$PROJECT" && "$CONOHA" app deploy "${SSH_FLAGS[@]}" e2e-target )

log "  verify active_target is set"
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
    -p "$HOST_SSH_PORT" -i "$WORKDIR/id_ed25519" root@127.0.0.1 \
    'curl -sf --unix-socket /var/lib/conoha-proxy/admin.sock http://admin/v1/services/e2e-app | grep -q active_target'

log "  verify GET / via proxy (Host: ${CONOHA_YML_HOST})"
# Accept 200 or a redirect: the proxy's default HTTP behavior is
# 301→HTTPS, which still proves that HTTP is being routed to the slot.
# TLS is out of scope for this repo (spec §1 bullet 4 and §2.2).
status=$(curl -sS --max-time 10 -o /dev/null -w '%{http_code}' \
     -H "Host: ${CONOHA_YML_HOST}" \
     "http://127.0.0.1:${HOST_HTTP_PORT}/" || echo 000)
case "$status" in
  2??|301|302|308) echo "  got HTTP $status" ;;
  *)
    echo "GET / through proxy returned unexpected status: $status" >&2
    docker exec "$TARGET_NAME" docker ps -a || true
    docker exec "$TARGET_NAME" docker logs conoha-proxy --tail 100 || true
    exit 1
    ;;
esac

log "Phase 1 E2E: all steps passed"
