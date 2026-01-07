# NameLens

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**See clearly. Name boldly.**

## Overview

Namelens is a fast, intelligent CLI tool and server that helps developers,
founders, and teams choose perfect project names—without the heartbreak of
downstream conflicts. It checks availability across key domains (.com, .io,
.dev, and more via RDAP with WHOIS fallback), package registries (npm, PyPI), and social handles
(GitHub)—then goes deeper with optional Grok-powered expert analysis for
trademarks, brand associations, sentiment, and hidden risks. From quick
availability scans to full brand proposals and launch plans, Namelens combines
precise technical checks with real-time internet intelligence to give you
confidence that your name is truly ready for prime time. Built for the moments
when “good enough” isn’t—because rebranding later hurts.

- **Domains** - via RDAP (Registration Data Access Protocol) with WHOIS fallback
- **Package registries** - npm, PyPI
- **Social handles** - GitHub

## Quick Start

```bash
# Install dependencies
make bootstrap

# Build
make build

# Check a name
./bin/namelens check acmecorp --profile=startup

# Start server (for MCP mode)
./bin/namelens serve
```

## CLI Commands

```bash
# Check a single name
namelens check <name>
namelens check <name> --profile=startup
namelens check <name> --tlds=com,io,dev
namelens check <name> --expert

# Batch check from file
namelens batch names.txt
namelens batch names.txt --output=json

# Profile management
namelens profile list
namelens profile show startup

# Expert prompts
namelens ailink list

# Server mode (for MCP)
namelens serve
namelens serve --port 9000

# Diagnostics
namelens version
namelens health
namelens doctor
namelens doctor ailink name-availability
namelens doctor ailink connectivity name-availability
namelens doctor ailink connectivity name-availability --output=json
```

## Profiles

Profiles define what to check for a given name:

| Profile   | Domains         | Registries | Handles |
| --------- | --------------- | ---------- | ------- |
| `startup` | .com, .io, .dev | npm, pypi  | github  |
| `minimal` | .com            | -          | -       |
| `web3`    | .xyz, .io, .gg  | npm        | github  |

## Configuration

NameLens uses environment variables with the `NAMELENS_` prefix:

```bash
NAMELENS_PORT=8080
NAMELENS_LOG_LEVEL=info
NAMELENS_DB_PATH=$XDG_DATA_HOME/namelens/namelens.db
# For Turso:
# NAMELENS_DB_URL=libsql://your-db.turso.io
# NAMELENS_DB_AUTH_TOKEN=your-auth-token

# AILink providers (optional)
# NAMELENS_AILINK_DEFAULT_PROVIDER=namelens-xai
# NAMELENS_AILINK_DEFAULT_TIMEOUT=60s
# NAMELENS_AILINK_CACHE_TTL=24h
# NAMELENS_AILINK_PROMPTS_DIR=/path/to/prompts
#
# Provider instance: namelens-xai
# NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_ENABLED=true
# NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_AI_PROVIDER=xai
# NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_BASE_URL=https://api.x.ai/v1
# NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_MODELS_DEFAULT=grok-4-1-fast-reasoning
# NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_LABEL=default
# NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=your-api-key
#
# Expert feature wiring (routes through AILink)
# NAMELENS_EXPERT_ENABLED=true
# NAMELENS_EXPERT_ROLE=name-availability
# NAMELENS_EXPERT_DEFAULT_PROMPT=name-availability
```

Copy `.env.example` to `.env` for local development.

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
  server/               # HTTP server for MCP mode
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
