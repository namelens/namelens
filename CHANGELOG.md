# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic
Versioning.

## [Unreleased]

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
