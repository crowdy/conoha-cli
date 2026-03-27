# v0.1.10 Design: Load Balancer CLI Completion + Image Upload

## Overview

Complete the Load Balancer CLI with full CRUD for all sub-resources (listener, pool, member, healthmonitor), and add image create/upload/import commands. The LB API layer already exists; this release focuses on CLI commands, model enrichment, and the new image upload pipeline.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| LB file splitting | Resource-based (5 files) | Matches server.go split pattern from v0.1.7 |
| Image upload UX | Both convenience + individual commands | `image import` for simple use, `image create` + `image upload` for flexibility |
| Model fields | Full OpenAPI spec reflection | `show` outputs entire struct; all API-returned fields should be present |
| `--wait` for LB | Yes, all sub-resource creates | `cmdutil.WaitFor()` already exists; LB resources have `provisioning_status` |
| Implementation order | LB first, then image | LB has existing API; image requires new binary upload capability |
| Update commands | Deferred | OpenAPI supports PUT for all sub-resources; out of scope for v0.1.10 |

## 1. Model Changes

### Shared Helper Type

```go
// IDRef is used for association arrays (loadbalancers, listeners, pools)
type IDRef struct {
    ID string `json:"id" yaml:"id"`
}
```

### `internal/model/loadbalancer.go`

Expand all sub-resource structs to match OpenAPI spec response fields.

**Listener** — OpenAPI response returns `loadbalancers` as an array, not a flat ID:
```go
type Listener struct {
    ID                 string  `json:"id" yaml:"id"`
    Name               string  `json:"name" yaml:"name"`
    Description        string  `json:"description" yaml:"description"`
    ProvisioningStatus string  `json:"provisioning_status" yaml:"provisioning_status"`
    OperatingStatus    string  `json:"operating_status" yaml:"operating_status"`
    AdminStateUp       bool    `json:"admin_state_up" yaml:"admin_state_up"`
    Protocol           string  `json:"protocol" yaml:"protocol"`
    ProtocolPort       int     `json:"protocol_port" yaml:"protocol_port"`
    ConnectionLimit    int     `json:"connection_limit" yaml:"connection_limit"`
    DefaultPoolID      string  `json:"default_pool_id" yaml:"default_pool_id"`
    Loadbalancers      []IDRef `json:"loadbalancers" yaml:"loadbalancers"`
    ProjectID          string  `json:"project_id" yaml:"project_id"`
    TenantID           string  `json:"tenant_id" yaml:"tenant_id"`
}
```
Note: `CreateListener` API request sends `loadbalancer_id` (flat string), but the response returns `loadbalancers` (array). The list row struct extracts the first element's ID for display.

**Pool** — OpenAPI response includes `loadbalancers`, `listeners`, `members` arrays:
```go
type Pool struct {
    ID                 string  `json:"id" yaml:"id"`
    Name               string  `json:"name" yaml:"name"`
    Description        string  `json:"description" yaml:"description"`
    ProvisioningStatus string  `json:"provisioning_status" yaml:"provisioning_status"`
    OperatingStatus    string  `json:"operating_status" yaml:"operating_status"`
    AdminStateUp       bool    `json:"admin_state_up" yaml:"admin_state_up"`
    Protocol           string  `json:"protocol" yaml:"protocol"`
    LBMethod           string  `json:"lb_algorithm" yaml:"lb_algorithm"`
    Loadbalancers      []IDRef `json:"loadbalancers" yaml:"loadbalancers"`
    Listeners          []IDRef `json:"listeners" yaml:"listeners"`
    Members            []string `json:"members" yaml:"members"`
    ProjectID          string  `json:"project_id" yaml:"project_id"`
    TenantID           string  `json:"tenant_id" yaml:"tenant_id"`
}
```

**Member** (add 4 fields):
```go
type Member struct {
    ID                 string `json:"id" yaml:"id"`
    Name               string `json:"name" yaml:"name"`
    Address            string `json:"address" yaml:"address"`
    ProtocolPort       int    `json:"protocol_port" yaml:"protocol_port"`
    Weight             int    `json:"weight" yaml:"weight"`
    OperatingStatus    string `json:"operating_status" yaml:"operating_status"`
    ProvisioningStatus string `json:"provisioning_status" yaml:"provisioning_status"`
    AdminStateUp       bool   `json:"admin_state_up" yaml:"admin_state_up"`
    ProjectID          string `json:"project_id" yaml:"project_id"`
    TenantID           string `json:"tenant_id" yaml:"tenant_id"`
}
```

