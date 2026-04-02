# Server Create Non-TTY Support (#66)

## Problem

`conoha server create` fails with "interactive selection requires a TTY" even when all required flags (`--name`, `--flavor`, `--image`, `--key-name`, `--security-group`, `-y`) are provided. The root cause is `resolveBootVolume()` — when `--volume` is not specified and the flavor requires a boot volume, it always shows an interactive "Create new / Use existing" prompt.

## Solution

Auto-create a boot volume with sensible defaults when running in a non-interactive context (non-TTY or `--no-input`).

### Auto-Create Defaults

| Field       | Value                              |
|-------------|------------------------------------|
| Name        | `{server-name}-boot`               |
| Description | empty                              |
| Size        | `maxBootVolumeGB(flavor)` — 100GB for standard plans, 30GB for 512MB plan |
| Volume type | `c3j1-ds02-boot`                   |
| Image       | already-resolved `imageID`         |

### Changes

**`resolveBootVolume()`** (`cmd/server/create.go`)
- Add `serverName` parameter for deriving the volume name.
- When `--volume` is not specified and not interactive (`!term.IsTerminal() || config.IsNoInput()`), skip the interactive prompt and auto-create with defaults.
- Interactive path (TTY without `--no-input`) remains unchanged.

**`createBootVolume()`** (`cmd/server/create.go`)
- Add parameters for name, description, and size so the caller can supply defaults.
- When called with pre-filled values, skip prompts for those fields.
- When called with empty values (interactive path), prompt as before.

### Non-Interactive Flow

```
flavor needs volume?
  ├─ no  → done (dedicated flavor)
  ├─ --volume given → validate existing volume
  └─ --volume not given
       ├─ interactive (TTY, no --no-input) → existing prompt flow
       └─ non-interactive → auto-create "{name}-boot" (100GB)
```

### No New Flags

The existing `--volume` flag covers the "use existing volume" case. Auto-create covers the common case. No additional flags needed.

## Testing

- Test that `resolveBootVolume` auto-creates when non-interactive
- Test that interactive path still prompts
- Test volume name derivation from server name
