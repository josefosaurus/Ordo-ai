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
  - [x] T3d — Ordo theme replacing the upstream themes (user choice 2026-10-08). Route: delegated direct (writer trigger: 2+ non-trivial files). Commits `4c149602`, `e36c4545`, `5af580d5`. One `ordo` theme (Rosé Pine) for Claude Code and OpenCode, overlay `{"theme":"ordo"}`; install/sync migrate legacy `gentleman`/`Gentleman Cute`/`gentleman-cute` selections and remove legacy theme files only on a sha256 match with the bytes Ordo shipped; uninstall cleans ordo and legacy files and clears only managed theme values. The user approved the extra surfaces `internal/cli/run.go`, `run_test.go`, `sync_test.go`, `run_component_paths_test.go` and `internal/update/upgrade/executor.go`/`executor_test.go`.
- [ ] T4 — Slice 4: docs.
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

- T3d done (`4c149602`, `e36c4545`, `5af580d5`).
  - RED observed before implementation: theme package build failure (`LegacyVisualThemePaths`, `ordoClaudeTheme`, `IsManagedThemeName` undefined); with old-behavior stubs 9 theme tests failed (overlay still `gentleman`, Ordo assets missing, migration not applied). Uninstall: `TestPartialUninstallClaudeThemeRemovesOnlyThemeAssets` (`ordo.json` kept) and `TestPartialUninstallClaudeThemeClearsOnlyManagedThemeSelection/{ordo,legacy_gentleman,legacy_cute}` failed. CLI: `TestComponentPathsVisualThemesMatchSelectedAdapter` and `TestInstallVisualThemesMigratesLegacySelection` (backup targets missing settings) failed; upgrade `TestConfigPathsForBackup_CoversManagedAgentPaths` missed the legacy files. `TestRunSyncMigratesLegacyVisualThemeSelection` was written after the theme migration landed and passed on first run.
  - Ownership: no ownership record exists for theme files. Migration proves ownership by sha256 of the exact bytes earlier releases wrote (digests computed from the removed palettes); unmatched or non-regular files stay and only the legacy setting switches. Uninstall keeps the existing path-based removal for `ordo.json` and the legacy paths.
  - Backups: install `backupTargets`, sync `syncBackupTargetsScoped` and upgrade backups include the legacy files and the settings file (backup-only; verification stays on `ordo.json`). The sync part landed in `e36c4545` (user approved `internal/cli/sync.go`). RED: `TestRunSyncMigratesLegacyVisualThemeSelection` failed with sync backup targets missing `.claude/settings.json`; GREEN after the backup-only hook. Re-check: `internal/cli` ok (1213s), components/update no failures, gofmtcheck pass, deadcode ratchet: no new unreachable functions.
  - Review fixes (`5af580d5`), from the native review warnings R4-settings-parse-hard-fail, R3-visual-theme-settings-parse-failure, R3-uninstall-theme-settings-parse-failure and R2-001:
    - Install and sync: unreadable or unparseable agent settings skip only the legacy selection switch; `ordo.json` is still written, owned legacy files are still removed, and the broken file is left byte-for-byte untouched (no result channel exists for a note, so the skip is silent).
    - `claude-theme` uninstall: unparseable settings skip only the managed-theme clearing; theme files are still removed. `theme` uninstall keeps its existing behavior (a parse error still fails, as before this branch).
    - OpenCode paths: theme assets stay under the adapter config root (`GlobalConfigDir`), like plugins, skills and `AGENTS.md`, which ignore `OPENCODE_CONFIG_DIR`. The theme selection follows the effective settings file, as every other settings writer does. Install, sync and upgrade backups, verification and uninstall all resolve through `VisualThemePaths`/`LegacyVisualThemePaths` and `VisualThemeSettingsPath`.
    - RED: `TestInjectVisualThemesSkipsSettingsMigrationForMalformedSettings`, `TestInstallVisualThemesToleratesMalformedSettings` and `TestSyncVisualThemesToleratesMalformedSettings` failed with `parse settings ...`; `TestPartialUninstallClaudeThemeToleratesMalformedSettings` failed with `unmarshal json object: unexpected EOF`. `TestOpenCodeVisualThemePathsWithConfigDirOverride` pins the existing behavior and passed on first run.
    - Checks: `go build ./... && go vet ./...` pass; components/update no failures; `internal/cli` ok (1263s); gofmtcheck pass; deadcode ratchet: no new unreachable functions.
  - GGA: `internal/components/gga/config.go:96` writes the `# Gentleman Guardian Angel Configuration` comment naming the third-party GGA tool, not a theme; kept.
  - Catalog text already neutral. Goldens: `go test ./internal/components/ -update` changed nothing. Bench: no theme assertions to change.
  - Checks: `go build ./... && go vet ./...` pass; components/catalog/tui/update tests no failures; `internal/cli` ok (1206s); `bash -n e2e/e2e_test.sh` ok; `gofmtcheck` pass; deadcode ratchet: no new unreachable functions.

## Next Step

- T4 (docs, including `docs/assets/brand`). Open decision: `deploy/telemetry` (upstream collector, unused by Ordo).
