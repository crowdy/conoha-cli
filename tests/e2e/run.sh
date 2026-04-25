#!/bin/bash
# tests/e2e/run.sh — E2E harness for conoha-proxy blue/green deploy.
#
# Scope covers spec §8 Phases 1-3 + legacy-env scenarios §3 #1-14:
#   1.  `conoha proxy boot`   — admin socket up, proxy container running.
#   2.  `conoha app init`     — service upserted on the proxy's Admin API.
#   3.  `conoha app deploy`   — first cycle: slot up, active target set,
#                               `GET /` routes through the proxy.
#   4.  `conoha app deploy`   — second cycle: active swaps, draining_target
#                               set, old slot container torn down after drain.
#   5.  `conoha app rollback` — within drain window: active swaps back.
#   6.  `conoha app rollback` — outside drain window:
#                               6a. default (no --target): degrade to stderr
#                                   warning + exit 0 (loop-friendly contract).
#                               6b. --target=web: stays fatal with the typed
#                                   "drain window has closed" error.
#   7.  `conoha app list`     — new service enumerated (regression guard for #95).
#   8.  `conoha app init` x2  — idempotent upsert.
#   9.  `app env set` + deploy — new slot receives the set env (regression
#                                guard for #94).
#   13. `app env migrate`     — legacy /opt/conoha/<app>.env.server moves to
#                               /opt/conoha/<app>/.env.server at 0600; idempotent.
#   14. legacy-only env state — `env list` warns + prints value; `env set`
#                               refuses with guard (exit 2 inside the remote
#                               shell script, non-zero CLI exit).
#   10. `conoha app destroy`  — proxy deregisters, /opt/conoha/<app> removed.
#   11. `conoha app destroy` x2 — idempotent (locks in current behavior).
#   12. `conoha proxy remove` — container gone, data dir kept without --purge.
#
# Step numbers match scenario IDs in the spec; #13/#14 run between #9 and #10
# because they need the app to exist but must complete before destroy wipes
# /opt/conoha/<app>/.
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

log "Step 3: conoha app deploy (first cycle, --slot blue)"
# Explicit slot IDs make Phase 2's swap assertions deterministic — we can
# name "blue" / "green" instead of chasing generated timestamps.
( cd "$PROJECT" && "$CONOHA" app deploy "${SSH_FLAGS[@]}" --slot blue e2e-target )

log "  verify active_target is set"
ssh_exec() {
  ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
      -p "$HOST_SSH_PORT" -i "$WORKDIR/id_ed25519" root@127.0.0.1 "$@"
}
svc_json() {
  ssh_exec 'curl -sf --unix-socket /var/lib/conoha-proxy/admin.sock http://admin/v1/services/e2e-app'
}
# Extract a nested JSON field; treats missing/null as empty string so the
# caller can just [ -z ... ] / [ "$x" = "$y" ] without jq's exit-1 behavior.
svc_field() {
  python3 -c "
import json, sys
d = json.load(sys.stdin)
keys = sys.argv[1].split('.')
for k in keys:
    if not isinstance(d, dict):
        d = None
        break
    d = d.get(k)
    if d is None:
        break
print(d if d is not None else '')
" "$1"
}

blue_url="$(svc_json | svc_field active_target.url)"
if [ -z "$blue_url" ]; then
  echo "no active_target after first deploy" >&2
  svc_json >&2
  exit 1
fi
echo "  active (blue) = $blue_url"

log "  verify GET / via proxy (Host: ${CONOHA_YML_HOST})"
# Accept 200 or a redirect: the proxy's default HTTP behavior is
# 301→HTTPS, which still proves that HTTP is being routed to the slot.
# TLS is out of scope for this repo (spec §1 bullet 4 and §2.2).
assert_http_routed() {
  local status
  status=$(curl -sS --max-time 10 -o /dev/null -w '%{http_code}' \
       -H "Host: ${CONOHA_YML_HOST}" \
       "http://127.0.0.1:${HOST_HTTP_PORT}/" || echo 000)
  case "$status" in
    2??|301|302|308) echo "  got HTTP $status" ;;
    *)
      echo "GET / through proxy returned unexpected status: $status" >&2
      docker exec "$TARGET_NAME" docker ps -a || true
      docker exec "$TARGET_NAME" docker logs conoha-proxy --tail 100 || true
      return 1
      ;;
  esac
}
assert_http_routed

log "Phase 1 passed (scenarios #1-3)"

###############################################################################
# Phase 2 (spec §8 / scenarios §3 #4-6)
###############################################################################

