# Known Issues

This file documents version-specific issues and temporary gaps in NameLens. See [Troubleshooting Guide](troubleshooting.md) for evergreen patterns.

## Version 0.1.3

### RDAP Canonical Endpoint for .app/.dev TLDs

**Status:** ✅ Fixed

**Issue:** Initial implementation used `rdap.nic.google` as the RDAP endpoint for `.app` and `.dev` TLDs, but this domain:
- Has no DNS resolution (NXDOMAIN) in many environments
- Is not the official Google Registry RDAP endpoint
- Returns empty responses in corporate networks with DNS filtering

**Solution:**

Updated RDAP override configuration in `internal/core/checker/domain.go` to use Google's official RDAP API:
- **Primary server:** `https://pubapi.registry.google/rdap` (Google's authoritative endpoint)
- **Fallback server:** `https://www.rdap.net/rdap` (redirect proxy)
- **URL construction:** Fixed to properly append `/domain/<fqdn>` to the base path

This ensures:
1. Queries succeed in most network environments
2. Provenance tracking shows the correct server URL
3. Cache backfill preserves the actual responding server

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

**Issue:** Cached domain results did not populate the `provenance.server` field correctly. The cache stored server information in `extra_data` but it was not being read back properly on cache hits.

**Solution:**

Updated `GetCachedResult()` in `internal/core/store/cache.go` to populate `provenance.Server` from cached `extra_data["resolution_server"]`:

- On cache read, check for `resolution_server` in `extra_data`
- If present, use it to populate `result.Provenance.Server`
- If absent, fall back to constructing URL from first server in override list

This ensures cached domain results show the correct server that originally answered the query, not the primary override server URL.

### Output Sinks (--out, --out-dir)

**Status:** ✅ Implemented

**New flags added:**
- `--out <path>` — Write output to a specific file
- `--out-dir <path>` — Write per-name outputs to a directory
- `--output-format <format>` — Specify output format (replaces deprecated `--output`)
- `--names-file <path>` — Read names from file

**Breaking change:** `--output` flag has been removed. Use `--output-format` instead.

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

Multi-name input is supported (positional names and `--names-file`), and JSON output is a single array of per-name batch results.

What’s still missing:
- Parallel processing with configurable concurrency
- Aggregate summary output (totals across all names)
- `--fail-fast` / partial error handling strategy for large batches
