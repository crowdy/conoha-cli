# Server Create `--for proxy` Preset Design Spec

Closes #182.

## Problem

Creating a proxy host today requires repeating four `--security-group` flags plus UUID-format `--flavor` and `--image` on every `conoha server create --no-input` invocation. Surfaced during the PR #162 row 7 smoke run — every iteration retyped the same boilerplate.

```bash
# Today
conoha server create --no-input --yes --wait \
  --name myproxy \
  --flavor 784f1ae8-... --image 722c231f-... --key-name mykey \
  --security-group default \
  --security-group IPv4v6-SSH \
  --security-group IPv4v6-Web \
  --security-group IPv4v6-ICMP
```

## Solution

Add `--for <preset>` flag. When present, fills in `--flavor`, `--image`, and `--security-group` (repeatable) from a named preset spec. Explicit flags on the same invocation always win.

```bash
# Proposed
conoha server create --no-input --yes --wait \
  --name myproxy --key-name mykey --for proxy
```

Single preset shipped in v0.7.x: `proxy`. Registry shape leaves room for future presets (e.g. `--for k8s-master`) without re-designing the surface.

## Preset definition (`proxy`)

| Field          | Value                                                                        |
|----------------|------------------------------------------------------------------------------|
| Flavor         | `g2l-t-c3m2` (3 vCPU, 2 GB RAM)                                              |
| Image          | latest `vmi-docker-*-ubuntu-*-amd64` resolved at preset-resolution time      |
| Security groups | `default`, `IPv4v6-SSH`, `IPv4v6-Web`, `IPv4v6-ICMP`                        |

**Why `c3m2` over the smaller `c2m1`:** issue #182 references `c1m512` OOMing during docker pull. `c3m2` is the smallest size proven comfortable for the proxy workload (caddy + nginx-proxy + docker pulls). Cost-tuned alternatives can be filed as follow-ups once the preset has real-world soak data.

**Why query latest image:** ConoHa rotates `vmi-docker-*` periodically. A literal pin goes stale within a quarter and forces a CLI release for an image bump. One extra `ListImages` call per preset use is cheap. Match logic: prefix `vmi-docker-`, contains `-ubuntu-`, suffix `-amd64`, status `active`, sort by name descending, take first.

**Why hardcode SG names:** ConoHa3 (v3 API, region `c3j1`) is currently single-region. Code today has zero region branching. Pre-flight `ListSecurityGroups` call validates the four names exist in the user's tenant; missing-name failure prints the actual SG list so the user can self-diagnose.

## Override semantics

| User passes                                  | Result                                                          |
|----------------------------------------------|-----------------------------------------------------------------|
| `--for proxy`                                | All three preset fields applied.                                |
| `--for proxy --flavor g2l-t-c4m4`            | Flavor c4m4, preset image + SGs.                                |
| `--for proxy --security-group custom`        | Custom SG only — preset SG list is **replaced**, not appended.  |
| `--for proxy --image <id>`                   | User image, preset flavor + SGs.                                |
| `--for unknown`                              | Error: lists known presets (`proxy`).                           |

The replace-not-append rule for `--security-group` matches user expectation: if you explicitly chose a SG list, you don't want preset SGs silently mixed in. To extend rather than replace, the user appends preset SG names manually.

## Failure modes

| Condition                                          | Behavior                                                                  |
|----------------------------------------------------|---------------------------------------------------------------------------|
| Any of the four preset SGs missing                 | Fail before server create. Print missing names + actual SG list.          |
| No `vmi-docker-*-ubuntu-*-amd64` image found       | Fail before server create. Suggest `conoha image list` for diagnosis.     |
| Network/Image API error during preset resolution   | Wrap and propagate; same UX as today's interactive selectors.             |
| `--for` and explicit `--flavor`/`--image`/`--security-group` together | Explicit wins; no warning printed.                              |

No silent fallback. The preset's value is being predictable — silent substitution defeats that.

## Implementation

### New file: `cmd/server/preset.go`