log "Step 4: conoha app deploy (second cycle, --slot green → blue/green swap)"
( cd "$PROJECT" && "$CONOHA" app deploy "${SSH_FLAGS[@]}" --slot green e2e-target )

log "  verify active=green, draining=blue"
post_swap="$(svc_json)"
active_after="$(echo "$post_swap" | svc_field active_target.url)"
draining_after="$(echo "$post_swap" | svc_field draining_target.url)"
phase_after="$(echo "$post_swap" | svc_field phase)"
echo "  active   = $active_after"
echo "  draining = $draining_after"
echo "  phase    = $phase_after"
if [ "$active_after" = "$blue_url" ] || [ -z "$active_after" ]; then
  echo "expected active to swap away from blue_url, got: $active_after" >&2
  exit 1
fi
if [ "$draining_after" != "$blue_url" ]; then
  echo "expected draining_target = blue_url ($blue_url), got: $draining_after" >&2
  exit 1
fi
green_url="$active_after"
assert_http_routed

# Scenario #5 must fire inside the drain window set by conoha.yml's
# deploy.drain_ms. Immediate rollback — no sleep before this step.
log "Step 5: conoha app rollback (within drain window)"
( cd "$PROJECT" && "$CONOHA" app rollback "${SSH_FLAGS[@]}" --drain-ms 2000 e2e-target )

log "  verify active swapped back to blue"
post_rb="$(svc_json)"
active_rb="$(echo "$post_rb" | svc_field active_target.url)"
draining_rb="$(echo "$post_rb" | svc_field draining_target.url)"
echo "  active   = $active_rb"
echo "  draining = $draining_rb"
if [ "$active_rb" != "$blue_url" ]; then
  echo "expected rollback to restore active=blue_url ($blue_url), got: $active_rb" >&2
  exit 1
fi
if [ "$draining_rb" != "$green_url" ]; then
  echo "expected draining_target = green_url ($green_url), got: $draining_rb" >&2
  exit 1
fi
assert_http_routed

log "Step 6: wait past drain window, then rollback — dual contract"
# --drain-ms 2000 on the step-5 rollback + slack past the deadline. We
# intentionally do NOT pre-check that draining_target is cleared in the
# admin GET: the proxy's rollback endpoint returns 409 based on the
# deadline, not on whether its internal sweep has fired yet, so asserting
# the JSON field would couple this test to sweep timing without covering
# anything the scenario requires.
#
# Phase 4 (#162) split rollback into two semantic branches and both must
# stay covered:
#   6a. Default (no --target) calls runRollbackAll: 409 on any one block
#       degrades to a stderr warning and the loop continues; the process
#       still exits 0 if no other block produced a hard error. This keeps
#       a single drained-out sub-host from blocking the rest of the app.
#   6b. --target=web pins runRollbackSingle: the user explicitly named
#       this block, so 409 stays fatal with a typed "drain window has
#       closed" error.
# Step 6a does not mutate proxy state (the 409 is a no-op for the service),
# so 6b sees the same expired-deadline state and triggers 409 again.
sleep 4

log "  Step 6a: default rollback (no --target) — expect warning + exit 0"
rb_out="$WORKDIR/rb6a.out"
rb_err="$WORKDIR/rb6a.err"
if ! ( cd "$PROJECT" && "$CONOHA" app rollback "${SSH_FLAGS[@]}" e2e-target ) \
      >"$rb_out" 2>"$rb_err"; then
  echo "default rollback unexpectedly failed after drain window expired:" >&2
  cat "$rb_err" >&2
  exit 1
fi
if ! grep -qi 'drain window expired for e2e-app' "$rb_err"; then
  echo "expected 'drain window expired for e2e-app' warning in stderr; got:" >&2
  cat "$rb_err" >&2
  exit 1
fi
echo "  default rollback warned and exited 0 as expected"

log "  Step 6b: --target=web rollback — expect fatal drain-window error"
rb_err="$WORKDIR/rb6b.err"
if ( cd "$PROJECT" && "$CONOHA" app rollback "${SSH_FLAGS[@]}" --target=web e2e-target ) \
      2>"$rb_err"; then
  echo "--target=web rollback unexpectedly succeeded after drain window expired" >&2
  cat "$rb_err" >&2
  exit 1
fi
if ! grep -qi 'drain window has closed' "$rb_err"; then
  echo "expected 'drain window has closed' in stderr; got:" >&2
  cat "$rb_err" >&2
  exit 1
fi
echo "  --target=web rollback failed as expected"

log "Phase 2 passed (scenarios #4-6)"

