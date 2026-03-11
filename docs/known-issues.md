# Known Issues

This file documents version-specific issues and temporary gaps in NameLens. See
[Troubleshooting Guide](troubleshooting.md) for evergreen patterns.

## Version 0.2.4

### B13: Expert Prompt Loading Fails Outside Git Repos

**Status:** 🔴 Open (High) — BUG-001

**Issue:** `check --expert` fails with "failed to load prompts" when run from
outside a git repository or any directory without `.git`/`go.mod` markers in a
parent. Affects all installed-binary users running from arbitrary directories.

**Root Cause:** `prompt/loader.go:catalogForSchemas()` calls `findRepoRoot()` to
locate JSON schemas on disk. Unlike the config layer (which falls back to
embedded assets via `standaloneAssetsRoot()`), the prompt loader has no fallback
— validation fails immediately when repo root discovery fails.

**Error chain:**
`findRepoRoot()` → `catalogForSchemas()` → `validateConfig()` → `Load()` →
`LoadDefaults()` → `buildPromptRegistry()` → `runExpert()` returns
`"failed to load prompts"`

**Workaround:** Run namelens from within the cloned repository directory.

**Fix approach:** Follow the config layer pattern — embed
`schemas/ailink/v0/prompt.schema.json` and add a temp-directory fallback in
`catalogForSchemas()` when `findRepoRoot()` fails.

**Key files:**

- `internal/ailink/prompt/loader.go` — `catalogForSchemas()`, `findRepoRoot()`
- `internal/config/embedded_assets.go` — pattern to follow
  (`standaloneAssetsRoot()`)
- `internal/config/loader.go:397-408` — `resolveConfigAssetRoot()` fallback
  pattern

---

### B14: First Name in Expert Batch Hits Rate Limit Burst

**Status:** 🟡 Open (Medium) — BUG-002

**Issue:** First name in a multi-name `--expert` batch occasionally gets
"provider request failed" due to rate limit burst. Reproducible with
`--expert-bulk`: first of 5 names (teamvoy) and first of 4 names (pulsevoy)
failed.

**Root Cause:** When `--expert-bulk` produces an incomplete response (schema
validation failure on some names), fallback per-name requests fire immediately
with no backoff. The bulk request consumes rate limit quota, then worker
goroutines (up to `--concurrency=3`) simultaneously send fallback requests
within the same rate limit window. The first name processed hits the provider's
429 response.

**Sequence:**

1. Bulk request completes (1 API call, consumes quota)
2. Worker pool spawns immediately (zero delay)
3. Workers send per-name fallback requests simultaneously
4. Provider returns HTTP 429 on first request(s)
5. No retry/backoff — error propagated as "provider request failed"

**Workaround:** Re-run the check; subsequent attempts usually succeed as the
rate limit window has passed.

**Fix approach:** Add inter-request backoff between bulk completion and fallback
requests, and/or add retry-with-backoff for 429 responses in the ailink search
layer.

**Key files:**

- `internal/cmd/check.go:194-323` — bulk/fallback sequencing
- `internal/ailink/search.go:121-131` — driver call with no retry logic
- `internal/ailink/provider_error_map.go` — 429 error mapping

---

## Version 0.1.4

### xAI Image Generation (Experimental)

**Status:** ⚠️ Experimental

**Issue:** xAI's grok-2-image model is supported but has significant limitations
for brand mark generation:

- **No size control:** The API does not support `size` parameters; output
  dimensions are model-determined (often non-square aspect ratios)
- **No quality/style control:** `quality` and `style` parameters are ignored
- **Inconsistent aspect ratios:** Brand marks typically require 1:1 square
  output for favicon/icon use; grok-2-image produces varied aspect ratios
- **Photorealistic tendency:** Output style leans photorealistic/3D rather than
  flat vector logos suitable for brand marks

