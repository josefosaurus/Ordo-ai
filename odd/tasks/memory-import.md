# Memory Import (`ordo memory import`)

## Objective

Let users load curated "golden" knowledge (Markdown, CSV, JSONL) into their local Engram memory with one Ordo command, idempotently.

## Problem

Engram only accepts its own export JSON through `engram import <file>`. Hand-building that JSON is error-prone: observations need an existing session, a per-observation `project`, and a stable `sync_id`, otherwise re-imports duplicate every entry.

## Why

Teams want agents to start with shared conventions, decisions, and known-good answers. Ordo already installs and wires Engram, so a thin converter that reuses `engram import` keeps Ordo out of Engram's storage while making the workflow one command.

## Scope

- `ordo memory import <path> --project <name> [--type <t>] [--dry-run]`.
- Inputs: `.md` (each `##` section is one memory; whole file when no `##`), `.csv` / `.jsonl` with `title,content[,type]`; a directory scans those extensions.
- Converter emits Engram export JSON v0.2.0 with one session `ordo-import-<project>`, `project` on session and every observation, defaults for `type`/timestamps, and deterministic `sync_id` (`obs-` + hash of project, source path, record key).
- Runs `engram import` (resolving `engram` on PATH, with Homebrew fallback); `--dry-run` prints entries and writes nothing.
- Docs: `docs/usage.md`, `docs/engram.md`, one README line.
- Out of scope: TUI entry (later), semantic/vector datasets.

## Constraints

- Ordo never writes the Engram DB directly; only through `engram import`.
- New package `internal/memorycmd`; dispatch in `internal/app/app.go`, help in `internal/app/help.go`.
- Add `memory` to `documentedInvocationExecutionExclusions` in `internal/app/documented_invocation_test.go`.
- Tests that run real `engram` must isolate with `ENGRAM_DATA_DIR` and `HOME` set to temp dirs, and skip when `engram` is absent.
- Deadcode ratchet: no exported test-only helpers that are unreachable from `./cmd/gentle-ai`.
- Technical artifacts in English. No push/PR without separate authorization.

## Engram facts (verified on engram 2.2.1, isolated)

- Dedupe is by `sync_id` only; same id is skipped, newer `updated_at` updates; `topic_key` does not dedupe.
- Missing session for an observation fails the whole import atomically (FK error, exit 1).
- Project comes from each observation's own `project` field.

## Tasks

- [x] **T1 — Converter.** `internal/memorycmd` parses md/csv/jsonl (file or dir), validates title/content, and builds deterministic Engram export JSON. Unit tests incl. determinism. Route: delegated (writer; writer trigger: converter + tests). Commit: `e2c267a8`.
- [x] **T2 — Command.** `ordo memory import` with `--project`, `--type`, `--dry-run`; engram resolution + exec seam; app dispatch + help; documented-invocation exclusion; isolated real-engram test (skip if absent); docs. Route: delegated (same writer; writer trigger: 2+ non-trivial files). Commit: `46dff7de`.

- [x] **T3 — Review hardening.** Address advisory findings from the second review (`review-d5f4bb17440be7e0`, approved, non-blocking): skip hidden/vendored dirs in the walk; `sync_id` source = path relative to the git root (else absolute), so direct-vs-dir imports and same-basename files no longer collide; follow symlinks for mtime; `--force` flag stamps `updated_at` = now to push edits whose mtime did not advance; strip UTF-8 BOM; fence tracking by marker kind and length; dry-run derives labels from entries, not parallel indexes; named `sync_id` length constant with rationale; missing-engram message lists searched dirs; `engram import` runs with signal cancellation and temp-file cleanup; automated test for edit → `updated` and append → `imported`. Route: delegated (writer; 2+ non-trivial files). Commit: _recorded in the follow-up docs(odd) commit_.

## Acceptance criteria

- Importing the same input twice yields no duplicates (second run: 0 imported, N skipped).
- Missing `--project` or empty input fails with a clear usage error.
- `--dry-run` writes nothing.
- `go build ./... && go vet ./... && go test ./internal/memorycmd/... ./internal/app/... && go run ./internal/gofmtcheck && ./scripts/deadcode-ratchet.sh` pass.

## Delivery

- Forecast ~300–400 authored changed lines; strategy `ask-on-risk`.

## Decisions

- Timestamps: `created_at` = `updated_at` = source file mtime (UTC, seconds). Unchanged file → `skipped stale`; edited file → `updated` for all its entries; new entry → `imported`. A fresh checkout reports `updated` once, never duplicates. A constant timestamp was rejected because edits would never update; current time was rejected because every re-import would report `updated`.
- `--type` overrides the per-record `type`; default `manual`. Duplicate titles in one file are rejected (they would share a `sync_id`).
- Engram resolution: PATH, then `/opt/homebrew/bin`, `/usr/local/bin`, `/home/linuxbrew/.linuxbrew/bin`. Missing engram error points to `ordo install --agent <agent> --component engram`.

