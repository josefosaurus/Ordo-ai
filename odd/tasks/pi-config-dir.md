# Pi config dir — honor gentle-pi's config locators

Locator: `odd/tasks/pi-config-dir.md` · Engram topic: `odd/pi-config-dir/tasks` · Branch: `fix/pi-config-dir`

## Objective

Write the gentle-pi config files where gentle-pi reads them, and make Pi detection and the uninstall report follow an isolated Pi agent dir.

## Problem / Why

- gentle-pi 4.2.0 exports `GENTLE_AI_PI_CONFIG_DIR` (upstream gentle-ai #5462) to the Ordo processes Gentle Shell starts, naming the exact folder for `persona.json` and `background-subagents.json` (`lib/gentle-ai-binary.ts:118`). Ordo ignores it, so bundled Gentle Shell setups get the persona in `~/.pi/gentle-ai` instead of the bundled config dir that gentle-pi reads (`lib/agent-home.ts:16`).
- Persona (`internal/components/persona/inject.go:27`) ignores `GENTLE_PI_CONFIG_HOME`, while the sibling background policy (`internal/cli/pi_background.go:154`) honors it.
- `internal/system/config_scan.go:48` and `retainedPiResources` (`internal/components/uninstall/service.go:775`) hardcode `~/.pi/agent`, ignoring `PI_CODING_AGENT_DIR`.

## Scope

- One shared resolver for the global gentle-pi config folder: `GENTLE_AI_PI_CONFIG_DIR` (the folder itself) > `GENTLE_PI_CONFIG_HOME/gentle-ai` (current Ordo semantics, kept) > `<home>/.pi/gentle-ai`. Used by global persona and background policy. Workspace-scoped persona stays `<workspace>/.pi/gentle-ai/persona.json`.
- Pi detection scan and uninstall retained-resources report resolve agent-owned paths through the Pi agent dir (`PI_CODING_AGENT_DIR`).
- `docs/pi.md` isolated-home paragraph matches the new behavior.

## Constraints

- Default behavior (no env vars) unchanged; existing goldens stay green.
- Do not change `GENTLE_PI_CONFIG_HOME` semantics (gentle-pi compensates for the parent-vs-folder mismatch).
- Skill-registry roots unchanged: gentle-pi's extension also hardcodes `~/.pi/agent/skills`.
- Watch import cycles (`internal/system` must not import `internal/agents`).

## Tasks

- [x] T1 — Shared gentle-pi config-dir resolver; global persona + background policy use it (+ tests with each env var). Route: delegated (2+ non-trivial files).
- [x] T2 — Detection scan and uninstall retained report follow the Pi agent dir (+ tests). Route: delegated (2+ non-trivial files).
- [x] T3 — Update `docs/pi.md` isolated-home paragraph. Route: delegated with T2 writer.

## Acceptance criteria

- `GENTLE_AI_PI_CONFIG_DIR=X`: global install/sync writes `X/persona.json` and `X/background-subagents.json`.
- Only `GENTLE_PI_CONFIG_HOME=Y`: both files under `Y/gentle-ai/`.
- No env: both under `~/.pi/gentle-ai/` (unchanged).
- `PI_CODING_AGENT_DIR=Z`: Pi detected when `Z` exists; uninstall report lists retained resources under `Z`.

## Checks

- `go test ./internal/components/persona/... ./internal/cli/ -run 'Pi|Persona|Background' ./internal/system/... ./internal/components/uninstall/...`
- `go test ./internal/components/ -run Golden`
- `go build ./... && go vet ./... && go run ./internal/gofmtcheck`

## Progress / evidence

- T1 — commit `8bd8d7ad` (delegated writer): resolver `pi.GentlePiConfigDir` in `internal/agents/pi/gentle_pi_config.go` (persona and cli already import-safe; no cycle). Global persona via `persona.InjectGlobalPiPersona`/`PiGlobalPersonaConfigPath`; workspace persona unchanged; install (`injectPiPersonaScoped`, `piPersonaConfigPaths`) and sync (inject + backup paths) route through the same resolver; background policy uses it too. RED observed (new pi/persona tests failed to build; `TestPiConfigFilesFollowGentlePiConfigDir` failed against the old cli code), then GREEN.
  - `go test ./internal/agents/pi/ ./internal/components/persona/`: ok
  - `go test ./internal/components/ -run Golden -count=1`: ok
  - `go test ./internal/cli/ -run 'Pi|Persona|Background|Sync|ComponentPaths' -count=1 -timeout 30m`: ok (64.7s)
  - `go run ./internal/gofmtcheck`: clean; `./scripts/deadcode-ratchet.sh`: no new unreachable functions

- T2+T3 (delegated writer, one code+docs commit): `ScanConfigs` scans the resolved `PI_CODING_AGENT_DIR` for Pi (replicated resolution in `internal/system/config_scan.go` because `internal/agents/pi` imports `internal/system`; parity test `internal/system/pi_agent_dir_parity_test.go` compares against `pi.AgentConfigPath`). `retainedPiResources` uses `pi.AgentConfigPath` for agent-owned paths and `pi.GentlePiConfigDir` for the global gentle-ai folder. `docs/pi.md` isolated-home paragraph rewritten. RED observed for all new T2 tests, then GREEN.
  - `go build ./... && go vet ./...`: ok; `go run ./internal/gofmtcheck`: clean
  - `go test ./internal/agents/pi/... ./internal/components/persona/... ./internal/system/... ./internal/components/uninstall/...`: ok
  - `go test ./internal/components/ -run Golden -count=1`: ok
  - `go test ./internal/cli/ -run 'Pi|Persona|Background|Sync|ComponentPaths' -count=1 -timeout 30m`: ok (62.7s)
  - `./scripts/deadcode-ratchet.sh`: no new unreachable functions

## Next step

Native review assessment of the work-unit commits, then PR (user decision).