**HealthMonitor** — OpenAPI response returns `pools` as an array, not `pool_id`:
```go
type HealthMonitor struct {
    ID                 string  `json:"id" yaml:"id"`
    Name               string  `json:"name" yaml:"name"`
    Type               string  `json:"type" yaml:"type"`
    Delay              int     `json:"delay" yaml:"delay"`
    Timeout            int     `json:"timeout" yaml:"timeout"`
    MaxRetries         int     `json:"max_retries" yaml:"max_retries"`
    URLPath            string  `json:"url_path" yaml:"url_path"`
    ExpectedCodes      string  `json:"expected_codes" yaml:"expected_codes"`
    AdminStateUp       bool    `json:"admin_state_up" yaml:"admin_state_up"`
    Pools              []IDRef `json:"pools" yaml:"pools"`
    ProvisioningStatus string  `json:"provisioning_status" yaml:"provisioning_status"`
    OperatingStatus    string  `json:"operating_status" yaml:"operating_status"`
    ProjectID          string  `json:"project_id" yaml:"project_id"`
    TenantID           string  `json:"tenant_id" yaml:"tenant_id"`
}
```
Note: `CreateHealthMonitor` API request sends `pool_id` (flat string), but the response returns `pools` (array). The list row struct extracts the first element's ID for display.

### `internal/model/image.go`

Add request struct for image creation:

```go
type ImageCreateRequest struct {
    Name            string `json:"name"`
    DiskFormat      string `json:"disk_format"`
    ContainerFormat string `json:"container_format"`
}
```

Enrich existing `Image` struct with additional fields from OpenAPI response:

```go
type Image struct {
    ID              string    `json:"id" yaml:"id"`
    Name            string    `json:"name" yaml:"name"`
    Status          string    `json:"status" yaml:"status"`
    DiskFormat      string    `json:"disk_format" yaml:"disk_format"`
    ContainerFormat string    `json:"container_format" yaml:"container_format"`
    MinDisk         int       `json:"min_disk" yaml:"min_disk"`
    MinRAM          int       `json:"min_ram" yaml:"min_ram"`
    Size            int64     `json:"size" yaml:"size"`
    Checksum        string    `json:"checksum" yaml:"checksum"`
    Visibility      string    `json:"visibility" yaml:"visibility"`
    Owner           string    `json:"owner" yaml:"owner"`
    CreatedAt       time.Time `json:"created_at" yaml:"created_at"`
}
```

## 2. API Changes

### `internal/api/loadbalancer.go`

**Add 4 Get methods:**

```go
func (a *LoadBalancerAPI) GetListener(id string) (*model.Listener, error)
func (a *LoadBalancerAPI) GetPool(id string) (*model.Pool, error)
func (a *LoadBalancerAPI) GetMember(poolID, memberID string) (*model.Member, error)
func (a *LoadBalancerAPI) GetHealthMonitor(id string) (*model.HealthMonitor, error)
```

Pattern: same as `GetLoadBalancer` — `var resp struct{ X model.X }` wrapper.

**Modify 2 Create signatures:**

```go
// Add name parameter; weight is optional (omit from body when 0, defaults to 1 server-side)
func (a *LoadBalancerAPI) CreateMember(poolID, name, address string, port int, weight *int) (*model.Member, error)

// Add name + optional HTTP health check fields
func (a *LoadBalancerAPI) CreateHealthMonitor(poolID, name, monitorType string, delay, timeout, maxRetries int, urlPath, expectedCodes string) (*model.HealthMonitor, error)
```

- `CreateMember`: `weight` is a pointer — `nil` means omit from request body (API defaults to 1). OpenAPI does not list `weight` in required fields for create.
- `CreateHealthMonitor`: `urlPath` and `expectedCodes` are only included in the request body when non-empty (for HTTP/HTTPS type monitors).

### `internal/api/image.go`

**Add 2 methods:**

```go
// POST /v2/images (returns 200, not 201)
func (a *ImageAPI) CreateImage(name, diskFormat, containerFormat string) (*model.Image, error)

// PUT /v2/images/{id}/file
func (a *ImageAPI) UploadImageFile(id string, reader io.Reader, size int64) error
```

