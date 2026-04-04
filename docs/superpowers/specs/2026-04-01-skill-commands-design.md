# Design: `conoha skill` Commands

**Issue:** #47
**Date:** 2026-04-01

## Summary

Add `conoha skill install|update|remove` commands to manage the `conoha-cli-skill` — a Claude Code skill that teaches Claude how to use ConoHa CLI via natural language.

## Commands

### `conoha skill install`

1. `exec.LookPath("git")` — not found → `ValidationError("git is required")`
2. Check `~/.claude/skills/conoha-cli-skill/` doesn't exist — exists → `ValidationError("already installed, use 'conoha skill update'")`
3. `git clone https://github.com/crowdy/conoha-cli-skill.git ~/.claude/skills/conoha-cli-skill/`
4. Print success to stderr

### `conoha skill update`

1. Check `~/.claude/skills/conoha-cli-skill/` exists — not found → `ValidationError("not installed, use 'conoha skill install'")`
2. Check `.git/` exists inside — not found → `ValidationError("not a git repository, remove and reinstall")`
3. `git -C <dir> pull`
4. Print success to stderr

### `conoha skill remove`

1. Check `~/.claude/skills/conoha-cli-skill/` exists — not found → `ValidationError("not installed")`
2. `prompt.Confirm("Remove conoha-cli-skill?")` — cancelled → print "Cancelled." and return nil
3. `os.RemoveAll(dir)`
4. Print success to stderr

## Structure

Single file: `cmd/skill/skill.go`

- Parent `Cmd` (Use: "skill", Short: "Manage Claude Code skills")
- Three subcommands registered in `init()`
- Constants: `skillRepo`, `skillName`
- `skillDir()` function using `os.UserHomeDir()`

## Registration

Add `skill.Cmd` to `cmd/root.go`.

## Error Handling

| Condition | Error Type | Exit Code |
|-----------|-----------|-----------|
| git not found | ValidationError | 4 |
| Already/not installed | ValidationError | 4 |
| git clone/pull fails | NetworkError | 6 |
| User cancels remove | no error | 0 |

## Design Decisions

- **No tarball fallback** — git is required; users of a CLI tool will have git installed.
- **Single file** — each subcommand is 20-30 lines; splitting would be over-engineering.
- **No API client needed** — these commands don't call ConoHa API, only local git/filesystem.
- **Fixed skill path** — `~/.claude/skills/conoha-cli-skill/` (Claude Code convention).

## Testing

Tests use real tmpdir + `git init` for filesystem operations. Mock the skill directory path via a helper function that accepts a base directory parameter.

## Out of Scope

- SKILL.md content authoring
- Multiple skill support
- Tarball/HTTP fallback