## Progress

- Branch `feat/memory-import` created from `main` (v0.1.3).
- T1 done (`e2c267a8`): RED observed (undefined `Entry`/`Collect`), then GREEN.
- T2 done (`46dff7de`): RED observed (undefined `lookPath`/`runEngram`), then GREEN.
- T3 done: after a no-behavior seam refactor (`BuildOptions`, context-taking `runEngram`, `Entry.Origin`), RED observed in `TestCollectSkipsHiddenAndVendoredDirs`, `TestSyncIDIsIndependentOfImportRoot`, `TestSyncIDDistinguishesSameBasenameInDifferentDirs`, `TestSyncIDOutsideGitUsesAbsolutePath`, `TestCollectFollowsSymlinkForModTime`, `TestCollectStripsUTF8BOM/{x.md,x.jsonl}`, `TestParseMarkdownFences` (4 of 5 cases; the unclosed-fence case already passed), `TestRunReportsEngramProblems/engram_missing_lists_searched_dirs`, `TestRunForceStampsUpdatedAtNow`, `TestRunInterruptRemovesTempFile`, `TestRunDryRunMatchesImportedIDs`, `TestRunImportsThroughEngram`, `TestRealEngramReimportIsIdempotent` (`--force` undefined; its new edit → `updated` and append → `imported` steps already passed). Then GREEN. `TestCollectWalkErrorNamesFileAndRecord` passed before the change (characterization).

## Verification evidence

- `go build ./... && go vet ./...`: pass.
- `go test ./internal/memorycmd/... ./internal/app/... -count=1`: ok (real-engram test ran, not skipped, engram 2.2.1).
- `go run ./internal/gofmtcheck`: pass. `./scripts/deadcode-ratchet.sh`: no new unreachable functions.
- Isolated smoke (built binary, `ENGRAM_DATA_DIR`/`HOME` in scratch): run 1 `2 imported, 0 updated, 0 skipped stale`; run 2 `0 imported, 0 updated, 2 skipped stale`; after appending a section `1 imported, 2 updated, 0 skipped stale`; missing `--project` exits 1 with usage error.
- RDD assessment: not run by the writer (parent owns review routing).
- RDD on `main..4ee1e0a7`: assessed high (process exec in `memorycmd.go`), consent granted, 4 lenses, approved with no blocking findings, acknowledged (lineage `review-a285a7caa74e5cb2`). Reviewed boundary advances to `4ee1e0a7`.
- Advisory follow-ups (non-blocking): R4 directory walk descends into `.git`/`node_modules` and aborts on first bad file; R3 `sync_id` differs when the same file is imported directly vs via a parent dir; R4 no timeout on `engram import`; R3 update/import-on-edit path lacks an automated test; R2/R3 fence toggle ignores fence kind; R3 mtime-only update trigger; R2 dry-run index coupling, missing-engram message omits Homebrew dirs, unexplained 16-hex `sync_id` length.

- Second RDD on `7358145..531b893e` (whole branch): high, granted, 4 lenses, approved, no blockers, acknowledged (`review-d5f4bb17440be7e0`). New advisory WARNINGs: basename `sync_id` collision across directories, mtime-only update trigger (cp -p/rsync/symlinks), BOM drops first section, fence kind toggle, dry-run index coupling. User authorized fixing them (T3) and opening the PR.

- T3 verification: `go build ./... && go vet ./...` pass (plus `GOOS=windows`/`linux go vet ./internal/memorycmd`); `go test ./internal/memorycmd/... ./internal/app/... -count=1` ok (real-engram test ran on engram 3.1.0); `go run ./internal/gofmtcheck` pass; `./scripts/deadcode-ratchet.sh` no new unreachable functions. Isolated smoke (built binary, `ENGRAM_DATA_DIR`/`HOME` in scratch, file inside a `.git` root): direct file import `2 imported, 0 updated, 0 skipped stale`; then its parent dir `0 imported, 0 updated, 2 skipped stale`; dry-run from the repo root and from the file show the same ids with paths `golden/faq.md` and `faq.md`.
- T3 deviation: `internal/app/help.go` usage line does not list `--force` (outside the writer's edit surface); `ordo memory help` does.

## Next step

T3 done; RDD assessment of the T3 commit (parent), then PR. Optional: add `--force` to `internal/app/help.go`. Previously: user decides whether to address the two advisory WARNINGs (walk scope, import-root-independent `sync_id`) before the PR; branch is ~1045 changed lines, over the 400-line PR budget, so the PR needs a chain strategy or `size:exception`. Push/PR remain user decisions. TUI entry is out of scope (later).
