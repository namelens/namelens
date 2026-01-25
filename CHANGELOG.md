# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic
Versioning.

## [Unreleased]

## [0.1.4] - 2026-01-24

### Security

- Updated golang.org/x/crypto v0.9.0 → v0.47.0 (addresses GHSA-v778-237x-gjrc
  critical, GHSA-hcg3-q754-cr77 high)
- Updated golang.org/x/net v0.10.0 → v0.48.0 (addresses GHSA-4374-p667-p6c8
  high)
- Updated golang.org/x/sys v0.36.0 → v0.40.0
- Updated github.com/stretchr/testify v1.4.0 → v1.11.1 (removes yaml.v2
  transitive)
- Updated github.com/alecthomas/units to latest (removes yaml.v2 transitive)
- Added SDR-001 documenting scanner false positives from openrdap's declared
  requirements

### Added

- `namelens mark` command for generating brand mark/logo directions and images
  - Routes text generation (mark directions) and image generation separately
  - Supports `--color` flag with modes: `monochrome`, `brand`, `vibrant`
  - Supports `--description` and `--audience` for better context
  - Supports `--background transparent` for compositing-ready output
  - Outputs WebP, PNG, or JPEG in configurable sizes
- `namelens image thumb` command for generating thumbnails
  - Produces agent-friendly JPEG/PNG thumbnails from mark images
  - Supports WebP, PNG, and JPEG input formats
  - Configurable max size (64-1024px) and output format
- OpenAI GPT Image model support for image generation
  - Recommended backend for brand mark generation
  - Better instruction following and cleaner output than legacy models
- xAI image generation support (experimental)
  - grok-2-image model available but limited (no size/aspect control)
  - Documented as experimental in known-issues.md
- Separate image provider routing via `brand-mark-image` role
  - Allows different providers for text vs image generation
- Brand mark prompt with color mode support and text-free enforcement

### Changed

- AILink now supports image-only provider configurations
  - Providers no longer require `models.default` for image-only routing
- Updated `.env.example` with GPT Image and image provider routing examples

## [0.1.3] - 2026-01-14

### Added

- `namelens compare` command for side-by-side candidate screening with
  availability, risk, phonetics, and suitability scores
- Cargo/crates.io registry checker for Rust crate availability
- `--mode=quick` flag for `review` command to enable fast screening workflows
- Standardized output sinks with `--out` and `--out-dir` flags across commands
- Multi-name input support via `--names-file` flag for check command
- Compare command documentation in user guide

### Fixed

- Domain checker now routes .app and .dev TLDs via Google RDAP for reliable
  availability checks
- SQLite store tolerates lock during WAL mode setup, improving initialization
  reliability
- Review command now honors `--include-raw` flag on failure paths

### Changed

- Developer profile now includes cargo registry alongside npm and pypi
- Updated documentation to reflect crates.io support across all registry
  references

## [0.1.2] - 2026-01-11

### Fixed

- Release workflow: Add explicit GITHUB_TOKEN to all artifact upload steps for
  reliable authentication in new GitHub organizations

## [0.1.1] - 2026-01-07

### Added

- `namelens rate-limit list` command to inspect stored rate limit state
- `namelens rate-limit reset` command to clear persisted rate limits
- Rate limit filtering by endpoint (exact match) or prefix
- Dry-run mode (`--dry-run`) for rate limit reset operations
- JSON output format for rate limit commands (`--output=json`)

### Fixed

- SQLite "database is locked" errors during batch checks by enabling WAL mode,
  busy timeout, and single-connection serialization for local libsql stores

### Changed

- CI: Updated macOS runner from deprecated macos-13 to macos-15-intel

## [0.1.0] - 2026-01-04

### Added

- Domain availability checks via RDAP bootstrap with caching and rate limits
- Optional WHOIS/DNS fallback for TLDs without RDAP
- Registry checks for npm, PyPI, and GitHub
- Expert analysis via AILink prompt library (xAI/Grok driver)
- AILink provider instances with multi-credential selection and role routing
- Generate command for AI-powered naming ideation
- Phonetics and suitability analysis prompts
- Built-in profiles: startup, developer, website, minimal, web3
- Tri-state availability reporting (available/taken/unknown)
- Batch checks with table/json/markdown output formats
- CLI diagnostics: envinfo, doctor, and AILink connectivity checks
