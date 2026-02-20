# Release Notes

This file keeps notes for the latest three releases in reverse chronological
order.

## v0.2.2 (2026-02-20)

Model refresh: updated Anthropic and OpenAI model tiers to current releases.

Highlights:

- **Anthropic models updated**: Default and reasoning tiers now use
  `claude-sonnet-4-6`; fast tier updated to `claude-haiku-4-5-20251001`
- **OpenAI reasoning tier added**: `o3` configured as the `reasoning` model
  for `--depth=deep` workloads — OpenAI's dedicated reasoning model delivers
  significantly higher quality for deep brand analysis

### Model Updates

| Provider  | Tier        | Before                     | After                    |
| --------- | ----------- | -------------------------- | ------------------------ |
| Anthropic | `default`   | claude-sonnet-4-5-20250929 | claude-sonnet-4-6        |
| Anthropic | `reasoning` | claude-sonnet-4-5-20250929 | claude-sonnet-4-6        |
| Anthropic | `fast`      | claude-3-5-haiku-20241022  | claude-haiku-4-5-20251001 |
| OpenAI    | `reasoning` | (not set)                  | o3                       |

To update an existing config, re-run the setup wizard or edit
`~/.config/namelens/config.yaml` directly.

### Upgrade Notes

No breaking changes. Existing configs with the old model names remain valid —
the models are still available from Anthropic. Update at your convenience.

---

## v0.2.1 (2026-02-14)

Agent-ready deployment: headless server API, guided setup, and safety
guardrails.

Highlights:

- **Setup wizard**: 30 seconds from install to full AI capability with
  `namelens setup`
- **Daemon mode**: Run server in background with `--daemon` flag
- **Control Plane API**: REST API for remote name checking and comparison
- **Anthropic Claude**: Third AI provider option alongside xAI and OpenAI
- **Expert mode guidance**: Safety warnings prevent false confidence from
  availability-only results
- **Environment files**: Auto-load `.env` from XDG config or specify with
  `--env-file`

### Setup Wizard

Get from zero to expert analysis in 30 seconds:

```bash
# Interactive — choose provider, paste key, verify connection
namelens setup

# Non-interactive (CI/automation)
namelens setup --provider xai --api-key $XAI_KEY --no-test

# Use a custom config path
namelens setup --config ~/my-project/namelens.yaml
```

The wizard walks through provider selection (xAI Grok, OpenAI GPT, or
Anthropic Claude), securely reads your API key (no-echo), runs a layered
connection test (DNS → TCP → TLS → HTTP auth), and writes the config. Existing
settings are preserved — updating a provider key won't clobber custom model
tiers or other providers you've already configured.

### Anthropic Claude

Anthropic is now a first-class AILink provider alongside xAI and OpenAI. The
default model is `claude-sonnet-4-5-20250929`. Configure via setup wizard or
manually in config:

```bash
namelens setup --provider anthropic --api-key $ANTHROPIC_KEY
```

### Expert Mode Guidance

When no AI backend is configured, `check` and `compare` commands now display a
warning banner:

```
Note: Running in limited analysis mode (no AI backend configured).

  Domain and registry checks show availability only, not commercial safety.
  Names may have trademark conflicts, active use, or brand confusion risks
  not detected by basic availability checks.

  To enable comprehensive analysis, run the setup wizard:
    namelens setup
```

When AI *is* configured but `--expert` wasn't used, a lighter tip appears after
results:

```
Tip: These results show availability only. For trademark, commercial use,
     and brand safety analysis, run with --expert flag.
```

This addresses the "domain available = name safe" footgun — availability checks
alone can miss trademark conflicts, active competitors, and brand confusion
risks that only surface through AI-powered web search.

### Daemon Mode

Run the server as a background process:

```bash
# Start server in background
namelens serve --daemon --port 8080

# Check server status
namelens serve status --port 8080

# Stop server gracefully
namelens serve stop --port 8080

# Force stop if needed
namelens serve stop --port 8080 --force

# Clean up orphaned processes
namelens serve cleanup --port 8080
```

PID files are stored in `~/.local/share/namelens/run/` following XDG
conventions.

### Control Plane HTTP API

Remote access to NameLens via REST API:

