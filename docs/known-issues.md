# Known Issues

This file documents version-specific issues and temporary gaps in NameLens. See
[Troubleshooting Guide](troubleshooting.md) for evergreen patterns.

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

**Status:** ⚠️ Implemented, sequential execution

Multi-name input is supported (positional names and `--names-file`), and JSON
output is a single array of per-name batch results.

What’s still missing:

- Parallel processing with configurable concurrency
- Aggregate summary output (totals across all names)
- `--fail-fast` / partial error handling strategy for large batches
