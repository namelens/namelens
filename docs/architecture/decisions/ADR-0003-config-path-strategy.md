# ADR-0003: Configuration Path Strategy

**Status:** Accepted **Date:** 2025-12-28 **Deciders:** @3leapsdave

## Context

NameLens needs a consistent strategy for:

1. Where to store user configuration files
2. How to reconcile multiple configuration sources (files, env vars, CLI flags)
3. Supporting both desktop CLI users and server/container deployments

The gofulmen config library provides XDG-compliant path helpers but doesn't
mandate a specific directory structure.

### Current Problem

During QA testing, user config files at
`~/.config/namelens/namelens/config.yaml` were not being loaded because the code
expected `~/.config/namelens/config.yaml`. This mismatch needs resolution.

## Decision

### 1. Config Path Pattern

Use **`<configroot>/<appname>`** pattern:

| Platform | Path                                         |
| -------- | -------------------------------------------- |
| macOS    | `~/.config/namelens/config.yaml`             |
| Linux    | `~/.config/namelens/config.yaml`             |
| Windows  | `%USERPROFILE%\.config\namelens\config.yaml` |

The path is constructed from `appidentity.ConfigName`, making it resilient to
tool renames.

### 2. Config Root Selection

Use gofulmen XDG helpers (`config.GetAppConfigDir`) which respect
`XDG_CONFIG_HOME` and fall back to `~/.config`:

- macOS: `~/.config` (XDG standard)
- Linux: `~/.config` (XDG standard)
- Windows: `%USERPROFILE%\.config` (XDG fallback)

This choice keeps behavior consistent across platforms and aligns with XDG
defaults.

### 3. Configuration Precedence

Sources are merged in order (later wins):

```
Defaults → Config File → Environment Variables → CLI Flags
```

| Source      | Example                                       | Use Case                     |
| ----------- | --------------------------------------------- | ---------------------------- |
| Defaults    | `config/namelens/v0/namelens-defaults.yaml`   | Sensible out-of-box behavior |
| Config File | `~/.config/namelens/config.yaml`              | Desktop user preferences     |
| Env Vars    | `NAMELENS_DOMAIN_WHOIS_FALLBACK_ENABLED=true` | Server/container deployment  |
| CLI Flags   | `--verbose`, `--config=/path`                 | Per-invocation overrides     |

### 4. Environment vs Config File

Both must work equally well:

- **Server mode** (containers, k8s): Env vars only, no config file needed
- **Desktop mode**: Config file primary, env vars for overrides
- **Dev mode**: May use `.env` file (sourced into shell), but this can cause
  confusion

### 5. Config Transparency

Implement `namelens doctor config` command to display:

- Effective merged configuration
- Source of each value (default, file, env, flag)
- Config file path being used (or "none")

This addresses the `.env` pollution problem by making config sources visible.

## Consequences

### Positive

- Clear, documented config location for users
- Path construction uses appidentity, so tool rename only changes app.yaml
- Both env vars and config files work as first-class citizens
- `doctor config` provides transparency for debugging

### Negative

- Existing user configs at legacy nested paths won't be found (migration needed)

### Implementation Required

1. **Fix `internal/config/loader.go:getUserConfigPaths()`** - use app config
   name and XDG paths
2. **Fix `internal/cmd/root.go:initConfig()`** - use gofulmen XDG config dir
3. **Add `doctor config` command** - dump effective config with sources
4. **Update documentation** - correct config paths in user guide

## Alternatives Considered

### A. Use `<configroot>/<vendor>/<appname>`

Provides namespacing but diverges from XDG defaults and complicates
cross-platform expectations. Rejected in favor of standard XDG paths.

### B. Ask gofulmen to mandate vendor prefix

Not appropriate - this should be implementer's choice. gofulmen could add
helpers but shouldn't mandate structure.

## References

- gofulmen config library: `~/dev/fulmenhq/gofulmen/config/`
- XDG Base Directory Specification
- Run journal: `.plans/runjournals/20251228-whois-fallback.md`