`UploadImageFile` uses `Client.Do` directly with `Content-Type: application/octet-stream`. Streams from `io.Reader` without loading entire file into memory. Expects 204 No Content response.

## 3. CLI Commands — LB File Split

### `cmd/lb/` directory structure

| File | Content |
|------|---------|
| `lb.go` | `Cmd`, top-level `init()`, LB list/show/create/delete, shared `waitForLBResource()` helper |
| `listener.go` | `listenerCmd` + list/show/create/delete |
| `pool.go` | `poolCmd` + list/show/create/delete |
| `member.go` | `memberCmd` + list/show/create/delete |
| `healthmonitor.go` | `healthMonitorCmd` + list/show/create/delete |

`lb.go` `init()` registers sub-resource commands: `Cmd.AddCommand(listenerCmd, poolCmd, memberCmd, healthMonitorCmd)`. Each file has its own `init()` to register sub-subcommands (existing pattern).

## 4. CLI Commands — LB Sub-Resources (12 new commands)

### Listener

```
conoha lb listener list
conoha lb listener show <id>
conoha lb listener create --name X --protocol TCP --port 80 --lb-id UUID [--wait]
conoha lb listener delete <id>
```

Create flags: `--name` (required), `--protocol` (required), `--port` (required, int), `--lb-id` (required).

### Pool

```
conoha lb pool list
conoha lb pool show <id>
conoha lb pool create --name X --protocol TCP --lb-algorithm ROUND_ROBIN --listener-id UUID [--wait]
conoha lb pool delete <id>
```

Create flags: `--name` (required), `--protocol` (required), `--lb-algorithm` (required), `--listener-id` (required).

### Member

```
conoha lb member list --pool-id UUID
conoha lb member show <member-id> --pool-id UUID
conoha lb member create --name X --address 1.2.3.4 --port 8080 --pool-id UUID [--weight 1] [--wait]
conoha lb member delete <member-id> --pool-id UUID
```

All member commands require `--pool-id` (nested resource under pool). `--weight` defaults to 1 if omitted.

### HealthMonitor

```
conoha lb healthmonitor list
conoha lb healthmonitor show <id>
conoha lb healthmonitor create --name X --pool-id UUID --type TCP --delay 5 --timeout 3 --max-retries 3 [--url-path /health] [--expected-codes 200] [--wait]
conoha lb healthmonitor delete <id>
```

Create flags: `--name` (required), `--pool-id` (required), `--type` (required), `--delay` (required, int), `--timeout` (required, int), `--max-retries` (required, int). `--url-path` and `--expected-codes` are optional (for HTTP/HTTPS type). Validation: `timeout` must be less than `delay`.

### Shared `--wait` Helper

```go
// In lb.go
func waitForLBResource(lbAPI *api.LoadBalancerAPI, resourceType, id, poolID string, wc *cmdutil.WaitConfig) error {
    return cmdutil.WaitFor(wc, func() (bool, string, error) {
        // Switch on resourceType to call appropriate Get method
        // Return done=true when provisioning_status == "ACTIVE"
        // Return error when provisioning_status == "ERROR"
    })
}
```

### Command Patterns

- **show**: `ExactArgs(1)`, Get API call, `FormatOutput(cmd, item)`
- **create**: required flags via `MarkFlagRequired`, API call, `FormatOutput(cmd, created)`, optional `--wait` polling
- **delete**: `ExactArgs(1)`, fetch resource via Get to display name, `prompt.Confirm("Delete {resource} \"{name}\" ({id})?")`, Delete API call, stderr success message. Also update existing `lb delete` command to use this pattern for consistency.

### List Command Row Structs

Existing list commands output full model structs. For consistency with other list commands (server, volume), add `row` structs to show only key fields in table output:

- **Listener list row**: ID, Name, Protocol, ProtocolPort, OperatingStatus (LB ID extracted from `Loadbalancers[0].ID` if present)
- **Pool list row**: ID, Name, Protocol, LBMethod, OperatingStatus
- **Member list row**: ID, Name, Address, ProtocolPort, Weight, OperatingStatus
- **HealthMonitor list row**: ID, Name, Type, Delay, Timeout, PoolID (extracted from `Pools[0].ID` if present)

