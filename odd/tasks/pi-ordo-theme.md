# Pi Ordo theme — ship the Ordo theme to Pi as an asset only

Locator: `odd/tasks/pi-ordo-theme.md` · Engram topic: `odd/pi-ordo-theme/tasks` · Branch: `feat/pi-ordo-theme` · Kanban: project 4, "feat(theme): ship ordo theme to Pi as asset only"

## Objective

The visual theme component (`model.ComponentClaudeTheme`, `theme.InjectVisualThemes`) also writes `<piAgentDir>/themes/ordo.json` in Pi's theme format, without selecting it.

## Problem / Why

- Claude Code and OpenCode get an `ordo` theme asset; Pi gets nothing.
- Pi loads user themes from `<agent-dir>/themes/<name>.json` (agent dir = `~/.pi/agent` or `PI_CODING_AGENT_DIR`), hot-reloaded; format: `$schema`, `name`, `vars`, `colors` (51 color tokens) — see Pi docs `docs/themes.md` (local copy under `~/.pi/agent/install/releases/1.3.0/node_modules/@earendil-works/pi-coding-agent/docs/`) and gentle-pi's `themes/Gentle.json` as a complete example.
- gentle-pi owns the active theme (`/gentle:customize`), so Ordo must not set `settings.theme` for Pi (PR #23 keeps selection to Claude Code/OpenCode).

## Scope

- Split the agent sets in `internal/components/theme/inject.go`: agents that get a theme asset (Claude Code, OpenCode, Pi) vs. agents whose theme Ordo selects (Claude Code, OpenCode). Pi asset dir follows the resolved Pi agent dir.
- Pi theme content: the Ordo (Rosé Pine) palette mapped onto Pi's tokens, valid against Pi's schema shape.
- Uninstall removes the Pi asset (existing `VisualThemePaths` cleanup); install/sync backup paths include it.
- Docs line where per-agent components are listed (if such a page exists).

## Constraints

- No `settings.theme` write for Pi; `VisualThemeSettingsPath` stays "" for Pi.
- Claude Code / OpenCode assets unchanged.
- Read-only use of the local Pi/gentle-pi copies; nothing executed from them.

## Tasks

- [x] T1 — Pi theme asset + agent-set split + tests (asset written at resolved agent dir with valid structure and all color tokens; no settings write; `SelectsTheme(pi)` stays false; uninstall removes it) + docs. Route: delegated writer (2+ non-trivial files).

## Acceptance criteria

- Install/sync with the visual theme component and Pi selected writes `~/.pi/agent/themes/ordo.json` (or under `PI_CODING_AGENT_DIR`), and does not touch `~/.pi/agent/settings.json`.
- Theme JSON has `name: "ordo"`, every `colors` token Pi requires, and all `colors` values resolve to a var or hex.
- Uninstall removes the file.

## Checks

- `go test ./internal/components/theme/... ./internal/components/uninstall/... -count=1`
- `go test ./internal/cli/ -run 'Theme|ComponentPaths|Sync|Uninstall' -count=1 -timeout 30m`
- `go test ./internal/components/ -run Golden -count=1`
- `go build ./... && go vet ./... && go run ./internal/gofmtcheck && ./scripts/deadcode-ratchet.sh`

## Forecast / delivery

~200 authored lines (palette mapping + tests); single PR.

## Progress / evidence

- T1 — delegated writer. `inject.go`: `shipsOrdoTheme` (asset: Claude Code, OpenCode, Pi) and `selectsOrdoTheme` (selection: Claude Code, OpenCode) as two documented sources; Pi `visualThemeDir` = `pi.AgentConfigPath(home)/themes` (no import cycle); `ordoPiTheme` (Rosé Pine vars, all 51 required colors, `appearance: dark`, `export`), `$schema` = `https://pi.dev/schemas/theme.schema.json`. No catalog change needed: install, sync, upgrade backup and uninstall all derive from `VisualThemePaths`. `docs/components.md` row added. +263/−8 (≈70 prod lines).
  - RED: Pi theme test (`VisualThemePaths(pi) = []`), cli Pi component path, uninstall Pi asset; then GREEN.
  - `go build ./... && go vet ./...`: ok; gofmtcheck: ok
  - `go test ./internal/components/theme/... ./internal/components/uninstall/...`: ok (parent spot check re-run: ok)
  - `go test ./internal/components/ -run Golden`: ok
  - `go test ./internal/cli/ -run 'Theme|ComponentPaths|Sync|Uninstall' -timeout 30m`: ok (48s)
  - `./scripts/deadcode-ratchet.sh`: no new unreachable functions
  - Parent check: generated Pi theme validated against Pi 1.3.0 `schemas/theme.schema.json` with Python `jsonschema`: 0 errors
