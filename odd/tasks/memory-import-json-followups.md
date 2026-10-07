# Memory Import `.json`: Review Follow-ups

## Objective

Close the seven non-blocking suggestions from the last native review of PR #6 (`.json` memory import, lineage `review-d7e19f3651faf4ed`).

## Problem

The `.json` importer works, but some edge cases give unclear errors, ignore flags without telling the user, render multi-line values ambiguously, or are documented inaccurately.

## Why

Each one is small, but together they make imports harder to trust and to debug. Fixing them now, while the code is fresh, is cheap.

## Scope

1. Truncated or empty `.json` reports `unexpected end of JSON input` instead of a bare `EOF`.
2. Test the `maxJSONDepth` guard.
3. `--title-field content` and `--title-field type` are rejected with a clear error.
4. Continuation lines of a multi-line string value are indented two spaces under their label, at every nesting level.
5. The CLI warns on stderr when a flag has no effect: `--include-json` with a file path, and `--title-field` when no `.json` file was read.
6. Docs and test name match the behavior: title-candidate fields that are not used as the title still render.
7. `odd/tasks/memory-import-json.md` acceptance criteria and scope match the final `--include-json` behavior.

Out of scope: any change to which files are read, title resolution order, or `sync_id` identity.

## Constraints

- Test-first for each behavior change.
- No new exported symbols unreachable from `./cmd/gentle-ai` (deadcode ratchet).
- Technical artifacts in English; Conventional Commits without AI attribution.

## Delivery

- Strategy: `ask-on-risk`. Forecast ~200 authored changed lines (under the ~400 budget): one PR.
- Branch: `fix/memory-import-json-followups` from `main` at `9b1dfe7b`.

## Tasks

- [x] T1 — Items 1–5 in `internal/memorycmd` with tests. Route: delegated direct (writer trigger: 2+ non-trivial files).
- [x] T2 — Items 6–7: docs, test name, feature document. Route: delegated direct (same writer, mechanical).

## Acceptance Criteria

- Each item above has an observed test or readback.
- `go build ./... && go vet ./...`, `go test ./internal/memorycmd/... ./internal/tui/... ./internal/app/... -count=1`, `go run ./internal/gofmtcheck`, `./scripts/deadcode-ratchet.sh` pass.

## Progress

- T1 done in two commits:
  - `dee1baa0` `fix(memory): clarify json import errors and reject reserved title fields` (items 1–3). RED: `empty_file` and `missing_value` failed with `x.json: unexpected EOF`; `TestRunRejectsReservedTitleFields` failed with `stat .../missing.json: no such file or directory` (flag not checked before reading). The unclosed-array case and the depth guard already passed (coverage tests). The usage text now notes `--title-field` is not `content` or `type`.
  - `71b8189d` `fix(memory): indent multi-line json values and warn on ignored flags` (items 4–5, plus the test rename). RED: build failed with `undefined: stderr`; after adding only the seam, `TestCollectJSONIndentsMultiLineStrings` showed unindented continuation lines (for example `Note: l1` / `l2`) and every warning case of `TestRunWarnsOnFlagsWithoutEffect` got empty stderr. The engram stderr in `runImport` now goes through the same `stderr` seam. Warnings print after collection succeeds, before the dry-run output or the import.
- T2 done in the `docs(memory): align json import docs with behavior` commit: `docs/usage.md` (`.json` rendering bullet, reserved title fields and warnings bullet) and `odd/tasks/memory-import-json.md` (acceptance criterion and Scope).
- Decision: a multi-line scalar item inside a numbered array keeps aligning under its first line (it is an item, not a field); only labeled fields gain the two-space continuation indent.
- Checks (observed):
  - `go build ./... && go vet ./...`: pass.
  - `go test ./internal/memorycmd/... ./internal/tui/... ./internal/app/... -count=1`: all ok.
  - `go run ./internal/gofmtcheck`: pass.
  - `./scripts/deadcode-ratchet.sh`: "no new unreachable functions" (it also notes 3 pre-existing baselined entries are reachable or gone; baseline untouched).

## Next Step

- Native review of the work-unit commits, then PR under ordinary repository policy.