```bash
# Generate an API key
namelens serve --generate-key
# Output: nlcp_a1b2c3d4e5f6...

# Start server with authentication
export NAMELENS_CONTROL_PLANE_API_KEY=nlcp_a1b2c3d4e5f6...
namelens serve --daemon

# Check name availability
curl -X POST http://localhost:8080/v1/check \
  -H "Content-Type: application/json" \
  -d '{"name": "myproject", "tlds": ["com", "io"]}'

# Compare candidates
curl -X POST http://localhost:8080/v1/compare \
  -H "Content-Type: application/json" \
  -d '{"names": ["alpha", "beta", "gamma"]}'
```

API endpoints:

- `GET /health` - Health check (no auth required)
- `GET /v1/status` - Provider and rate limit status
- `POST /v1/check` - Check name availability
- `POST /v1/compare` - Compare multiple candidates
- `GET /v1/profiles` - List available profiles

Authentication:

- Localhost requests without an API key header bypass authentication
- Localhost requests that include an API key header are validated normally
- Remote requests require `X-API-Key` header
- Keys are generated with `namelens serve --generate-key`

### Environment Files

The server automatically loads environment variables from `.env` files:

```bash
# Auto-loading (in order of precedence):
# 1. $XDG_CONFIG_HOME/namelens/.env (~/.config/namelens/.env)
# 2. ./.env (current directory)

# Or specify explicitly:
namelens serve --env-file /path/to/custom.env
```

Example `.env` for server:

```bash
NAMELENS_CONTROL_PLANE_API_KEY=nlcp_your_key_here
NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=xai-xxx
```

### Configuration

New flags for `namelens serve`:

- `--daemon` / `-d`: Run in background
- `--env-file` / `-e`: Load environment from file

New subcommands:

- `namelens serve stop [--port] [--force]`
- `namelens serve status [--port] [--all]`
- `namelens serve cleanup [--port] [--force]`

### Bug Fixes

- Health endpoint returns proper schema-conformant `status` field (`pass`,
  `fail`, `warn`)
- API error codes are consistently lowercase snake_case across all endpoints
- Invalid profile names in `/v1/check` now return 400 instead of silently
  falling back
- Localhost requests with an incorrect API key are properly rejected when a key
  is configured
- Setup wizard respects `--config` global flag for custom config paths
- Config merge preserves custom model tiers when updating provider keys
- Piped input handling in setup wizard works correctly with buffered readers
- AILink trace file is properly flushed on exit
- Anthropic expert check no longer fails with "provider request failed" when
  model returns tool-call hallucinations outside the expected schema
- Anthropic generate no longer truncates candidate lists
- Expert mode warning only displays when no AI backend is genuinely configured
  (previously fired even with a working provider)
- xAI image responses now preserve original MIME type correctly
- Anthropic provider capabilities reporting corrected
- Server `serve stop` and API auth hardened for edge cases

### Dependencies

- gofulmen v0.3.0 → v0.3.3 (fixes `namelens version --extended` reporting)
- crucible v0.4.2 → v0.4.9
- sysprims v0.1.10 → v0.1.11 (dynamic library compatibility)
- golang.org/x/term v0.39.0 (secure terminal input for setup wizard)

### Upgrade Notes

- PID files moved from `~/.namelens/` to `~/.local/share/namelens/run/`
- Old `~/.namelens/` directory can be safely removed

## v0.2.0 (2026-02-01)

End-to-end naming workflow: from idea generation through brand mark
visualization.

Highlights:

- **Brand mark generation**: New `namelens mark` generates logo concepts and
  images for finalist names with color modes and context support
- **Bulk expert analysis**: Check up to 10 names with a single AI call—90%
  reduction in API costs for shortlist triage
- **Concurrent checks**: Parallel domain/registry checks with `--concurrency`
  flag for 3-5x faster batch operations
- **Context extraction**: New `namelens context` prepares project corpus for AI
  workflows
- **Thumbnail utility**: New `namelens image thumb` creates agent-friendly
  thumbnails
- **AILink tracing**: Debug provider issues with `--trace` flag
- **Security updates**: Critical CVE fixes in x/crypto, x/net dependencies

### Recommended Workflow

```
generate → compare → review → mark → thumb
```

1. **Generate** candidates: `namelens generate "my product concept"`
2. **Compare** finalists: `namelens compare name1 name2 name3`
3. **Review** top picks: `namelens review winner --depth=deep`
4. **Mark** chosen name: `namelens mark winner --out-dir ./marks --color brand`
5. **Thumb** for sharing: `namelens image thumb --in-dir ./marks`

### Brand Mark Generation

