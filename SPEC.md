# ConoHa CLI Specification

**Version**: 0.1.0
**Status**: Initial Release

## Scope

ConoHa VPS3 API の全エンドポイントに対応する CLI ツール。

## Supported APIs

| Service | Endpoint Prefix | Commands |
|---------|----------------|----------|
| Identity | `identity.{region}.conoha.io/v3` | auth, identity |
| Compute | `compute.{region}.conoha.io/v2.1` | server, flavor, keypair |
| Block Storage | `block-storage.{region}.conoha.io/v3` | volume |
| Image | `image.{region}.conoha.io/v2` | image |
| Network | `networking.{region}.conoha.io/v2.0` | network, lb |
| DNS | `dns-service.{region}.conoha.io/v1` | dns |
| Object Storage | `object-storage.{region}.conoha.io/v1` | storage |

## Auth Flow

1. `POST /v3/auth/tokens` → X-Subject-Token (24h TTL)
2. Token cached in `~/.config/conoha/tokens.yaml`
3. Auto-refresh 5 min before expiry

## Output Contract

- `--format json`: stdout = JSON only, stderr = progress/warnings
- Exit codes: 0=ok, 1=general, 2=auth, 3=not-found, 4=validation, 5=api, 6=network, 10=cancelled

## Version History

| Version | Date | Description |
|---------|------|-------------|
| 0.1.0 | 2026-03-10 | Initial implementation - all API endpoints |