###############################################################################
# Phase 3 (spec §8 / scenarios §3 #7-12)
###############################################################################

log "Step 7: conoha app list — expect e2e-app"
list_out="$WORKDIR/list.out"
"$CONOHA" app list "${SSH_FLAGS[@]}" e2e-target >"$list_out"
if ! grep -q '^e2e-app[[:space:]]' "$list_out"; then
  echo "app list did not include e2e-app:" >&2
  cat "$list_out" >&2
  exit 1
fi
echo "  list output:"
sed 's/^/    /' "$list_out"

log "Step 8: conoha app init (second call — idempotent upsert)"
( cd "$PROJECT" && "$CONOHA" app init "${SSH_FLAGS[@]}" e2e-target )
if ! svc_json | grep -q '"name":"e2e-app"'; then
  echo "service e2e-app missing after second init" >&2
  svc_json >&2
  exit 1
fi
echo "  service still registered"

log "Step 9: env set → deploy → env visible inside web container"
"$CONOHA" app env set "${SSH_FLAGS[@]}" --app-name e2e-app e2e-target E2E_SENTINEL=phase3-ok
( cd "$PROJECT" && "$CONOHA" app deploy "${SSH_FLAGS[@]}" --slot env-check e2e-target )
env_value="$(ssh_exec \
  'docker exec e2e-app-env-check-web printenv E2E_SENTINEL 2>/dev/null' || true)"
env_value="$(printf '%s' "$env_value" | tr -d '\r\n')"
if [ "$env_value" != "phase3-ok" ]; then
  echo "E2E_SENTINEL not visible in new slot: got $(printf '%q' "$env_value")" >&2
  ssh_exec 'docker exec e2e-app-env-check-web env' || true
  exit 1
fi
echo "  E2E_SENTINEL=phase3-ok picked up by e2e-app-env-check-web"

###############################################################################
# Phase 3.5 (spec §3 #13-14 — legacy env path migration + deprecation)
###############################################################################
# Runs #14 before #13 because both need a legacy-only filesystem state and
# running #14 first lets us reuse the same staged legacy file for #13's
# migrate assertion.

LEGACY_ENV_PATH='/opt/conoha/e2e-app.env.server'
NEW_ENV_PATH='/opt/conoha/e2e-app/.env.server'

log "Step 13/14 setup: stage legacy-only env state"
ssh_exec "rm -f ${NEW_ENV_PATH}"
ssh_exec "printf 'LEGACY_KEY=legacy-val\n' > ${LEGACY_ENV_PATH} && chmod 600 ${LEGACY_ENV_PATH}"

log "Step 14: conoha app env list — expect legacy warning on stderr + value on stdout"
list_out="$WORKDIR/legacy-list.out"
list_err="$WORKDIR/legacy-list.err"
"$CONOHA" app env list "${SSH_FLAGS[@]}" --app-name e2e-app e2e-target \
    >"$list_out" 2>"$list_err"
if ! grep -q '^LEGACY_KEY=legacy-val$' "$list_out"; then
  echo "env list stdout missing LEGACY_KEY=legacy-val:" >&2
  cat "$list_out" >&2
  cat "$list_err" >&2
  exit 1
fi
if ! grep -q 'legacy env file' "$list_err"; then
  echo "env list stderr missing legacy warning:" >&2
  cat "$list_err" >&2
  exit 1
fi
echo "  env list warned on legacy path and still printed the value"

log "Step 14 (cont.): conoha app env set — expect data-loss guard to refuse"
# The remote guard script exits 2; the CLI surfaces that as the error string
# "env set exited with code 2" and exits non-zero. We assert on the marker
# string rather than the CLI's process exit code so this stays stable if the
# CLI later maps the guard to a typed ExitCoder (currently ExitGeneral=1).
set_err="$WORKDIR/legacy-set.err"
if "$CONOHA" app env set "${SSH_FLAGS[@]}" --app-name e2e-app e2e-target \
       NEWKEY=nope 2>"$set_err"; then
  echo "env set unexpectedly succeeded on legacy-only state" >&2
  cat "$set_err" >&2
  exit 1
fi
# Assert on a guard-unique marker. The shorter "legacy env file" substring
# also appears in the `env list` read-time warning, so matching on it would
# spuriously pass if the guard regressed to a plain warning. The "Writing
# here would silently hide" line is only emitted by the data-loss guard.
if ! grep -q 'Writing here would silently hide' "$set_err"; then
  echo "env set stderr missing guard marker 'Writing here would silently hide':" >&2
  cat "$set_err" >&2
  exit 1
