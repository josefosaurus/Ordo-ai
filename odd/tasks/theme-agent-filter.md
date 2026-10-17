# Theme agent filter — select and strip only the managed Ordo theme

Locator: `odd/tasks/theme-agent-filter.md` · Engram topic: `odd/theme-agent-filter/tasks` · Branch: `fix/theme-agent-filter` · Kanban: project 4, "fix(theme): --include-theme writes/strips theme…"

## Objective

`--include-theme` selects `"theme":"ordo"` only for agents that ship an Ordo theme, and uninstall removes the `theme` key only when it names a managed Ordo theme.

## Problem / Why

- `theme.Inject` (`internal/components/theme/inject.go:113`) merges `{"theme":"ordo"}` into `adapter.SettingsPath(home)` for every selected adapter (install `internal/cli/run.go:2799`, sync `internal/cli/sync.go:1582`). Only Claude Code and OpenCode get an `ordo` theme asset (`visualThemeDir`); for Pi, Gemini, Qwen, etc. it selects a theme that does not exist. Pi's active theme is owned by gentle-pi (`/gentle:customize`).
- Uninstall of `ComponentTheme` (`internal/components/uninstall/service.go:922-926`) strips the `theme` key unconditionally (`rewriteJSONFile(path, jsonPath{"theme"})`), erasing a user's or gentle-pi's own theme choice. `removeManagedThemeSetting` (`cleaners.go:88`) already does the safe, value-checked removal and is used for the visual theme component.

## Scope

- Theme selection (install, sync, backup paths `run.go:3629`) only for agents with an Ordo theme asset (Claude Code, OpenCode); others are a no-op.
- Uninstall `ComponentTheme` uses `removeManagedThemeSetting` (strip only managed names) for every agent, which also cleans an `ordo` value earlier releases wrote into other agents.

## Constraints

- Claude Code and OpenCode behavior unchanged (OpenCode keeps `InjectAtPath` with the loaded settings path).
- Golden/existing tests stay green; one concern per change.

## Tasks

- [x] T1 — Theme selection skips agents without an Ordo theme (inject + install/sync backup paths) and uninstall strips only managed theme values (+ tests: Pi selected writes nothing; uninstall keeps a user theme and removes `ordo`). Route: delegated writer (2+ non-trivial files).

## Acceptance criteria

- Install/sync with `--include-theme` and Pi selected leaves `~/.pi/agent/settings.json` without a `theme` key (and declares no Pi theme backup path).
- Claude Code / OpenCode still get `"theme":"ordo"`.
- Uninstall keeps `"theme":"dark"` (or any non-managed value) and removes `"theme":"ordo"`.

## Checks

- `go test ./internal/components/theme/... ./internal/components/uninstall/...`
- `go test ./internal/cli/ -run 'Theme' -count=1 -timeout 30m`
- `go test ./internal/components/ -run Golden -count=1`
- `go build ./... && go vet ./... && go run ./internal/gofmtcheck`

## Forecast / delivery

~120 authored lines; single PR.

## Progress / evidence

- T1 — delegated writer. `theme.SelectsTheme` + private `shipsOrdoTheme` (single source with `visualThemeDir`: Claude Code, OpenCode); `theme.Inject` no-op otherwise; `componentPathsWithWorkspaceScoped` skips the theme path (covers install and sync backups); uninstall `ComponentTheme` uses `removeManagedThemeSetting`. +200/−5 (≈35 prod lines). No existing test changed.
  - RED: `TestInjectSkipsAgentWithoutOrdoThemeAsset` (Pi settings written), `TestComponentThemeUninstallRemovesOnlyManagedThemeSelection` dark cases (theme stripped), `TestThemeSelectionSkipsAgentsWithoutOrdoTheme` (Pi path declared); then GREEN.
  - `go build ./... && go vet ./...`: ok; gofmtcheck: ok
  - `go test ./internal/components/theme/... ./internal/components/uninstall/...`: ok (parent spot check re-run: ok)
  - `go test ./internal/components/ -run Golden`: ok
  - `go test ./internal/cli/ -run 'Theme|ComponentPaths|Sync|Uninstall' -timeout 30m`: ok (46s)
  - `./scripts/deadcode-ratchet.sh`: no new unreachable functions
- Review of 21900d25: reliability lens, approved, acknowledged. Advisory R3-001 applied: uninstall test also covers OpenCode and legacy managed names (`Gentleman Cute`, `gentleman-cute`); 12 cases pass, package ok.
  - Out of scope, noted: catalog still labels `ComponentTheme` "OpenCode Theme" although Claude Code also gets it.