Per [xAI documentation](https://docs.x.ai/docs/guides/image-generations), only
`model`, `prompt`, `n`, and `response_format` are currently supported.

**Recommendation:** Use OpenAI GPT Image models (e.g. `gpt-image-1.5`) for
`brand-mark-image` routing until xAI adds dimension and style controls. xAI
image generation may be viable for other use cases where aspect ratio and style
are less constrained.

**Future:** xAI may expose additional parameters through their SDK or future API
updates. Monitor their documentation for `size`, `aspect_ratio`, or `style`
support.

**Configuration:**

```bash
# Recommended: OpenAI GPT Image models for brand marks
NAMELENS_AILINK_ROUTING_BRAND_MARK_IMAGE=namelens-openai-image
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_IMAGE_MODELS_IMAGE=gpt-image-1.5

# Experimental: xAI (non-square output, photorealistic style)
# NAMELENS_AILINK_ROUTING_BRAND_MARK_IMAGE=namelens-xai
```

---

## Version 0.1.3

### RDAP Canonical Endpoint for .app/.dev TLDs

**Status:** ✅ Fixed

**Issue:** Initial implementation used `rdap.nic.google` as the RDAP endpoint
for `.app` and `.dev` TLDs, but this domain:

- Has no DNS resolution (NXDOMAIN) in many environments
- Is not the official Google Registry RDAP endpoint
- Returns empty responses in corporate networks with DNS filtering

**Solution:**

Updated RDAP override configuration in `internal/core/checker/domain.go` to use
Google's official RDAP API:

- **Primary server:** `https://pubapi.registry.google/rdap` (Google's
  authoritative endpoint)
- **Fallback server:** `https://www.rdap.net/rdap` (redirect proxy)
- **URL construction:** Fixed to properly append `/domain/<fqdn>` to the base
  path

This ensures:

1. Queries succeed in most network environments
2. Provenance tracking shows the actual server URL used
3. Cache backfill preserves the actual responding server

**Not a bug:** `www.rdap.net` is a redirect proxy. In some environments you may
see `provenance.server` use `https://www.rdap.net/rdap/...` if the canonical
endpoint is blocked or unavailable.

**Verification:**

```bash
namelens check namelens --tlds=app,dev --output-format=json --no-cache
# Returns taken status for both TLDs with correct provenance.server
# Example output:
{
  "name": "namelens.app",
  "server": "https://pubapi.registry.google/rdap/domain/namelens.app"
}
```

### Cache Backfill for Domain Results

**Status:** ✅ Fixed

**Issue:** Cached domain results did not populate the `provenance.server` field
correctly. The cache stored server information in `extra_data` but it was not
being read back properly on cache hits.

**Solution:**

Updated `GetCachedResult()` in `internal/core/store/cache.go` to populate
`provenance.Server` from cached `extra_data["resolution_server"]`.

**Not a bug:** cache state is reported as `provenance.from_cache` in JSON
output; a top-level `from_cache` field will read as null.

- On cache read, check for `resolution_server` in `extra_data`
- If present, use it to populate `result.Provenance.Server`
- If absent, fall back to constructing URL from first server in override list

This ensures cached domain results show the correct server that originally
answered the query, not the primary override server URL.

### Output Sinks (--out, --out-dir)

**Status:** ✅ Implemented

**New flags added:**

- `--out <path>` — Write output to a specific file
- `--out-dir <path>` — Write per-name outputs to a directory
- `--output-format <format>` — Specify output format (replaces deprecated
  `--output`)
- `--names-file <path>` — Read names from file

**Breaking change:** `--output` flag has been removed. Use `--output-format`
instead.

**Example usage:**

```bash
# Write single name check to file
namelens check myname --output-format=json --out /tmp/result.json

# Write multiple name checks to directory
namelens check myname yourname --out-dir /tmp/reports --output-format=markdown

# Read names from file
namelens check --names-file candidates.txt --output-format=table
```

---

## Future Enhancements

### Multi-Name Checks (Performance)

**Status:** ✅ Concurrent execution (v0.2.0)

Multi-name input is supported (positional names and `--names-file`), with
parallel domain/registry checks via `--concurrency` flag (default: 3).

What's still missing:

- Aggregate summary output (totals across all names)
- `--fail-fast` / partial error handling strategy for large batches
