# Release Notes

This file keeps notes for the latest three releases in reverse chronological
order.

## Unreleased

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

## v0.1.3 (2026-01-14)

Candidate comparison and Rust ecosystem support.

Highlights:

- **Compare command**: New `namelens compare` for side-by-side candidate
  screening with availability, risk, phonetics, and suitability scores
- **Cargo/crates.io checker**: Rust crate availability now included in registry
  checks
- **Review quick mode**: New `--mode=quick` flag for fast screening workflows
- **Improved reliability**: .app/.dev domains now route via Google RDAP

### Compare Command

Screen 2-10 candidates before running expensive brand analysis:

```bash
# Quick availability screening
namelens compare fulgate agentanvil toolcrux --mode=quick

# Full analysis with phonetics and suitability
namelens compare fulgate toolcrux

# Export for stakeholders
namelens compare alpha beta gamma --output-format=markdown --out comparison.md
```

Output:

```
╭──────────┬──────────────┬──────┬───────────┬─────────────┬────────╮
│ NAME     │ AVAILABILITY │ RISK │ PHONETICS │ SUITABILITY │ LENGTH │
├──────────┼──────────────┼──────┼───────────┼─────────────┼────────┤
│ fulgate  │ 7/7          │ low  │ 83        │ 95          │      7 │
│ toolcrux │ 7/7          │ low  │ 81        │ 95          │      8 │
╰──────────┴──────────────┴──────┴───────────┴─────────────┴────────╯
```

### Cargo/crates.io Support

Check Rust crate availability:

```bash
# Check specific registry
namelens check mycrate --registries=cargo

# Developer profile now includes cargo
namelens check myproject --profile=developer
```

### Review Quick Mode

Fast screening without full brand analysis:

```bash
namelens review myproject --mode=quick
```
