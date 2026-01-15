# Release Notes

This file keeps notes for the latest three releases in reverse chronological
order.

## Unreleased

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

## v0.1.2 (2026-01-11)

Release workflow fix.

Highlights:

- **CI fix**: Add explicit `GITHUB_TOKEN` to all `softprops/action-gh-release`
  steps in `release.yml` for reliable artifact uploads in new GitHub
  organizations

## v0.1.1 (2026-01-07)

Rate limit management and stability improvements.

Highlights:

- **Rate limit admin commands**: New `namelens rate-limit list` and
  `namelens rate-limit reset` commands to inspect and clear persisted rate limit
  state without manual database access
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
