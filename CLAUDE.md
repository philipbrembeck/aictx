# aictx — AI Context Switcher

Context switcher for AI tool configurations, inspired by kubectx.

## Build & Run

```bash
go build -o aictx .
go run .
```

## Project Structure

- `cmd/` — Cobra CLI commands (root, add, rm, show, current, discover, targets, version)
- `internal/config/` — Config types and load/save (`~/.config/aictx/config.yaml`)
- `internal/target/` — Target interface, registry, and implementations
  - `claudecli/` — Claude Code CLI target (`~/.claude/settings.json`)
  - `claudevscode/` — Claude Code for VSCode target (surgical edits to VSCode `settings.json`)
  - `picli/` — pi Coding Agent CLI target
- `internal/picker/` — Interactive single-select and multi-select (checkbox) pickers

## Architecture

- **Context**: Named configuration with a shared Provider, Options, and a list of target entries
- **TargetEntry**: `{ID, Env}` only — per-target custom env vars; provider/options live at context level
- **Target interface**: `ID()`, `Name()`, `Detect()`, `Apply(TargetEntry)`, `Discover()`
- Provider and Options live at Context level — one API key and model shared across all targets in a context
- `switchContext` constructs an effective TargetEntry (merges `ctx.Provider` + `ctx.Options` + `te.Env`) before calling `t.Apply()`
- `DiscoveryResult` is returned by `Discover()` and carries the discovered Provider alongside ID and Env

## Conventions

- Always commit changes after completing work
- Always execute tests after making changes
- Always build a new binary after making changes
- Use atomic file writes (temp file + rename) for config files
- VSCode settings: only touch claude-specific keys, preserve everything else (use sjson/gjson)
- Empty provider = OAuth/native auth mode
