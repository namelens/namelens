# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic
Versioning.

## [0.2.1] - 2026-02-06

### Added

- **Setup wizard** (`namelens setup`) for interactive AI backend configuration
  - Provider selection: xAI (Grok), OpenAI (GPT), Anthropic (Claude)
  - Secure no-echo API key input
  - Layered connection testing (DNS/TCP/TLS/HTTP auth)
  - Config merge preserves existing providers and custom model tiers
  - Non-interactive mode with `--provider`, `--api-key`, `--no-test` flags
  - Respects `--config` global flag for custom config paths
- **Server daemon mode** for background operation
  - `namelens serve --daemon` starts server in background
  - `namelens serve stop` gracefully stops running server
  - `namelens serve status` checks server status and uptime
  - `namelens serve cleanup` kills orphaned processes on port
  - PID files stored in XDG data directory (`~/.local/share/namelens/run/`)
- **Environment file support** for server configuration
  - `--env-file` flag to load custom .env file
  - Auto-loads `$XDG_CONFIG_HOME/namelens/.env` if present
  - Auto-loads `./.env` from current directory (overrides XDG)
- **Control Plane HTTP API** for remote access
  - REST API at `/v1/*` for name availability checking
  - `POST /v1/check` - Check name availability
  - `POST /v1/compare` - Compare multiple candidates
  - `GET /v1/status` - Provider and rate limit status
  - `GET /v1/profiles` - List available profiles
  - API key authentication for non-localhost requests
  - `namelens serve --generate-key` creates secure API keys
- **Anthropic Claude driver** for AILink provider system
  - Direct HTTP integration with Anthropic Messages API
  - Supports Claude Sonnet 4.5 as default model
  - Enables deep analysis and conflict-aware name generation

### Fixed

- Health endpoint response now returns proper HealthCheck objects with
  `status` field (`pass`, `fail`, `warn`) matching the OpenAPI schema
- API error responses now use consistent lowercase snake_case error codes
  (`bad_request`, `not_found`, etc.) across all endpoints
- Invalid profile names in `/v1/check` now return 400 Bad Request instead
  of silently falling back to the minimal profile
- Localhost requests with an incorrect API key are now properly rejected
  when a key is configured (previously bypassed validation)
- AILink trace file is now properly flushed on exit
- Setup wizard respects `--config` global flag for custom config paths
- Setup config merge preserves custom model tiers when updating provider keys
- Piped input to setup wizard no longer loses data between prompts

### Changed

- Build system now uses `-tags sysprims_shared` for CGO compatibility
- PID files moved from `~/.namelens/` to XDG-compliant
  `~/.local/share/namelens/run/`

### Dependencies

- Updated `github.com/fulmenhq/gofulmen` v0.3.0 → v0.3.3 (fixes version
  reporting in `--extended` output)
- Updated `github.com/fulmenhq/crucible` v0.4.2 → v0.4.9
- Updated `github.com/3leaps/sysprims` v0.1.10 → v0.1.11 (dynamic library fix)
- Added `github.com/joho/godotenv` v1.5.1 for .env file loading
- Added `golang.org/x/term` v0.39.0 for secure terminal input
- Go directive updated 1.25.1 → 1.25.5

## [0.2.0] - 2026-02-01

### Added

- **Brand mark generation** for visual identity exploration
  - `namelens mark` command generates logo/mark directions and images
  - Routes text generation (mark directions) and image generation separately
  - Supports `--color` flag with modes: `monochrome`, `brand`, `vibrant`
  - Supports `--description` and `--audience` for better context
  - Supports `--background transparent` for compositing-ready output
  - Outputs WebP, PNG, or JPEG in configurable sizes
- **Thumbnail utility** for agent-friendly image sharing
  - `namelens image thumb` produces JPEG/PNG thumbnails from mark images
  - Supports WebP, PNG, and JPEG input formats
  - Configurable max size (64-1024px) and output format
- **Bulk expert analysis** for efficient multi-name screening
  - `namelens check <names> --expert --expert-bulk` analyzes up to 10 names in a
    single AI call
  - `--expert-bulk-limit` flag to control batch size (default: 10)
  - Significantly reduces API costs and latency for shortlist triage
- **Concurrent domain checks** for faster batch operations
  - `--concurrency` flag enables parallel domain/registry checks (default: 3)
  - Reduces total check time for multiple names
  - Configurable concurrency level based on rate limit tolerance
- **Context command** for corpus generation and prompt preparation
  - `namelens context <dir>` scans directories and generates structured corpus
  - Supports JSON, markdown, and prompt output formats
  - File classification by type (readme, architecture, decisions, planning)
  - Budget-aware content allocation for AI prompt optimization
  - Pipeline-friendly: pipe output directly to `namelens generate --corpus=-`
- **AILink request/response tracing** for debugging and transparency
  - `--trace <file>` flag captures full provider interactions to NDJSON
  - Includes request body, response, timing, and token usage
  - Essential for debugging provider issues and auditing AI calls
- **AI backend status check** in doctor command
  - `namelens doctor` now validates AI provider configuration
  - Reports provider health, model availability, and credential status
  - Clear guidance when AI backend is not configured
- **Expert mode guidance** in check and compare commands
  - Helpful hints when running without `--expert` flag
  - Explains benefits of comprehensive analysis vs basic availability
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

- Domain prompts updated with improved context extraction support
- xAI driver updated to use Responses API with enhanced tracing support
- AILink now supports image-only provider configurations
  - Providers no longer require `models.default` for image-only routing
- Updated `.env.example` with GPT Image and image provider routing examples

### Fixed

- Effective break statement in concurrent check enqueue loop (context
  cancellation now properly stops all workers)
- Schema validation error messages now include `--trace` hint for debugging

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
