# Remove Upstream Branding

## Objective

Remove every visible trace of the upstream Gentle AI brand from Ordo: binary output, agent-facing assets, repository docs and meta, logos and themes. The single remaining upstream mention is the legally required MIT copyright line in `LICENSE`, plus a short `NOTICE` that states the fork origin.

## Problem

Ordo is a fork of Gentle AI and still shows the upstream name in about 2,500 lines of user-visible text:
- CLI and TUI messages, and the help footer
- the `based on Gentle AI` attribution in the version line
- personas and orchestrator assets installed into agents, which also send agents to file defects in the upstream repository
- README, docs, issue templates, `package.json`
- upstream logos and themes

## Why

The upstream trademark policy (`TRADEMARKS.md`) requires forks to use distinct primary names and branding. The user also wants Ordo to carry no upstream brand trace beyond the fork notice.

## Scope

- Slice 1 (`chore/rebrand-binary-text`): user-visible binary output.
  - Hardcoded "Gentle AI", "Gentleman", "Gentle Logo", "Gentle Pi" and "AI Gentle Stack" strings in CLI, TUI, update and install messages.
  - Remove the brand `attribution` and the help-footer upstream link.
- Slice 2: agent-facing assets.
  - Persona display names, orchestrator text and its defect-report target (point to Ordo).
  - The `author` frontmatter of skills.
  - Review prompt and schema prose, only where contract-safe.
  - The persona file `name:` marker, with uninstall still matching the old text.
- Slice 3: README, top-level meta, `.github` templates, `package.json`, `NOTICE` and an Ordo `TRADEMARKS.md`.
  - Remove the upstream logo images and the "Gentleman" themes, the operator telemetry defaults, and the install script leftovers.
- Slice 4: `docs/**`.
  - Rewrite user docs and delete upstream-history documents (audits, PRDs, backlog dispositions).
- Optional slice 5: Go code comments.

## Constraints

- **Keep technical identifiers unchanged** (user decision, 2026-10-08). That covers:
  - the Go module path and `cmd/gentle-ai`
  - `~/.gentle-ai`, `GENTLE_AI_*` env vars and `gentle-ai.*` schema IDs
  - managed-section markers in user configs and stored persona or preset IDs (`gentleman`, `full-gentleman`)
  - agent and file IDs and telemetry metric names.

  These are protocol names, not brand use.
- **Keep functional upstream dependencies.** These are real external tools and packages: Engram downloads, GGA, the `gentle-pi` / `gentle-engram` npm packages, and the Homebrew tap.
- **Keep the `LICENSE` copyright line** (MIT requirement).
- **Do not change frozen contract content** (schemas or provider contract bundles) where a hash or compatibility check depends on it.
- Each slice is a stacked PR to `main`, within ~400 changed lines where it splits cleanly.

## Delivery

- Strategy: chain, `stacked-to-main` (user choice, 2026-10-08). Each slice branches from current `main` and merges on its own.
- Forecast: ~2,500 authored changed lines across 4–5 PRs.

## Tasks

- [x] T1 — Slice 1: binary output. Route: delegated direct (writer trigger: 2+ non-trivial files across ~25 files).
  - [x] T1a — CLI/TUI/update/install message strings and help footer (commit `2ee3e987`).
  - [x] T1b — Remove brand `attribution` (field, YAML, `Headline`, `ordo brand` show) and rename the telemetry notice (commit `815c1556`). The user approved the extra surfaces `internal/brandcmd/*`, `internal/tui/styles/styles_test.go` and `docs/telemetry.md` (notice block only).
- [x] T2 — Slice 2: agent-facing assets. Route: delegated direct (writer trigger: 2+ non-trivial files across ~60 files).
  - [x] T2a — Show the `gentleman` persona as "Mentor" (picker label, output style `name: Mentor`, legacy `outputStyle: "Gentleman"` migration and cleanup) and write the `name: Ordo Persona` marker while still recognizing `name: Gentle AI Persona`. Commit `0c754778`, with the 5 persona/combined goldens (user approved `testdata/golden/` as an extra surface).
  - [x] T2b — Replace the upstream brand in agent-facing assets (orchestrator defect handoff now targets `josefosaurus/Ordo-ai`, Hermes identity, OpenCode agent prompts, skill `author:` frontmatter, contract-safe review prose). Commit `e9aaadfc`, with the 8 skills goldens.
  - [x] T2c — Share the logo plugin label from `opencodeplugin.LogoPluginLabel()` and assert the dry-run header (commit `7ff9ccb2`).
