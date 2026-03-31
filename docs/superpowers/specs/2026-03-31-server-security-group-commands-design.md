# Server add-security-group / remove-security-group Commands

**Issue:** #40
**Date:** 2026-03-31

## Summary

Add two new server subcommands to manage security group assignments on existing servers:
- `conoha server add-security-group <server-id> --name <sg-name>`
- `conoha server remove-security-group <server-id> --name <sg-name>`

## Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `server add-security-group` | `add-sg` | Add a security group to an existing server |
| `server remove-security-group` | `remove-sg` | Remove a security group from an existing server |

### Flags

- `--name` (required): Security group name to add/remove

### Arguments

- First positional argument: server ID or name (resolved via `resolveServerID()`)

### Confirmation

Both commands require `prompt.Confirm()` before execution:
- Add: modifying server security posture warrants user consent
- Remove: destructive operation, already specified in the issue

## API

Both use the existing `ComputeAPI.ServerAction()` pattern (`POST /v2.1/servers/{server_id}/action`):

- **Add:** `{"addSecurityGroup": {"name": "<sg-name>"}}`
- **Remove:** `{"removeSecurityGroup": {"name": "<sg-name>"}}`

## Implementation

### Files to create
- `cmd/server/security_group.go` — command definitions
- `cmd/server/security_group_test.go` — tests

### Files to modify
- `cmd/server/server.go` — register new commands in `init()`
- `internal/api/compute.go` — add `AddSecurityGroup()` / `RemoveSecurityGroup()` wrapper methods

### API methods

```go
func (a *ComputeAPI) AddSecurityGroup(id, name string) error {
    return a.ServerAction(id, map[string]any{
        "addSecurityGroup": map[string]string{"name": name},
    })
}

func (a *ComputeAPI) RemoveSecurityGroup(id, name string) error {
    return a.ServerAction(id, map[string]any{
        "removeSecurityGroup": map[string]string{"name": name},
    })
}
```

### Command flow

1. Get compute API client via `getComputeAPI(cmd)`
2. Resolve server ID via `resolveServerID(compute, args[0])`
3. Get `--name` flag value
4. `prompt.Confirm()` with descriptive message
5. Call API method
6. Print success message to stderr

### Testing

- Use `httptest.Server` to mock API responses (project convention)
- Test cases: successful add/remove, API error handling
- Verify correct HTTP method, URL path, and request body
