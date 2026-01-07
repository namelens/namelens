# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic
Versioning.

## [Unreleased]

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