- [ ] T3 — Slice 3: README, meta, assets, themes.
  - [x] T3b — README, top-level meta, issue templates, `package.json`. Route: delegated direct (writer trigger: 2+ non-trivial files). Commits `2bbce367` (add `NOTICE`, Ordo copyright line in `LICENSE`, delete upstream `TRADEMARKS.md`, README license footer) and `71043511` (README, `CONTRIBUTING.md`, `CONTRIBUTORS.md`, `AGENTS.md`, `package.json`, `.github/ISSUE_TEMPLATE/*`).
    - Decision: no Ordo `TRADEMARKS.md`; the user chose to delete the upstream one and keep attribution in `NOTICE` only.
  - [x] T3c — Skills and install scripts. Route: delegated direct (writer trigger: 2+ non-trivial files across 17 files). Repo-root `skills/` and the three parity-pinned embedded copies now use `author: ordo`, "Ordo" prose and `josefosaurus/Ordo-ai` repository references (commit `a43e693e`). `install.sh`/`install.ps1` drop "based on Gentle AI" and point at the Ordo raw URL, and `gentle-telemetry` defaults `--github-repo` to `josefosaurus/Ordo-ai` (commit `91524f45`). The unreachable Homebrew branch was removed from `install.sh`, because `detect_install_method` always returns `binary` before reaching any brew path.
  - [x] T3d — Ordo theme replacing the upstream themes (user choice 2026-10-08). Route: delegated direct (writer trigger: 2+ non-trivial files). Commits `4c149602`, `e36c4545`, `5af580d5`, `47bc45b3`, plus the keep-legacy-files commit. One `ordo` theme (Rosé Pine) for Claude Code and OpenCode, overlay `{"theme":"ordo"}`. Install and sync write `ordo.json` and migrate a legacy `gentleman`/`Gentleman Cute`/`gentleman-cute` settings selection to `ordo`; they never delete legacy `gentleman*.json` files (user decision 2026-10-11). `claude-theme` uninstall removes `ordo.json` and the legacy files and clears only managed theme values, resolving settings through `settingsTargets`. The user approved the extra surfaces `internal/cli/run.go`, `run_test.go`, `sync_test.go`, `run_component_paths_test.go` and `internal/update/upgrade/executor.go`/`executor_test.go`.