## 5. CLI Commands — Image (3 new commands)

### `image create`

```
conoha image create --name my-iso [--disk-format iso] [--container-format bare]
```

- `--name` required
- `--disk-format` default `iso` (ConoHa only supports iso)
- `--container-format` default `bare` (optional per OpenAPI; omit from request body if empty)
- Output: created image record (id, name, status=queued)

### `image upload`

```
conoha image upload <id> --file ubuntu.iso
```

- `ExactArgs(1)` for image ID
- `--file` required
- Validates file exists and is readable
- Prints to stderr: `Uploading {filename} ({human-readable size})...`
- Calls `UploadImageFile(id, reader, size)`
- On success, calls `GetImage(id)` and outputs result

### `image import`

```
conoha image import --name my-iso --file ubuntu.iso [--wait]
```

- Convenience command: internally calls create then upload
- `--name` required, `--file` required
- On upload failure, prints: `Image record created (ID: xxx) but upload failed. Retry with: conoha image upload xxx --file ...`
- `--wait`: polls `GetImage(id)` until status becomes `active`. Status flow: `queued` → `saving` → `active`. Treat `killed` or `deactivated` as terminal errors.

### Binary Upload Implementation

In `internal/api/image.go`:

```go
func (a *ImageAPI) UploadImageFile(id string, reader io.Reader, size int64) error {
    url := fmt.Sprintf("%s/images/%s/file", a.baseURL(), id)
    req, err := http.NewRequest("PUT", url, reader)
    if err != nil {
        return err
    }
    req.Header.Set("Content-Type", "application/octet-stream")
    req.ContentLength = size
    resp, err := a.Client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    if resp.StatusCode != http.StatusNoContent {
        // Parse error response
    }
    return nil
}
```

Streams from `io.Reader` (via `os.Open`) — no memory buffering for large ISO files.

**Implementation notes for binary upload:**
- **Timeout**: The default `Client.HTTP` has a 30s timeout, which will fail for large ISO uploads. `UploadImageFile` must create a dedicated `http.Client` with no timeout (or use `context.WithTimeout` with a generous limit) for the upload request.
- **Debug logging bypass**: `Client.Do` reads the entire request body into memory when debug logging is enabled (`io.ReadAll(req.Body)`). The upload must use `Client.HTTP.Do(req)` directly (after setting auth headers manually) to avoid this, or check body size before debug-logging.
- **Response format**: Image API responses are flat JSON (not wrapped in `{"image": {...}}`), unlike LB resources. `CreateImage` should unmarshal directly into `model.Image`.

## 6. Client-Side Validation

Validate flag values before API calls to provide clear error messages:

| Flag | Valid Values | Commands |
|------|-------------|----------|
| `--protocol` (listener) | `TCP`, `UDP` | `lb listener create` |
| `--protocol` (pool) | `TCP`, `UDP` | `lb pool create` |
| `--lb-algorithm` | `ROUND_ROBIN`, `LEAST_CONNECTIONS` | `lb pool create` |
| `--type` (healthmonitor) | `TCP`, `HTTP`, `HTTPS`, `PING`, `UDP-CONNECT` | `lb healthmonitor create` |
| `--timeout` < `--delay` | numeric comparison | `lb healthmonitor create` |

## 7. Verification Plan

1. `go build ./...` — compiles
2. `golangci-lint run ./...` — no lint errors
3. `go test ./...` — existing tests pass
4. Manual testing:
   - `conoha lb listener list` — existing command still works
   - `conoha lb listener create/show/delete` — new commands
   - `conoha lb listener create --wait` — polling works
   - `conoha image create/upload/import` — new commands
5. `conoha lb listener --help` — shows all subcommands

## 8. Commit Strategy

| # | Scope | Description |
|---|-------|-------------|
| 1 | Model | Expand LB model structs with full OpenAPI fields, add ImageCreateRequest |
| 2 | API | Add LB Get methods, fix CreateMember/CreateHealthMonitor signatures, add Image create/upload |
| 3 | CLI (LB) | Split lb.go into 5 files (no behavior change) |
| 4 | CLI (LB) | Add show/create/delete commands for all sub-resources + --wait |
| 5 | CLI (Image) | Add image create, upload, import commands |

Single PR: `v0.1.10: LB CLI completion + image upload`
