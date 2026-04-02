# Volume Improvements Design (Group A)

Issues: #67, #68, #69

## Approach

Single-file extension of `cmd/volume/volume.go`. No API layer changes needed — `UpdateVolume`, `ListVolumes`, `CreateVolume` already exist with all required capabilities.

## 1. Volume Rename (#67)

New subcommand: `conoha volume rename <id|name>`

```
conoha volume rename <id|name> --name <new-name> [--description <desc>]
```

- At least one of `--name` or `--description` is required (error if neither)
- Name/UUID resolution via `ListVolumes()` — if multiple volumes match by name, error with "multiple volumes found with name X, use UUID instead"
- Calls `UpdateVolume(id, map[string]any{...})` with provided fields
- Outputs updated volume in standard format (table/json/yaml/csv)

## 2. Volume Create Duplicate Name Warning (#68)

Add duplicate check at the start of `volume create` RunE.

**Flow:**
1. Call `ListVolumes()` to get all volumes
2. Check if any volume has the same name as `--name`
3. If duplicate found:
   - Print warning to stderr: `Warning: volume with name "xxx" already exists (ID: yyy)`
   - Call `prompt.Confirm("Create anyway?")`
   - If denied, exit with code 10 (cancel)
4. Non-interactive environment (stdin not a terminal): error and abort on duplicate

No additional flags (`-y`, `--no-input`) — uses existing `prompt.Confirm()` pattern consistent with the rest of the codebase.

## 3. Volume Create --image Flag (#69)

Add optional `--image` flag to `volume create`.

```
conoha volume create --name my-boot-vol --size 30 --image vmi-ubuntu-24.04-amd64 [--wait]
```

**Flow:**
1. If `--image` value looks like UUID, use directly; otherwise resolve name via `ListImages()`
2. If image not found, error: "image not found: xxx"
3. Set resolved image ID to `VolumeCreateRequest.Volume.ImageRef`
4. Rest of create flow unchanged

`--type` is NOT auto-set when `--image` is specified — user must specify explicitly if needed. The `c3j1-ds02-boot` auto-selection in `server create` is server-context specific.

`--size` remains required even with `--image`.

## 4. Testing

All tests use `httptest.Server` for API mocking (project convention).

- **rename**: success case, no flags error, multiple volumes with same name error
- **create duplicate**: duplicate exists → warning message verified via API call checks
- **create --image**: image name resolution, UUID direct use, image not found error
