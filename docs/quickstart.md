# Quickstart

Ordo is based on Gentle AI. It configures the AI coding agents you already have; it never installs an agent for you.

## Prerequisites

- macOS or Linux (Ubuntu/Debian, Arch, or Fedora/RHEL family). Windows is not supported yet.
- `curl` and `git` on `PATH`.
- At least one supported agent installed (for example Claude Code or OpenCode).

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/josefosaurus/Ordo-ai/main/scripts/install.sh | bash
```

The installer downloads the signed release for your platform from [GitHub Releases](https://github.com/josefosaurus/Ordo-ai/releases), verifies its checksum, and installs `ordo`. To choose the folder:

```bash
curl -fsSL https://raw.githubusercontent.com/josefosaurus/Ordo-ai/main/scripts/install.sh | bash -s -- --dir ~/.local/bin
```

If `ordo` is not found afterwards, the install folder is not on your `PATH`:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc   # bash: ~/.bash_profile
```

Check it:

```bash
ordo --version
```

## Upgrade

```bash
ordo upgrade
```

`ordo upgrade` downloads the latest signed release, verifies its signature against Ordo's release key, and replaces the binary.

## Build from source

```bash
git clone https://github.com/josefosaurus/Ordo-ai.git && cd Ordo-ai
go build -o ordo ./cmd/gentle-ai     # requires Go 1.25.10+
```

Do not use `go install github.com/gentleman-programming/gentle-ai/...`: the module path still names upstream, so it installs Gentle AI rather than Ordo.

## Run

```bash
ordo install --dry-run
```

Use `--dry-run` first to validate selections and execution plan without applying changes. The dry-run output includes a `Platform decision` line showing the detected OS, distro, package manager, and support status.

## First real install

```bash
ordo install
```

The installer detects your platform automatically — no flags needed to select macOS vs Linux. Install commands are resolved through the appropriate package manager (brew, apt, pacman, or dnf) based on detection.

After completion, verify that agent configs and selected components were installed to their expected paths.

The agents you select during install become the default scope for future `ordo sync` runs. Ordo records that selection in `~/.gentle-ai/state.json` and does not automatically sync every agent config directory that exists on your machine. To check what will be updated after an upgrade, run:

```bash
ordo sync --dry-run
```

To update a different set explicitly, pass every target agent:

```bash
ordo sync --agent claude-code --agent opencode
```

## Verification outcome

When checks pass, installer reports:

`You're ready. Run 'claude' or 'opencode' and start building.`

If something looks wrong after install, run `ordo doctor` for a read-only health check. It verifies tool binaries, `state.json` validity, Engram™ MCP reachability, and disk space — each check reports pass/warn/fail with a remedy hint.

Gentle Pi is not supported by Ordo yet. For reference, upstream's Pi-only flow shows the Pi package stack instead of the regular components. It installs `gentle-pi` and `gentle-engram`, runs `pi-engram init` through the pinned `gentle-engram` package, then installs `pi-web-access` and `pi-btw`. Pi's built-in MCP support (Pi >= 0.99.0) runs the Engram and CodeGraph MCP servers from `mcp.json`. It removes a previously installed `pi-mcp-adapter`, because an extension that registers `/mcp` replaces Pi's built-in MCP support.

## Start working with ODD

Open your agent in the project and describe an outcome, for example: "Add CSV export using the existing report filters." [Organic Driven Development (ODD)](usage.md#organic-driven-development-odd) is the development workflow: explore, implement authorized work, and check it. Substantial work keeps one recoverable feature document; small/read-only work avoids durable artifacts.

## Hardening recommendations for users

Ordo pins versions and disables postinstall scripts on every npm install it generates. When you install the `permissions` component, a sensitive-paths deny list is applied to Claude Code and OpenCode blocking access to `~/.ssh/*`, `**/*.pem`, `**/*.key`, `**/.env*`, `~/.aws/credentials`, and other credential paths. See [Components](../docs/components.md) for the full list.

For broader protection across npm packages you install yourself, set these once on your machine:

- `npm config set ignore-scripts true` — blocks postinstall scripts globally; the primary supply-chain attack vector.
- `npm config set min-release-age 3` — skip packages published in the last 3 days; catches malicious typosquats before you install them.
- `npm config set allow-git none` — block git: dependencies, which can be moving targets.

Optional wrapper tools for extra defense:

- [`npq`](https://github.com/lirantal/npq) — audits a package against several heuristics before it installs.
- [`sfw`](https://socket.dev/) (Socket Firewall) — runtime guard that intercepts suspicious behavior at install/run time.

## Unsupported platforms

If you run the installer on an unsupported OS or Linux distro, it exits immediately with an error:

- `unsupported operating system: only macOS, Linux, and Windows are supported (detected <os>)`
- `unsupported linux distro: Linux support is limited to Ubuntu/Debian, Arch, and Fedora/RHEL family (detected <distro>)`