- [ ] T4 — Slice 4: docs. Branch `docs/rebrand-docs`, stacked on `feat/ordo-theme` (PR #14).
  - [x] T4a — User docs and brand assets. Route: delegated direct (writer trigger: 2+ non-trivial files). Rewrite visible upstream brand text in user docs (`docs/*.md`, not audits/history), remove unused upstream logo images in `docs/assets/brand`, fix `docs/goals.md` `TRADEMARKS.md` citation, rename visible upstream text in `deploy/telemetry` (file names and metric names stay).
    - Done: 22 user docs say Ordo/`ordo`, with factual fixes (no `ordo` Homebrew formula, no Windows `go install` self-upgrade, no default telemetry collector, signed comment `repo=josefosaurus/Ordo-ai`). 5 unreferenced upstream images deleted. `deploy/telemetry` visible text and Documentation URLs say Ordo. `docs/goals.md` is gitignored, so its local fix is not committed.
    - Kept: technical IDs, third-party names (Engram, GGA, `gentle-pi` "Gentle Shell"), the module path, `deploy/telemetry/install.sh` `REPO` (functional `--release-tag` download), and 3 dashboard strings pinned by `internal/telemetrycollector/runtime_dashboard_test.go`.
    - Open decisions: `docs/consulting.md` (upstream consulting pitch, unchanged), `docs/community-roadmap.md` (neutralized only), `docs/assets/features/gentle-shell.png`, pinned dashboard strings, telemetry `REPO`.
    - Checks: `go build ./... && go vet ./...` pass; doc-reading tests in `internal/assets`, `internal/app` (incl. `TestDocumentedInvocationsRunAsDocumented`), `internal/cli`, `internal/telemetry*`, `internal/recoverytrace`, `internal/update` ok.
  - [x] T4b — Delete upstream-history documents (audits, backlog dispositions, release ledgers) that no code, test or script reads. Route: delegated direct. Documents read by code (for example `internal/recoverytrace`, `refusal_resolution_ratchet_test.go`) stay.
    - Done: 13 docs deleted (4,564 lines): `docs/consulting.md` (user-approved 2026-10-11), `docs/release-v0.1.0-checklist.md`, `docs/releases/v2.2.0-closure-ledger.md`, four `docs/architecture/rdd-*` history docs (backlog disposition, wave7 deletion tracker, freeze expansion policy, ownership inventory) and six `docs/audits/*.md` audits. Links to them fixed in 3 kept docs.
    - Kept (read by code): `docs/audits/data/organic-rdd-recovery/**` and `docs/audits/2026-07-23-systemic-remediation-architecture.md` (`internal/recoverytrace`); `docs/audits/2026-07-27-detection-gap-audit.md` and `docs/architecture/rdd-root-simplification-design.md` (cited in Go comments).
    - Open decisions: `docs/architecture/rdd-shadow-evaluation.md` (looks stale), `docs/architecture/the-organic-rdd-story.md` (upstream narrative), `docs/community-roadmap.md`.
    - Checks: `go build ./... && go vet ./...` pass; `internal/recoverytrace`, `internal/app`, `internal/assets` ok; `internal/cli -run 'Ratchet|Refusal|Guard|Doc'` ok; no remaining references to deleted paths outside `odd/`.
- [ ] T5 — Optional: code comments.

## Acceptance Criteria

- Running `ordo` (help, version, TUI screens, install, sync, upgrade, telemetry, dry-run) shows no "Gentle", "Gentleman" or upstream URL, except functional third-party tool names (Engram, GGA, Pi package names).
- Installed agent assets carry no upstream brand name, and defect reports target `josefosaurus/Ordo-ai`.
- README and docs carry no upstream brand. The only upstream mentions are in `LICENSE` and `NOTICE`.
- Each slice passes:
  - `go build ./... && go vet ./...`
  - `go test ./...`
  - `go run ./internal/gofmtcheck`
  - `./scripts/deadcode-ratchet.sh`

## Progress

- Inventory done (read-only mapper): about 9,000 matching lines.
  - About 6,200 are technical identifiers, which are kept.
  - About 80 are user-visible binary strings, across about 30 files.
  - About 225 are asset lines, across 55 files.
  - About 1,300 are docs and meta lines, across 121 files.
  - About 580 are comments.

- T1a done (`2ee3e987`, 25 files, +58/−42).
  - Display names now use `brand.Current().Name` / `brand.Command`; self-upgrade hints use `r.Tool.Owner`/`r.Tool.Repo`; Ordo install hint uses `brand.ReleaseOwner`/`brand.ReleaseRepo`.
  - Kept: GGA, Engram, Homebrew tap URLs and refs; `gentle-logo` plugin ID; `gentle-ai-bench` skill ID; `--persona gentleman` notice (IDs only).
  - Checks: `go build ./... && go vet ./...` pass; `go test` passes for all packages (`internal/cli` needs `-timeout 60m`, ~1280s); `gofmtcheck` pass; deadcode ratchet: no new unreachable functions. No refusal-baseline or golden changes.

- T1b done (`815c1556`, 9 files, +28/−26).
  - `Headline` is now `"<name> <version>[ — <tagline>]"`; `ordo --help` starts with `Ordo dev`; `ordo brand show` no longer lists an attribution; a legacy `attribution:` key in a user override is ignored, and a test covers this.
  - `NoticeLine` now starts with "Ordo sends…"; only the matching block in `docs/telemetry.md` changed.
  - Checks: `go build ./... && go vet ./...` pass; `go test ./internal/brand/... ./internal/brandcmd/... ./internal/tui/... ./internal/telemetry/... ./internal/app/... -count=1` all ok; `gofmtcheck` pass; deadcode ratchet: no new unreachable functions.

- T2 done (`7ff9ccb2`, `0c754778`, `e9aaadfc`).
  - RED observed before implementation: `PersonaLabel` / `removeManagedOutputStyleSetting` / `LogoPluginLabel` undefined (build failures); with a stub keeping the old behavior, `TestInjectClaudeGentlemanSelectsMentorOutputStyle`, `TestInjectMigratesLegacyGentlemanOutputStyleToMentor/{install,sync}` ("outputStyle = Gentleman, want Mentor"), `TestRemoveManagedOutputStyleSetting/current_Mentor`, `TestInjectVSCodeWritesOrdoPersonaMarker`, `TestInjectVSCodeReplacesLegacyPersonaMarker` and `TestRemoveManagedPersonaPreamble_RecognizesOrdoAndLegacyMarkers/name:_Ordo_Persona` failed. The dry-run header test passed immediately (Slice 1 already implemented it).
  - Kept unchanged because a released or frozen contract pins them: `LensResultSchema`, `RefuterResultSchema`, `TargetedValidatorResultSchema` titles and `bundleREADME` (shipped in the provider contract bundle at `CONTRACT_SEMVER` 1.2.0, whose manifest binds each file's sha256); the capture-result dry-run schema title (byte-equal mirror of `contracts/review-integration/v2/schemas/capture-result-dry-run.schema.json`).
  - Kept unchanged because `TestPublicBundledSkillsMatchEmbeddedAssets` / `TestRDDDefectWorkflowSkillContract` require byte parity with repo-root `skills/` (Slice 3): `systemic-issue-triage`, `gentle-ai-bench`, `rdd-defect-workflow` embedded skills.
  - Goldens: 13 files under `testdata/golden/` regenerated with `go test ./internal/components/ -update` (14 lines each way: skill `author:`, Mentor output style, persona style line).
  - Checks: `go build ./... && go vet ./...` pass; non-cli `go test` pass; `internal/cli` pass (1297s); `gofmtcheck` pass; deadcode ratchet: no new unreachable functions. No refusal-baseline changes.
  - Kept as technical IDs: persona ID `gentleman`, `output-style-gentleman.md` / `gentleman.md` file names, Kimi agent `name: gentleman`, OpenCode `agent.gentleman`, TS identifiers and relay registry keys in the OpenCode review transport plugins, `gentle-ai.*` schema IDs.

- T3b done (`2bbce367`, `71043511`).
  - Audit `command grep -n -i -E "gentle|gentleman|buscaglia"` over the touched files leaves only the `NOTICE` fork lines, the `LICENSE` MIT copyright line, and technical identifiers (`cmd/gentle-ai`, `~/.gentle-ai`, `gentle-ai-*` skill IDs and paths, the `gentle-pi` package).
  - Also fixed: bug template version hint now says `ordo version` (was `gga version`); CONTRIBUTING run step is `./ordo` and clone dir `Ordo-ai`.
  - Checks: `node --test .github/scripts/*.test.cjs` 26/26 pass; issue templates parse as YAML; `package.json` parses; `go build ./...` pass; `go test ./internal/assets/ ./internal/app/` ok (they read `AGENTS.md`, `CONTRIBUTING.md`, `README.md`).
  - Left for Slice 4: `docs/goals.md` still cites the deleted `TRADEMARKS.md`.
- T3c done (`a43e693e`, `91524f45`).
  - Kept as technical IDs or functional references: skill IDs (`gentle-ai-branch-pr`, `gentle-ai-chained-pr`, `gentle-ai-bench`, `gentle-ai-collab-perfect`) and the `gentle-ai-bench run` command; `GO_MAIN_PACKAGE="gentle-ai"`, `GENTLE_AI_CHANNEL`, and the `github.com/gentleman-programming/gentle-ai/v4` module path in `install.ps1` (`TestWindowsInstallAndUpgradeContainNoRemoteBinaryOrScriptPath` requires that exact source-install command); the `gentle-telemetry` command name and module import; `gentle-pi`/`gentle-engram` npm defaults.
  - Goldens: none changed (`go test ./internal/components/ -update` produced no diff, because no golden embeds these skills).
  - Checks: `go build ./... && go vet ./...` pass; `go test ./internal/assets/... ./internal/components/... ./cmd/... ./internal/app/... ./internal/update/... -count=1` has no failures (24 packages ok); `bash scripts/test-install-module-path.sh` passes 6/6; `bash -n scripts/install.sh` ok; `gofmtcheck` pass; deadcode ratchet: no new unreachable functions.

- T3d done (`4c149602`, `e36c4545`, `5af580d5`, `47bc45b3`, keep-legacy-files commit). The history below records each step; the last bullet states the final behavior.
  - RED observed before implementation: theme package build failure (`LegacyVisualThemePaths`, `ordoClaudeTheme`, `IsManagedThemeName` undefined); with old-behavior stubs 9 theme tests failed (overlay still `gentleman`, Ordo assets missing, migration not applied). Uninstall: `TestPartialUninstallClaudeThemeRemovesOnlyThemeAssets` (`ordo.json` kept) and `TestPartialUninstallClaudeThemeClearsOnlyManagedThemeSelection/{ordo,legacy_gentleman,legacy_cute}` failed. CLI: `TestComponentPathsVisualThemesMatchSelectedAdapter` and `TestInstallVisualThemesMigratesLegacySelection` (backup targets missing settings) failed; upgrade `TestConfigPathsForBackup_CoversManagedAgentPaths` missed the legacy files. `TestRunSyncMigratesLegacyVisualThemeSelection` was written after the theme migration landed and passed on first run.
  - Ownership: no ownership record exists for theme files. Until the keep-legacy-files commit, install and sync proved ownership by sha256 of the bytes earlier releases wrote and deleted only matching files; that path is gone. Uninstall keeps the existing path-based removal for `ordo.json` and the legacy paths.
  - Backups: install `backupTargets` and sync `syncBackupTargetsScoped` include the settings file (backup-only; verification stays on `ordo.json`). They included the legacy files until the keep-legacy-files commit. Upgrade backups still include the legacy files. The sync part landed in `e36c4545` (user approved `internal/cli/sync.go`). RED: `TestRunSyncMigratesLegacyVisualThemeSelection` failed with sync backup targets missing `.claude/settings.json`; GREEN after the backup-only hook. Re-check: `internal/cli` ok (1213s), components/update no failures, gofmtcheck pass, deadcode ratchet: no new unreachable functions.
  - Review fixes (`5af580d5`), from the native review warnings R4-settings-parse-hard-fail, R3-visual-theme-settings-parse-failure, R3-uninstall-theme-settings-parse-failure and R2-001:
    - Install and sync: unreadable or unparseable agent settings skip only the legacy selection switch; `ordo.json` is still written and the broken file is left byte-for-byte untouched (no result channel exists for a note, so the skip is silent).
    - `claude-theme` uninstall: unparseable settings skip only the managed-theme clearing; theme files are still removed. `theme` uninstall keeps its existing behavior (a parse error still fails, as before this branch).
    - OpenCode paths: theme assets stay under the adapter config root (`GlobalConfigDir`), like plugins, skills and `AGENTS.md`, which ignore `OPENCODE_CONFIG_DIR`. The theme selection follows the effective settings file, as every other settings writer does. Install, sync and upgrade backups, verification and uninstall all resolve through `VisualThemePaths`/`LegacyVisualThemePaths` and `VisualThemeSettingsPath`.
    - RED: `TestInjectVisualThemesSkipsSettingsMigrationForMalformedSettings`, `TestInstallVisualThemesToleratesMalformedSettings` and `TestSyncVisualThemesToleratesMalformedSettings` failed with `parse settings ...`; `TestPartialUninstallClaudeThemeToleratesMalformedSettings` failed with `unmarshal json object: unexpected EOF`. `TestOpenCodeVisualThemePathsWithConfigDirOverride` pins the existing behavior and passed on first run.
    - Checks: `go build ./... && go vet ./...` pass; components/update no failures; `internal/cli` ok (1263s); gofmtcheck pass; deadcode ratchet: no new unreachable functions.
  - Second review fixes (`47bc45b3`), from R4/R3-locked-settings and R4-silent-skip-leaves-dangling-legacy-selection:
    - A refused (locked or symlinked) OpenCode settings file or a failed settings write now skips the selection migration without an error; `ordo.json` is still written.
    - Owned legacy theme files were removed only when the selection switched to `ordo`, or the settings were missing or readable and selected no legacy theme; a skipped migration kept them. Superseded by the next bullet.
    - RED: `TestInjectVisualThemesKeepsLegacyFilesWhenSettingsLocked` failed with `refuse to rewrite settings ...`; `TestInjectVisualThemesKeepsLegacyFilesWhenSettingsUnreadable` and the updated `TestInjectVisualThemesSkipsSettingsMigrationForMalformedSettings` failed because owned legacy files were deleted. The new `TestInjectVisualThemesRemovesOwnedLegacyFilesWhenSelectionNotLegacy` and the existing migration tests pass.
    - Checks: `go build ./... && go vet ./...` pass; components/update no failures; `internal/cli` ok (1251s); gofmtcheck pass; deadcode ratchet: no new unreachable functions.
  - Keep legacy files (user decision 2026-10-11), commit `refactor(theme): keep legacy theme files until uninstall`:
    - Install and sync never delete legacy `gentleman*.json` files, so no selection can dangle, including a project-scoped one the step never reads. The sha256 ownership check and `removeOwnedLegacyTheme` are removed. Only `claude-theme` uninstall removes the legacy files.
    - Install and sync backup sets drop the legacy files and keep the settings file.
    - Tests: theme migration tests now assert legacy files are kept after a switch, after a non-legacy selection and for user-edited files (`TestInjectVisualThemesKeepsLegacyFilesWhenSelectionNotLegacy`, `TestInjectVisualThemesKeepsEditedLegacyFilesButSwitchesSetting`). `TestInstallVisualThemesMigratesLegacySelection` and `TestRunSyncMigratesLegacyVisualThemeSelection` assert install and sync keep an edited and an unedited legacy file and leave them out of the backup set. Uninstall deletion tests unchanged. No RED: the change removes behavior, so the old deletion assertions failed first (4 theme tests) and were rewritten.
    - Checks: `go build ./... && go vet ./...` pass; components/update no failures; `internal/cli` ok (1262s); gofmtcheck pass; deadcode ratchet: no new unreachable functions.
  - GGA: `internal/components/gga/config.go:96` writes the `# Gentleman Guardian Angel Configuration` comment naming the third-party GGA tool, not a theme; kept.
  - Catalog text already neutral. Goldens: `go test ./internal/components/ -update` changed nothing. Bench: no theme assertions to change.
  - Checks: `go build ./... && go vet ./...` pass; components/catalog/tui/update tests no failures; `internal/cli` ok (1206s); `bash -n e2e/e2e_test.sh` ok; `gofmtcheck` pass; deadcode ratchet: no new unreachable functions.

## Next Step

- **Finish T3d on `feat/ordo-theme` (resume here).** User decision (2026-10-11): install and sync no longer delete legacy `gentleman*.json` files. They only write `ordo.json` and migrate a legacy *selection* (outcomes switched / not-legacy / skipped). Legacy files are removed only by `claude-theme` uninstall.
  - An interrupted agent left uncommitted, untested edits in `internal/components/theme/inject.go` (deletion path removed, about −80 lines), `internal/cli/run.go` and `internal/cli/sync.go` (backup sets).
  - Backups: keep the settings path, which the migration still rewrites. Drop the legacy files from the install and sync backup-only sets, because they are no longer deleted there. Keep them where uninstall or upgrade can remove them.
  - Tests: remove the install/sync legacy-deletion assertions. Add a test that install and sync keep owned legacy files after a successful switch. Keep the uninstall deletion tests.
  - Rewrite the T3d bullets above once, consistently. Fix the contradiction around the "owned legacy files are still removed" bullet, and state that uninstall resolves settings through `settingsTargets`.
  - Commit as `refactor(theme): keep legacy theme files until uninstall` (no AI trailer). Verify `go build/vet`, `go test ./internal/components/... ./internal/update/...`, `go test ./internal/cli/ -timeout 60m` (about 21 min), `gofmtcheck` and `deadcode-ratchet`.
  - Then: one native review of the branch, then a PR with `type:chore` + `size:exception` (user-approved).
- Then T4: docs, including `docs/assets/brand`. Rename the visible upstream text in `deploy/telemetry` (user decision); file names and metrics stay.