fi
if ! grep -q 'exited with code 2' "$set_err"; then
  echo "env set stderr missing 'exited with code 2' marker from guard:" >&2
  cat "$set_err" >&2
  exit 1
fi
echo "  env set refused legacy-only state (guard exit 2)"

log "  confirm legacy file untouched by the rejected env set"
legacy_content="$(ssh_exec "cat ${LEGACY_ENV_PATH}")"
if [ "$legacy_content" != "LEGACY_KEY=legacy-val" ]; then
  echo "legacy env file was mutated by env set (should have been rejected):" >&2
  echo "  got: $legacy_content" >&2
  exit 1
fi
if ssh_exec "test -f ${NEW_ENV_PATH}"; then
  echo "env set created ${NEW_ENV_PATH} despite guard — data-loss guard broken" >&2
  exit 1
fi

log "Step 13: conoha app env migrate — move legacy → new at 0600"
"$CONOHA" app env migrate "${SSH_FLAGS[@]}" --app-name e2e-app e2e-target

if ssh_exec "test -f ${LEGACY_ENV_PATH}"; then
  echo "legacy env file still present after migrate" >&2
  exit 1
fi
migrated="$(ssh_exec "cat ${NEW_ENV_PATH}")"
if [ "$migrated" != "LEGACY_KEY=legacy-val" ]; then
  echo "migrated file content mismatch (want LEGACY_KEY=legacy-val):" >&2
  echo "  got: $migrated" >&2
  exit 1
fi
mode="$(ssh_exec "stat -c '%a' ${NEW_ENV_PATH}" | tr -d '\r\n')"
if [ "$mode" != "600" ]; then
  echo "migrated file mode expected 600, got: $mode" >&2
  exit 1
fi
echo "  migrate moved legacy → new, content preserved, mode=600"

log "  verify migrate is idempotent (second call is a no-op)"
rerun_out="$WORKDIR/migrate-rerun.out"
"$CONOHA" app env migrate "${SSH_FLAGS[@]}" --app-name e2e-app e2e-target \
    >"$rerun_out"
if ! grep -qi 'nothing to migrate' "$rerun_out"; then
  echo "second migrate did not report 'Nothing to migrate':" >&2
  cat "$rerun_out" >&2
  exit 1
fi
echo "  second migrate was a no-op"

log "Phase 3.5 passed (scenarios #13-14)"

log "Step 10: conoha app destroy"
( cd "$PROJECT" && "$CONOHA" app destroy "${SSH_FLAGS[@]}" --yes --app-name e2e-app e2e-target )

log "  verify service deregistered from proxy"
# /v1/services/<name> returns 404 after delete. curl -f: exit 22 on 4xx.
if ssh_exec \
    'curl -sf --unix-socket /var/lib/conoha-proxy/admin.sock http://admin/v1/services/e2e-app >/dev/null'; then
  echo "service still returns 200 after destroy" >&2
  exit 1
fi
echo "  admin API returned 404 as expected"

log "  verify /opt/conoha/e2e-app removed"
if ssh_exec 'test -d /opt/conoha/e2e-app'; then
  echo "/opt/conoha/e2e-app still exists after destroy" >&2
  ssh_exec 'ls -la /opt/conoha/' >&2
  exit 1
fi
echo "  app dir removed"

log "Step 11: conoha app destroy (second call — idempotent)"
# "既存挙動 lock-in" per spec §3 #12: whatever current behavior is, pin it.
# Today destroy is idempotent (rm -rf + compose down are no-ops when already
# gone), so a second destroy must succeed. If the CLI later chooses to
# error with 'not initialized', update this assertion together.
( cd "$PROJECT" && "$CONOHA" app destroy "${SSH_FLAGS[@]}" --yes --app-name e2e-app e2e-target )
echo "  second destroy succeeded (idempotent, as expected)"

log "Step 12: conoha proxy remove — verify container gone, data dir kept"
"$CONOHA" proxy remove "${SSH_FLAGS[@]}" e2e-target
if ssh_exec 'docker inspect conoha-proxy >/dev/null 2>&1'; then
  echo "conoha-proxy container still exists after remove" >&2
  exit 1
fi
echo "  container removed"
# No --purge passed, so /var/lib/conoha-proxy must survive.
if ! ssh_exec 'test -d /var/lib/conoha-proxy'; then
  echo "/var/lib/conoha-proxy unexpectedly removed without --purge" >&2
  exit 1
fi
echo "  data dir /var/lib/conoha-proxy preserved"

log "Phase 3 passed (scenarios #7-12)"
log "E2E harness: all phases passed"
