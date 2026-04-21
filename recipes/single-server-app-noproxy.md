# Single-Server App — No-Proxy Mode

This recipe shows a TLS-less, single-slot deployment of a small web app. Use it when:

- You do not have a public domain.
- The service exposes a non-HTTP protocol.
- You prefer `docker compose up` semantics over blue/green.

For the proxy-backed blue/green variant, see [single-server-app.md](./single-server-app.md).

## 1. Create the VPS

```bash
conoha server create --name myapp --flavor g2l-cpu1-1g --image ubuntu-22.04-x86-64 --ssh-key default
```

## 2. Install Docker and mark the app no-proxy

```bash
conoha app init --no-proxy --app-name myapp myapp
```

This verifies Docker is present on the server and writes the `no-proxy` marker to `/opt/conoha/myapp/.conoha-mode`.

## 3. Prepare a compose file locally

`compose.yml`:

```yaml
services:
  web:
    build: .
    ports:
      - "80:8080"
```

No `conoha.yml` needed.

## 4. Deploy

```bash
conoha app deploy --no-proxy --app-name myapp myapp
```

The CLI tars the current directory (respecting `.dockerignore`), uploads to `/opt/conoha/myapp/` on the VPS, and runs `docker compose -p myapp up -d --build`.

## 5. Day-two operations

```bash
conoha app logs --no-proxy --app-name myapp myapp
conoha app status --no-proxy --app-name myapp myapp
conoha app stop    --no-proxy --app-name myapp myapp
conoha app restart --no-proxy --app-name myapp myapp
conoha app destroy --no-proxy --app-name myapp myapp
```

`conoha app rollback` is not supported in no-proxy mode — deploy a previous revision instead (`git checkout <rev> && conoha app deploy --no-proxy ...`).

## Switching to proxy mode

Run `conoha app destroy ... myapp` followed by `conoha app init ... myapp` (without `--no-proxy`). The CLI refuses implicit mode switches.
