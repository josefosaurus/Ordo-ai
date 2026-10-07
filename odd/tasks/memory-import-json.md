# Memory Import: `.json` Datasets

## Objective

Let `ordo memory import` (CLI and TUI) load plain `.json` datasets directly, without a manual `jq` conversion to `.jsonl`.

## Problem

Real datasets (for example a golden dataset's `documents.json`) are a JSON array of rich records (`id`, `titulo`, `modulos`, `pasos`, ...) with no `title`/`content` fields. The importer only reads `.md`, `.csv`, and `.jsonl` with `title`/`content`, so `.json` files are silently skipped in directory scans and rejected as a single file.

## Why

Converting by hand is error-prone and blocks the TUI path, which has no place for a conversion step. Rendering records as labeled text keeps memories readable for agents and spends less of the 8000-character budget than raw JSON.

## Scope

- `.json` input: a top-level array of objects, or an object holding exactly one array of objects.
- Title: `--title-field <name>` when given; otherwise the first non-empty of `title`, `titulo`, `name`, combined with `id` as `<id> — <title>` when `id` is present and different; `id` alone when no title field; otherwise a clear error naming `--title-field`.
- Content: a non-empty string `content` field is used as is; otherwise the remaining fields (except the title fields and `type`) render as labeled text in source key order (scalars `Label: v`, scalar arrays joined by `, `, objects and object arrays indented and numbered). Null and empty values are skipped.
- Type: record `type` string; `--type` still overrides; default `manual`.
- `--include-json`: directory scans skip `.json` files unless this flag is given; a `.json` path given directly is always read.
- Existing rules unchanged: unique titles per file, non-empty content, 8000-character cap, BOM tolerance, `sync_id` identity.
- Docs: `memorycmd` usage text, `docs/usage.md`, `docs/engram.md`, TUI path hint.
- Out of scope: per-record splitting of oversized records, YAML input, schema-specific renderers.

## Constraints

- Ordo still never writes Engram's database; output goes through `engram import`.
- No new deadcode: every new exported symbol must be reachable from `./cmd/gentle-ai`.
- Technical artifacts in English. Conventional Commits without AI attribution. No push/PR without separate authorization.

## Delivery

- Strategy: `ask-on-risk`. Forecast ~350 authored changed lines (under the ~400 budget).
- Branch: `feat/memory-import-json` from `main` at `3d40c514`.

## Tasks

- [x] T1 — `.json` parsing, title resolution, labeled content rendering, `--title-field` flag, tests (`internal/memorycmd`). Route: delegated direct (writer trigger: 2+ non-trivial files).
- [x] T2 — Docs and TUI hint list `.json` and `--title-field`. Route: delegated direct (same writer, mechanical).

## Acceptance Criteria

- A `documents.json`-shaped array imports with titles like `AUTH-001 — Inicio de sesion con RUT y clave` and readable labeled content.
- `{title, content, type}` arrays import identically to the same records as `.jsonl`.
- Directory scans read `.json` only with `--include-json`, and then every `.json` file must be a valid dataset; a `.json` path given directly needs no flag; invalid `.json` fails naming the file and record index.
- `go build ./... && go vet ./...`, `go test ./internal/memorycmd/... ./internal/tui/... ./internal/app/...`, `go run ./internal/gofmtcheck`, `./scripts/deadcode-ratchet.sh` pass.

## Progress

- T1 done in `0181a369` `feat(memory): import .json datasets with rendered content`: `internal/memorycmd/json.go` (ordered decode with `UseNumber`, title resolution, labeled rendering), `collect(path, collectOptions)` behind `Collect`, `--title-field` flag, usage text, `internal/app/help.go`, TUI hint, and tests. RED observed first (`json_test.go` failed to build: `undefined: collect`, `collectOptions`; then `flag provided but not defined: -title-field`), then GREEN.
- T2 done in `96102941` `docs(memory): document .json import and --title-field` (`docs/usage.md`, `docs/engram.md`). The TUI hint and its new test went with T1.
- Decisions: an object top level may hold other non-array fields but exactly one array field; a non-empty string `content` is used as is, any other `content` renders like the remaining fields (R3-001 fix); array items are numbered sequentially over non-empty items; nested labeled objects keep labels, while fields inside array items use raw keys; nesting deeper than 1000 levels fails.
- Checks (observed):
  - `go build ./... && go vet ./...`: pass.
  - `go test ./internal/memorycmd/... ./internal/tui/... ./internal/app/... -count=1`: all ok.
  - `go run ./internal/gofmtcheck`: pass.
  - `./scripts/deadcode-ratchet.sh`: "no new unreachable functions" (it also notes 3 pre-existing baselined entries are now reachable; baseline untouched).
  - Sample `documents.json` dry run: `obs-e6826728c4bb66bd  AUTH-001 — Inicio de sesion con RUT y clave  (documents.json)`, `1 entries (dry run, nothing written)`.

- Parent spot check: `go test ./internal/memorycmd/... -count=1` re-run: ok.
- RDD: `ordo review assess --base-ref main --committed-only` → medium (`executable_change` in `internal/app/help.go`), `review_due` (`slice_budget_reached`, 699 changed lines). Consent granted; lineage `review-e5facf6f539b23c5`, one lens (`review-reliability`) → approved; acknowledged, authority burned. Reviewed boundary advances to `d4ec61bd`.
- Non-blocking follow-ups from that review (separate later work):
  - R3-001 (WARNING) `json.go:184`: a non-string or blank `content` field is dropped silently; docs imply it would render. Untested. **Fixed**: non-string `content` now renders as `Content:`; covered by `TestCollectJSONRendersNonStringContent` (RED observed: `entry "B": content is empty`, then GREEN).
  - R3-002 `json.go:83-85`: no test for the `maxJSONDepth` guard.
  - R3-003 `json.go:251-253`: continuation lines of a multi-line top-level scalar are not indented.
  - R3-004 `memorycmd.go:118`: `--title-field` is silently ignored when no `.json` file is imported.

- Second review (lineage `review-0a0325e3d856d661`, approved, acknowledged) raised a new WARNING: directory scans picked up every `.json`, so an unrelated `package.json` or `tsconfig.json` failed a previously working directory import. **Fixed**: `parseJSON` marks files that are not datasets (invalid JSON, not an array of objects) with `notDatasetError`; directory scans skip them and the CLI prints `skipped <file>: not a JSON dataset (...)` to stderr. A single file passed directly still fails, and record errors inside a real dataset still fail. Tests: `TestCollectDirectorySkipsNonDatasetJSON`, `TestCollectDirectoryStillFailsOnInvalidDatasetRecord`, `TestCollectSingleNonDatasetJSONFails` (RED: `unknown field onSkip`, then GREEN). The TUI skips silently; its preview lists the included sources.
- Remaining suggestions: `--title-field content` or `--title-field type` collide with the content/type fields (untested).

- Third review (lineage `review-42582ea8f0823665`, approved, acknowledged) found the skip heuristic two-sided: an object holding one array of objects (web manifest `icons`) still counted as a dataset and failed directory imports, while a truncated real dataset was skipped with only a notice. User chose the strict option: directory scans ignore `.json` unless the CLI's `--include-json` is given, and then every `.json` file must be a valid dataset (no skipping). A `.json` path given directly needs no flag. The `notDatasetError`/`onSkip` skip machinery from `83a49908` was removed. The TUI scans directories without `.json`; a `.json` file can be entered directly. Route: direct inline (mechanical edits in already-mapped code). Tests: `TestCollectDirectoryIgnoresJSONByDefault`, `TestCollectDirectoryIncludesJSONWhenAsked`, `TestCollectDirectoryIncludedJSONMustBeDatasets`, `TestCollectSingleJSONFileNeedsNoFlag`, `TestRunDryRunIncludeJSON` (RED: `unknown field includeJSON`, then GREEN).

## Next Step

- Delivery is the user's decision. The branch has 699 changed lines (over the 400-line PR policy): chain the PR or request `size:exception`. Optionally address R3-001..R3-004 first.