Generate early-stage logo directions for finalist names:

```bash
namelens mark "myproject" --out-dir ./marks \
  --description "Static analysis tool for Go" \
  --audience "developers" \
  --color brand
```

Color modes: `monochrome` (B&W), `brand` (tech palette), `vibrant` (bold colors)

### Bulk Expert Analysis

Screen multiple candidates efficiently before running expensive deep analysis:

```bash
# Check 5-10 names with single AI call (default limit: 10)
namelens check envoyrelay conduitx nexusiq wardex sievex \
  --expert --expert-bulk --expert-depth=quick

# Custom batch size
namelens check name1 name2 name3 name4 name5 name6 name7 \
  --expert --expert-bulk --expert-bulk-limit 7
```

Output shows per-name availability, risk levels, and insights from a single
provider call instead of N separate calls.

**Cost savings**: 10 names × 1 call vs 10 names × 10 calls = 90% reduction  
**Time savings**: Parallel domain checks + single AI call = 60-80% faster

### Concurrent Checks

Speed up batch operations with parallel domain checks:

```bash
# Check 10 names with 5 concurrent workers (default: 3)
namelens check name1 name2 ... name10 \
  --tlds com,org,net --concurrency 5

# Recommended for RDAP-heavy TLDs (.com, .org, .net)
# Avoid high concurrency with WHOIS-heavy TLDs (.io, .sh, .co)
```

Each name's domains are checked in parallel up to the concurrency limit,
reducing total wall-clock time for large batches.

### Context Command

Prepare corpus from project directories for AI generation:

```bash
# Generate JSON corpus from planning docs
namelens context ./planning --output=json > corpus.json

# Pipe directly to generate command
namelens context ./planning | namelens generate "my product" --corpus=-

# Human-readable markdown for review
namelens context ./docs --output=markdown > context.md
```

The context command:

- Classifies files by type (readme, architecture, decisions, planning)
- Allocates content budget by priority
- Extracts metadata from project files
- Optimizes for AI prompt inclusion

### AILink Tracing

Debug provider issues with full request/response capture:

```bash
# Trace all AILink interactions
namelens check myname --expert --trace /tmp/debug.ndjson

# Analyze trace data
jq 'select(.error)' /tmp/debug.ndjson  # Find errors
jq '.duration_ms' /tmp/debug.ndjson | jq -s 'add/length'  # Avg latency
```

Trace files include:

- Full request body (prompts, tools, parameters)
- Complete responses (content, tool calls, errors)
- Timing data (request start, completion, duration)
- Token usage and cost information

### Thumbnail Generation

Create agent-friendly thumbnails for sharing:

```bash
namelens image thumb --in-dir ./marks --max-size 256 --format jpeg
```

### Configuration Updates

No breaking changes. All v0.1.3 configurations remain valid.

New optional config for brand mark image routing:

```bash
NAMELENS_AILINK_ROUTING_BRAND_MARK_IMAGE=namelens-openai-image
```

New optional flags:

- `--expert-bulk`: Enable bulk expert mode
- `--expert-bulk-limit`: Control batch size (default: 10)
- `--concurrency`: Parallel workers for domain checks (default: 3)
- `--trace`: Capture AILink interactions to NDJSON file

### Documentation

- [Expert Search Guide](docs/user-guide/expert-search.md) - Updated for bulk
  mode
- [Context Command Guide](docs/user-guide/context.md) - New corpus extraction
  docs
- [Batch Operations](docs/user-guide/batch.md) - Concurrent check examples

### Security Updates

| Package  | Update            | Advisory            | Severity |
| -------- | ----------------- | ------------------- | -------- |
| x/crypto | v0.9.0 → v0.47.0  | GHSA-v778-237x-gjrc | Critical |
| x/net    | v0.10.0 → v0.48.0 | GHSA-4374-p667-p6c8 | High     |
| testify  | v1.4.0 → v1.11.1  | removes yaml.v2     | High     |

### Upgrade Notes

No breaking changes. All v0.1.3 configurations, caches, and workflows remain
valid.

### Performance Improvements

| Operation               | v0.1.3 | v0.2.0   | Improvement |
| ----------------------- | ------ | -------- | ----------- |
| 10 names + expert       | ~300s  | ~60s     | 5x faster   |
| AI API calls (10 names) | 10     | 1        | 10x fewer   |
| Domain check throughput | serial | parallel | 3-5x faster |
