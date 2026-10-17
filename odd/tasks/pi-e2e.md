# Pi e2e — install coverage in the Docker e2e suite

Locator: `odd/tasks/pi-e2e.md` · Engram topic: `odd/pi-e2e/tasks` · Branch: `test/pi-e2e`

## Objective

Cover a real `ordo install --agent pi` in the Docker e2e suite (`e2e/e2e_test.sh` Tier 2), like the other agents.

## Problem / Why

- The Docker e2e suite covers claude-code, opencode, codex, gemini-cli, qwen-code, and windsurf, but not Pi (no `pi`/`.pi` match under `e2e/`).
- Pi is the only agent whose install step runs commands: `pi install npm:<pkg>` per managed source and `npm exec --yes --package gentle-engram@latest -- pi-engram init`. The preflight needs `pi` and `npm` on PATH.

## Scope

- `e2e/lib.sh`: `setup_fake_pi_binary` helper (fake `pi` logging its argv; `npm` wrapper that fakes only `exec … pi-engram init` and delegates everything else to the real npm), cleanup of `~/.pi`.
- `e2e/e2e_test.sh`: Tier 2 Pi tests: persona install writes `~/.pi/gentle-ai/persona.json`, `pi install npm:gentle-pi` requested, engram absent from `~/.pi/agent/mcp.json`, idempotent re-install; optional Tier 1 dry-run.

## Constraints

- Offline and deterministic: no real npm downloads of Pi packages.
- Follow the existing fake-binary pattern (`setup_fake_engram_binary`).
- No product code changes.

## Tasks

- [x] T1 — Fake Pi tooling + cleanup in `e2e/lib.sh`, Pi Tier 2 tests in `e2e/e2e_test.sh`. Route: delegated writer (2 non-trivial files; mapping already done by an explorer).
- [ ] T2 — Run the Pi e2e locally in Docker (one platform) or report why not; CI covers all three.

## Acceptance criteria

- `RUN_FULL_E2E=1` Docker run passes the new Pi tests and the existing ones.
- New tests fail if the persona file is missing or if install stops calling `pi install`.

## Checks

- `bash -n e2e/lib.sh e2e/e2e_test.sh`
- Docker e2e on one platform (`cd e2e && ./docker-test.sh` subset) when Docker is available.

## Notes

- No RED: test-only addition against existing behavior; the smoke shows the assertions observe real install output.

## Forecast / delivery

~150 authored lines, under the 400 budget; single PR or direct push per user.

## Progress / evidence

- T1 — delegated writer (trigger: 2 non-trivial files; mapping by a prior explorer). `e2e/lib.sh`: `setup_fake_pi_binary` (fake `pi` logging argv to `PI_CALLS_LOG`; `npm` wrapper faking only `exec … pi-engram init`, real npm resolved with the shim dir stripped from PATH), `cleanup_test_env` removes `~/.pi` and the log. `e2e/e2e_test.sh`: Tier 1 `test_dry_run_agent_pi` (dry-run returns before the pipeline, so no preflight); Tier 2 Category 14 `test_pi_persona_install`, `test_pi_engram_not_in_mcp_json`, `test_pi_persona_idempotent` (`cmp -s`). +138 lines. Observed install calls: `pi install npm:gentle-pi`, `npm:gentle-engram`, `npm exec … pi-engram init`, `npm:pi-web-access`, `npm:pi-btw`.
  - `bash -n e2e/lib.sh && bash -n e2e/e2e_test.sh`: ok (parent spot check re-run: ok)
  - shellcheck: not installed, not run
  - Sandboxed macOS smoke (temp HOME, Pi env vars unset, Pi test block extracted): Tier 2 PASSED=8 FAILED=0; Tier 1 dry-run without `pi` on PATH PASSED=1; real `~/.pi` and `~/.gentle-ai` untouched
  - Docker e2e: pending (Docker unavailable locally) → T2 via CI
- T1 follow-up (inline, review advisories R3-001/R4-pi-shim-path-leak and R2-002): `cleanup_test_env` removes the `pi`/`npm` shims so later tests see the real tools; shims use a bash shebang to match their `%q` quoting. Sandboxed smoke: PASSED=8 FAILED=0, shim removed after cleanup, `npm` resolves to the real binary. Review of 85e3f33e: 4 lenses, approved, acknowledged; R1 (rm -rf ~/.pi, existing pattern) and R3-003 (absent mcp.json) left as noted. Review of 12386a74: 4 lenses, approved, acknowledged; advisory only: derive shim paths from one variable, `hash -r` after removing shims.
