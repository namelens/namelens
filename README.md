<img src="https://raw.githubusercontent.com/namelens/.github/main/assets/namelens-icon-128.png" width="64" align="right" alt="namelens">

# NameLens

[![Go Report Card](https://goreportcard.com/badge/github.com/3leaps/namelens)](https://goreportcard.com/report/github.com/3leaps/namelens)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/3leaps/namelens)](https://github.com/3leaps/namelens/releases)

**Naming, illuminated.**

## Overview

Naming shouldn't be a gamble. **Namelens** prevents the disasters that sink
projects—conflicts that only surface after you've printed business cards, built
websites, and committed to a brand.

We combine **precise technical checks** with **real-time internet
intelligence**:

**What we check:**

- **Domains** — RDAP with WHOIS fallback (.com, .io, .dev, .app, and more)
- **Package registries** — npm, PyPI, crates.io
- **Social handles** — GitHub (more coming soon)

**What we discover:**

- **Trademark conflicts** — AI-powered search across databases and the web
- **Brand associations** — sentiment analysis and hidden risks
- **Cultural fit** — phonetics, typeability, and cross-market suitability
- **Launch readiness** — comprehensive reports with clear guidance

**Who this is for:**

| Role                    | What Namelens gives you                                   |
| ----------------------- | --------------------------------------------------------- |
| **Founders**            | Confidence before printing business cards and filing LLCs |
| **Developers**          | Quick, CLI-native checks that fit your workflow           |
| **Product Teams**       | Competitive intelligence and brand gap analysis           |
| **Marketing**           | Risk visibility and social handle coordination            |
| **Side Project Makers** | Stop wasting time on unavailable names                    |

> **Proven in production**: We used Namelens to name itself. The tool caught a
> critical trademark conflict our initial codename had—saving us from an
> expensive rebrand.
> [Read the origin story →](docs/examples/namelens-origin-story.md)
>
> **Designed with itself**: The namelens icon was generated using
> `namelens mark`, validated through TinEye (0 matches), and adopted as our
> official brand mark.
> [Read the brand mark story →](docs/examples/namelens-brand-mark-story.md)

## Quick Start

```bash
# Install dependencies
make bootstrap

# Build
make build

# Configure AI backend (interactive wizard)
./bin/namelens setup

# Check a name
./bin/namelens check acmecorp --profile=startup

# Deep analysis with AI-powered brand research
./bin/namelens check acmecorp --expert --phonetics --suitability

# Start server (HTTP API for agents/automation)
./bin/namelens serve
```

**Five-second version**:

```bash
namelens check myproject
```

## See Namelens in Action

```bash
$ namelens check namelens --profile=startup --expert
╭────────┬──────────────┬────────────────┬───────────────────────────────────────────────────────────────────────╮
│ TYPE   │ NAME         │ STATUS         │ NOTES                                                                 │
├────────┼──────────────┼────────────────┼───────────────────────────────────────────────────────────────────────┤
│ domain │ namelens.app │ taken          │ exp: 2026-12-30; registrar: Dynadot LLC.                              │
│ domain │ namelens.com │ taken          │ exp: 2026-11-17; registrar: GoDaddy.com, LLC                          │
│ domain │ namelens.dev │ available      │                                                                       │
│ domain │ namelens.io  │ available      │                                                                       │
│ npm    │ namelens     │ available      │                                                                       │
│ pypi   │ namelens     │ available      │                                                                       │
│ github │ @namelens    │ taken          │ url: https://github.com/namelens                                      │
│ expert │ namelens     │ risk: low      │ No direct mentions as existing brand; highly available with low risks │
├────────┼──────────────┼────────────────┼───────────────────────────────────────────────────────────────────────┤
│        │              │ 4/8 AVAILABLE  │                                                                       │
╰────────┴──────────────┴────────────────┴───────────────────────────────────────────────────────────────────────╯
```

## CLI Commands

```bash
# Check a single name
namelens check <name>
namelens check <name> --profile=startup
namelens check <name> --tlds=com,io,dev
namelens check <name> --expert

# Generate name alternatives from a concept
namelens generate "static analyzer for shell scripts"
namelens generate "my product" --scan-dir ./docs
namelens generate "my product" --provider namelens-anthropic

# Compare candidates side-by-side
namelens compare name1 name2 name3
namelens compare name1 name2 --mode=quick

# Deep review with AI analysis
namelens review myproject --depth=deep

# Generate brand marks/logos
namelens mark "myproject" --out-dir ./marks --color brand
namelens image thumb --in-dir ./marks

# Batch check from file
namelens batch names.txt
namelens batch names.txt --output-format=json

# Generate context corpus for AI workflows
namelens context ./my-project --output=json

# Profile management
namelens profile list
namelens profile show startup

# Expert prompts
namelens ailink list

# Server mode (Control Plane HTTP API)
namelens serve                              # Foreground, localhost:8080
namelens serve --daemon                     # Background (daemon mode)
namelens serve --daemon --api-key nlcp_...  # With authentication
namelens serve status                       # Check daemon status
namelens serve stop                         # Stop daemon

# Setup and diagnostics
namelens setup                              # Interactive AI backend setup
namelens setup --provider xai --api-key KEY # Non-interactive setup
namelens version
namelens health
namelens doctor
namelens doctor ailink connectivity
```

### Recommended Workflow

```
generate → compare → review → mark → thumb
```

1. **Generate** candidates from your product concept
2. **Compare** 3-10 finalists with phonetics and suitability scores
3. **Review** top picks with deep brand analysis
4. **Mark** your chosen name with logo concepts
5. **Thumb** for sharing and AI agent workflows

## Profiles

Profiles define what to check for a given name:

| Profile     | Domains                                | Registries       | Handles |
| ----------- | -------------------------------------- | ---------------- | ------- |
| `startup`   | .com, .io, .dev, .app                  | npm, pypi        | github  |
| `developer` | .com, .io, .dev, .app, .sh, .org, .net | npm, pypi, cargo | github  |
| `minimal`   | .com                                   | -                | -       |
| `website`   | .com, .org, .net                       | -                | -       |
| `web3`      | .xyz, .io, .gg                         | npm              | github  |

## Configuration

NameLens uses environment variables with the `NAMELENS_` prefix, a YAML config
file, or a `.env` file:

```bash
# Core settings
NAMELENS_PORT=8080
NAMELENS_LOG_LEVEL=info
NAMELENS_DB_PATH=$XDG_DATA_HOME/namelens/namelens.db

# Control Plane API authentication (for namelens serve)
# NAMELENS_CONTROL_PLANE_API_KEY=nlcp_your_secret_key

# AILink providers (optional, for --expert and generate features)
# NAMELENS_AILINK_DEFAULT_PROVIDER=namelens-xai
# NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=your-api-key
```

Config file location: `~/.config/namelens/config.yaml` (XDG convention).

Copy `.env.example` to `.env` for local development. The server auto-loads
`.env` from the XDG config directory and the current working directory, or you
can specify a file explicitly with `namelens serve --env-file path/to/.env`.

See [docs/user-guide/configuration.md](docs/user-guide/configuration.md) for the
full configuration reference.

## Why Namelens?

**The hidden cost of bad names:**

- Rebranding mid-launch: $50,000–$200,000+ in legal, design, and opportunity
  costs
- Lost SEO momentum: Domain age and backlinks reset
- Customer confusion: Existing users can't find you
- Trademark infringement: Cease-and-desist letters at best, lawsuits at worst

**Namelens catches these before you invest:**

| Risk Namelens Detects      | Consequence if Missed                         |
| -------------------------- | --------------------------------------------- |
| Active competitor on .com  | Market confusion, SEO disadvantage            |
| Registered trademark       | Legal threats, forced rebranding              |
| Negative social sentiment  | Brand damage from the start                   |
| Cultural inappropriateness | Market entry blockers, PR nightmares          |
| Hard-to-spell/pronounce    | Viral marketing failure, word-of-mouth issues |

**What makes Namelens different:**

- **AI-powered intelligence**, not just API queries — We use Grok's web search
  to find what databases miss
- **Direct AI provider connections** — No intermediary SDKs or libraries
  filtering your content. Our `--expert` interface connects directly to AI
  providers via HTTP, giving you full transparency and control over the
  request/response pipeline
- **Dogfooded from day one** — We used Namelens to name itself; it caught a
  critical conflict our codename had
- **CLI-native for developers** — Integrates into your workflow, not a web
  dashboard you need to visit
- **Works with AI agents** — HTTP API enables any agent or automation tool to
  check names directly via REST endpoints

[Read how we named ourselves →](docs/examples/namelens-origin-story.md)

## Development

```bash
make help          # Show all targets
make bootstrap     # Install dependencies
make check         # Run fmt, lint, test
make build         # Build binary
make run           # Run server
make test          # Run tests
make clean         # Clean artifacts
```

Note: The libsql store uses `github.com/tursodatabase/go-libsql`, which requires
`CGO_ENABLED=1` and supports precompiled libraries for darwin/linux amd64/arm64.
For musl-based images, use a glibc-based base image or provide compatible libsql
artifacts. See `docs/operations/builds.md`.

## Architecture

Built on the
[Fulmen workhorse pattern](https://github.com/fulmenhq/crucible/docs/architecture/fulmen-forge-workhorse-standard.md):

```
cmd/namelens/          # CLI entry point
internal/
  cmd/                  # Cobra commands (check, batch, serve, etc.)
  core/                 # Business logic
    checker/            # Availability checkers (domain, npm, github, etc.)
    engine/             # Orchestration, rate limiting, profiles
    store/              # Database (libsql/Turso)
  server/               # HTTP server and Control Plane API
  config/               # Configuration management
config/namelens/       # Default configuration
```

## Important Notice

This tool is provided for informational and exploratory purposes only. Results
do not constitute legal or professional advice. See
[docs/USAGE-NOTICE.md](docs/USAGE-NOTICE.md) for full disclaimer.

## Status

**Version**: See [VERSION](VERSION)

**MVP Goal**: Use this tool to choose its own final name. (We did it - see
[NameLens origin story](docs/examples/namelens-origin-story.md))

## License

Apache License 2.0. See [LICENSE](LICENSE).

## Links

- [3leaps Crucible](https://crucible.3leaps.dev/) - Standards
- [FulmenHQ](https://github.com/fulmenhq) - Ecosystem

---

Built by the [3 Leaps](https://3leaps.net) team
