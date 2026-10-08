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
- [ ] T2 — Slice 2: agent-facing assets.
- [ ] T3 — Slice 3: README, meta, assets, themes.
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

## Next Step

- T2 (Slice 2: agent-facing assets).