```go
package server

type presetSpec struct {
    Flavor         string   // flavor name, e.g. "g2l-t-c3m2"
    SecurityGroups []string // SG names
    ImageMatch     func(name string) bool
}

var presets = map[string]presetSpec{
    "proxy": {
        Flavor: "g2l-t-c3m2",
        SecurityGroups: []string{"default", "IPv4v6-SSH", "IPv4v6-Web", "IPv4v6-ICMP"},
        ImageMatch: matchDockerUbuntuAmd64,
    },
}

// resolvePresetImage queries ListImages and returns the lexicographically
// newest active image whose name matches the preset's ImageMatch.
func resolvePresetImage(api *api.ImageAPI, match func(string) bool) (string, error) { ... }

// validatePresetSecurityGroups returns nil if all names exist, or an error
// listing the missing names alongside the actual SG list.
func validatePresetSecurityGroups(api *api.NetworkAPI, want []string) error { ... }

func matchDockerUbuntuAmd64(name string) bool {
    return strings.HasPrefix(name, "vmi-docker-") &&
        strings.Contains(name, "-ubuntu-") &&
        strings.HasSuffix(name, "-amd64")
}
```

### Wire into `create.go`

After flag parsing, before flavor/image/SG resolution:

```go
forName, _ := cmd.Flags().GetString("for")
if forName != "" {
    spec, ok := presets[forName]
    if !ok {
        return fmt.Errorf("unknown preset %q (known: %s)", forName, knownPresetList())
    }
    if flavorID == "" { flavorID = spec.Flavor }
    if imageID == "" {
        imageID, err = resolvePresetImage(imageAPI, spec.ImageMatch)
        if err != nil { return err }
    }
    if len(sgNames) == 0 {
        if err := validatePresetSecurityGroups(networkAPI, spec.SecurityGroups); err != nil {
            return err
        }
        sgNames = spec.SecurityGroups
    }
}
```

Three integration points (flavorID, imageID, sgNames) match the existing variables already in `create.go`. Preset resolution happens *before* the existing `selectFlavor`/`selectImage`/`selectSecurityGroups` interactive fallbacks, so a user passing `--for proxy` in interactive mode skips the prompts for those three fields.

Flag registration: add to `init()` in `create.go`:
```go
createCmd.Flags().String("for", "", "preset (e.g. \"proxy\") that fills in flavor, image, and security groups")
```

## Testing

`cmd/server/preset_test.go`:
- `TestPresetApply_AllDefaults` — no explicit flags, preset values flow through to server create request.
- `TestPresetApply_ExplicitFlavorWins` — `--for proxy --flavor X` keeps X.
- `TestPresetApply_ExplicitSGReplaces` — `--for proxy --security-group custom` produces only `custom` in the request.
- `TestPresetApply_UnknownName` — error message lists known presets.
- `TestPresetSGValidation_Missing` — when one of the four SGs is missing, error names it and lists actual SGs.
- `TestPresetImageMatch` — table test for `matchDockerUbuntuAmd64`: positive (`vmi-docker-25.10-ubuntu-22.04-amd64`), negatives (`vmi-ubuntu-22.04-amd64`, `vmi-docker-rocky-9-amd64`, `vmi-docker-ubuntu-22.04-arm64`).
- `TestResolvePresetImage_PicksLatest` — multiple matches sorted descending by name; oldest filtered out.
- `TestResolvePresetImage_NoMatch` — empty/non-matching list returns clear error.

Existing `cmd/server/create_test.go` already mocks Compute/Image/Network APIs; reuse those fixtures.

## Out of scope (file as follow-ups if needed)

- Per-region SG name maps (no second region exists yet).
- Multi-preset stacking (`--for proxy --for k8s-worker`).
- Preset definitions loaded from `~/.config/conoha/presets.yaml`.
- Preset-aware image fallback (e.g. `vmi-docker-*-rocky-*` if ubuntu missing).
- Cost-tuned `proxy-min` variant — wait for soak data.
