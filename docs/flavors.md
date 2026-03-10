# ConoHa VPS3 Flavor Naming Convention

## Format

```
g2X-T-cNmM
```

| Part | Meaning | Example |
|------|---------|---------|
| `g2` | Generation 2 | - |
| `X` | Type character | `l`, `w`, `d` |
| `T` | Tier | `t` (trial), etc. |
| `cN` | vCPU count | `c2` = 2 vCPU |
| `mM` | Memory (GB) | `m1` = 1GB RAM |

## Type Character (3rd character)

| Char | Type | Boot Volume Required |
|------|------|---------------------|
| `l` | Linux | Yes |
| `w` | Windows | Yes |
| `d` | Dedicated | No (local disk) |

## Boot Volume Behavior

- **Linux / Windows flavors** (`g2l-*`, `g2w-*`): The API requires `block_device_mapping_v2`. You must provide a boot volume, either an existing one (`--volume <id>`) or create a new one interactively.
- **Dedicated flavors** (`g2d-*`): Boot directly from an image. No volume needed.

## Volume Types

When creating a boot volume, choose a volume type:

| Name | API Value | Use Case |
|------|-----------|----------|
| boot-vps-default | c3j1-ds02-boot | Standard VPS boot volume (default) |
| boot-vps-gpu | c3j1-ds03-boot | VPS GPU plan boot volume |
| boot-game-default | c3j1-ds01-boot | Game server boot volume |
| boot-game-gpu | c3j1-ds03-boot | Game server GPU boot volume |

Additional (non-boot) volume types:

| Name | API Value |
|------|-----------|
| add-vps-default | c3j1-ds02-add |
| add-game-default | c3j1-ds01-add |

## Volume Sizes

- **100GB** — boot volume (default)
- **200GB** — additional volume
- **500GB** — additional volume

## Examples

```
g2l-t-c2m1    # Linux, trial, 2 vCPU, 1GB RAM  → volume required
g2l-p-c2m1    # Linux, production, 2 vCPU, 1GB RAM → volume required
g2w-t-c2m4    # Windows, trial, 2 vCPU, 4GB RAM → volume required
g2d-t-c2m1    # Dedicated, trial, 2 vCPU, 1GB RAM → no volume needed
```
