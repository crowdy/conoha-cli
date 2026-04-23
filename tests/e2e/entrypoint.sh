#!/bin/bash
# Target container entrypoint: wire the authorized_keys mount, generate SSH
# host keys if missing, start dockerd in the background, then run sshd in the
# foreground so `docker stop` cleanly terminates the container.
set -euo pipefail

if [ -f /authorized_keys ]; then
  install -m 0600 -o root -g root /authorized_keys /root/.ssh/authorized_keys
fi

ssh-keygen -A >/dev/null

dockerd --host=unix:///var/run/docker.sock >/var/log/dockerd.log 2>&1 &

for _ in $(seq 1 30); do
  if docker info >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

exec /usr/sbin/sshd -D -e
