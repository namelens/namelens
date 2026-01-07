# Release Notes

This file keeps notes for the latest three releases in reverse chronological
order.

## Unreleased

## v0.1.1 (2026-01-07)

Rate limit management and stability improvements.

Highlights:

- **Rate limit admin commands**: New `namelens rate-limit list` and
  `namelens rate-limit reset` commands to inspect and clear persisted rate
  limit state without manual database access
- **SQLite concurrency fix**: Batch checks no longer fail with "database is
  locked" errors thanks to WAL mode and connection serialization
- **CI update**: macOS runner updated from retired macos-13 to macos-15-intel

### Rate Limit Commands

```bash
# List all stored rate limits
namelens rate-limit list

# List endpoints matching a prefix
namelens rate-limit list --prefix=rdap.

# Reset all rate limits (requires confirmation)
namelens rate-limit reset --all --yes

# Reset a specific endpoint
namelens rate-limit reset --endpoint=rdap.verisign.com

# Dry run to see what would be deleted
namelens rate-limit reset --all --dry-run

# JSON output for automation
namelens rate-limit list --output=json
```

## v0.1.0 (2026-01-04)

Initial release of NameLens.

Highlights:

- Domain availability via RDAP bootstrap with cache and rate limits
- Optional WHOIS/DNS fallback for TLDs without RDAP
- Registry and handle checks (npm, PyPI, GitHub)
- Expert analysis via AILink prompt library (xAI/Grok)
- Generate command for AI-powered naming ideation
- Phonetics and suitability analysis prompts
- Diagnostics: `doctor`, `doctor ailink`, and `doctor ailink connectivity`
- Built-in profiles: startup, developer, website, minimal, web3
- Tri-state availability: available/taken/unknown
- Batch command and multiple output formats
