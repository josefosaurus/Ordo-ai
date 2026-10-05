<!-- markdownlint-disable-next-line MD041 -->
<a id="top"></a>

<div align="center">

<pre>
 ██████╗ ██████╗ ██████╗  ██████╗
██╔═══██╗██╔══██╗██╔══██╗██╔═══██╗
██║   ██║██████╔╝██║  ██║██║   ██║
██║   ██║██╔══██╗██║  ██║██║   ██║
╚██████╔╝██║  ██║██████╔╝╚██████╔╝
 ╚═════╝ ╚═╝  ╚═╝╚═════╝  ╚═════╝
</pre>

<h1>Ordo</h1>

<p><strong>Memory, workflow, and evidence for the AI coding agent you already use — with your team's brand and voice.</strong></p>

<p>
<a href="https://github.com/josefosaurus/Ordo-ai/releases"><img src="https://img.shields.io/github/v/release/josefosaurus/Ordo-ai?style=for-the-badge&labelColor=191724&color=c4a7e7" alt="Release"></a>
<img src="https://img.shields.io/badge/macOS%20%C2%B7%20Linux-9ccfd8?style=for-the-badge&labelColor=191724" alt="Platforms: macOS and Linux">
<a href="LICENSE"><img src="https://img.shields.io/badge/MIT-ebbcba?style=for-the-badge&labelColor=191724" alt="License: MIT"></a>
</p>

<sub>Ordo is an independent distribution <strong>based on <a href="https://github.com/Gentleman-Programming/gentle-ai">Gentle AI</a></strong>. It is not affiliated with or endorsed by the Gentle AI project.</sub>

</div>

## What it does

Ordo configures the agents you already have — Claude Code, OpenCode, Codex, Cursor, Gemini CLI, VS Code Copilot and more — so they share one workflow. It never installs an agent for you; it writes their native configuration and snapshots every file before it changes it.

| | |
| :--- | :--- |
| **Engram memory** | Agents save what they learn and check it before asking you again, so context survives restarts and compaction. [Docs](docs/engram.md) |
| **ODD workflow** | Small changes stay small; substantial work keeps one recoverable feature document. [Docs](docs/usage.md#organic-driven-development-odd) |
| **RDD review** | Finished work is frozen and reviewed at the depth its risk calls for, from a structural readback to four reviewer lenses. [Docs](docs/review-integration.md) |
| **Skills, MCP, permissions** | A shared skills library, optional Context7 docs, a security deny-list, and config backups. [Docs](docs/components.md) |
| **Your brand** | Name, tagline, logo, and colors are yours — per user. [Below](#make-it-yours) |
| **Your team's voice** | The `ordo` persona carries your team's tone, chat language, and engineering rules into every agent. [Below](#make-it-yours) |

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/josefosaurus/Ordo-ai/main/scripts/install.sh | bash
```

The installer downloads the signed release for your platform and verifies its checksum. If it installs into `~/.local/bin` and `ordo` is not found afterwards, add that folder to your `PATH`:

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc
```

Then:

```bash
ordo            # pick your agents, components, and persona
ordo doctor     # read-only health report
ordo upgrade    # update to the latest signed release
```

Windows is not supported yet; build from source with `go build -o ordo.exe ./cmd/gentle-ai`.

## Make it yours

Branding and persona are stored per user under `~/.gentle-ai/`, so everyone can tailor Ordo without forking it. Invalid values fall back to the defaults with a warning; they never break the CLI.

Run `ordo` and pick **Customize brand & persona** to edit them with a live preview, or use the commands below.

**Brand** — what the CLI and TUI show:

```bash
ordo brand set name "Acme"
ordo brand set tagline "Ship calmly"
ordo brand set logo ./logo.txt          # plain text, up to 24 lines × 80 columns
ordo brand set color.primary "#ff5f87"  # roles: primary accent text muted border success error warning highlight
ordo brand show
ordo brand reset
```

**Persona** — how your agents talk to you (code, docs, and commits always stay in English):

```bash
ordo persona set voice "Direct, pragmatic senior teammate."
ordo persona set language Spanish
ordo persona add-rule "Ship risky changes behind feature flags."
ordo persona show                       # includes exactly what agents receive
ordo sync                               # apply it to your agents
```

**Memory** — seed Engram with your team's curated knowledge (Markdown, CSV, or JSONL; re-imports never duplicate):

```bash
ordo memory import ./golden --project my-app
```

## Documentation

| Where | What |
| :--- | :--- |
| [Intended usage](docs/intended-usage.md) | The mental model — read this first |
| [Usage](docs/usage.md) · [Quickstart](docs/quickstart.md) | Commands, flags, and setup details |
| [Agents](docs/agents.md) | Supported agents and what each one gets |
| [Review integration](docs/review-integration.md) | The RDD contract and lifecycle |
| [Engram](docs/engram.md) · [Components](docs/components.md) | Memory, skills, presets, and personas |
| [Telemetry](docs/telemetry.md) | Ordo ships with telemetry off (no collector configured) |
| [Codebase guide](docs/CODEBASE-GUIDE.md) · [Contributing](CONTRIBUTING.md) | Working on Ordo itself |

The detailed docs are inherited from Gentle AI and may still use its name; commands work the same with `ordo`.

## Known limits

- macOS and Linux only; no Homebrew formula yet.
- Gentle Pi integration is not supported.

## License and trademarks

Ordo is released under the [MIT License](LICENSE), like the Gentle AI code it is based on. "Gentle AI" and "Engram" are trademarks of their owner; see [TRADEMARKS.md](TRADEMARKS.md). Ordo uses them only to describe what it is based on and compatible with.

<div align="right"><a href="#top">Back to top</a></div>
