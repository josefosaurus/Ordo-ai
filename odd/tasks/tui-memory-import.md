# TUI: Import Memories (`F1` of memory-import)

## Objective

Let users run `ordo memory import` from the interactive TUI: pick a path and project, preview the entries, then import into Engram.

## Problem

`ordo memory import` (PR #2) is CLI-only. TUI users have no way to discover or run it.

## Why

The TUI is Ordo's default entry point (`ordo` with no arguments). Follow-up F1 in `odd/tasks/memory-import.md` (PR #3).

## Scope

- Welcome menu entry "Import memories", placed directly above "Receipt-Driven Development" so the bench `reviewModeMenuUpPresses` (=5, `bench/journeys_issue3766.go`) stays valid.
- Flow: path input → project input (default: current directory's git repo name, else its base name) → preview (dry-run: entry count and titles with sync IDs) → confirm → run the import → result screen showing Engram's summary or the error → back to welcome.
- Reuse `internal/memorycmd` (Collect, BuildExport, engram execution). Expose only what the TUI needs from that package; no duplicated logic.
- Esc goes back one step; errors (empty path, no entries, engram missing) are shown inline and never crash the TUI.
- Out of scope: `--type` and `--force` in the TUI (CLI keeps them).

## Constraints

- Follow the customize screen pattern (commit `992028a4`): `internal/tui/<feature>.go`, `internal/tui/screens/<feature>.go`, screen const and routing in `internal/tui/model.go`, backward route in `internal/tui/router.go`.
- Update welcome option-count tests (`internal/tui/model_test.go`, `internal/tui/screens/welcome_test.go`, `internal/tui/review_mode_test.go`).
- The import runs as a Bubbletea command (async), not inside `Update`.
- Tests never touch the real `~/.engram`: use the memorycmd exec seam or a stub.
- Deadcode ratchet must stay clean. No push/PR without authorization.

## Tasks

- [x] **T1 — Import memories screen.** Menu entry, multi-step screen, memorycmd API for the TUI, tests, `docs/usage.md` TUI section. Route: delegated (writer; 4+ files to understand, 2+ non-trivial files to write). Commit: `feat(tui): add import memories screen` (sha in Progress).

## Acceptance criteria

- The entry appears above RDD; selecting it walks path → project → preview → import → result.
- Preview shows the same sync IDs as `ordo memory import --dry-run`.
- Empty path, no entries and missing engram show an inline error.
- `go build ./... && go vet ./...`, `go test ./internal/tui/... ./internal/memorycmd/... ./internal/app/... -count=1`, `go run ./internal/gofmtcheck`, `./scripts/deadcode-ratchet.sh` pass; bench build in `bench/` unaffected (`cd bench && go build ./... && go vet ./...`).

## Progress

- Branch `feat/tui-memory-import` from `main` (`11b276cc`).
- T1 done (delegated writer).
  - memorycmd API: `Preview`, `Import`, `DefaultProject`; the CLI `runImport` and `Import` share one unexported `importEntries` path. The `runEngram` seam now takes a stderr writer so engram's stderr stays off the TUI (CLI still passes `os.Stderr`).
  - TUI: `internal/tui/memory_import.go`, `internal/tui/screens/memory_import.go`; "Import memories" sits directly above RDD (RDD stays 5th from the end, so bench `reviewModeMenuUpPresses = 5` holds; asserted by `TestImportMemoriesSitsDirectlyAboveReceiptDrivenDevelopment`). The import runs as a `tea.Cmd`; tests inject `Model.memoryImportRun`.
  - Surface expansion (user-approved): `internal/tui/review_store_reset_test.go` (now asserts backups < reset < import memories < uninstall) and `internal/tui/screens/welcome_internal_test.go` (compact viewport height 17 -> 18, with its twin in `internal/tui/model_test.go`), because the compact menu grew to 18 rows.
  - RED: memorycmd build failed (`undefined: Preview`, `undefined: Import`); tui build failed (`undefined: ScreenMemoryImport`, `m.MemoryImport undefined`); welcome count tests failed with 15 options against 16.
  - GREEN/verification:
    - `go build ./... && go vet ./...`: ok
    - `go test ./internal/tui/... ./internal/memorycmd/... ./internal/app/... -count=1`: ok
    - `go run ./internal/gofmtcheck`: clean
    - `./scripts/deadcode-ratchet.sh`: no new unreachable functions (its note "3 baselined entries are now reachable or gone" is pre-existing and not from this change; baseline not updated)
    - `cd bench && go build ./... && go vet ./...`: ok
  - Native review: not run by the writer; left to the parent under RDD.

## Next step

Parent review of T1, then a PR decision (push/PR remain the user's call).
