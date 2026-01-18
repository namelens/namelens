# Configuration Guide

NameLens uses a three-layer configuration system following the Fulmen Forge
Workhorse Standard.

## Configuration Layers

Configuration is merged in order of precedence (lowest to highest):

| Layer | Source      | Description                                     |
| ----- | ----------- | ----------------------------------------------- |
| 1     | Defaults    | `config/namelens/v0/namelens-defaults.yaml`     |
| 2     | User Config | Platform-specific config file (see paths below) |
| 3     | Environment | `NAMELENS_*` environment variables              |

Higher layers override lower layers. Environment variables always win.

## Configuration File Paths

NameLens uses XDG paths for user configuration. By default:

| Platform | Path                                            |
| -------- | ----------------------------------------------- |
| macOS    | `~/.config/namelens/config.yaml`                |
| Linux    | `~/.config/namelens/config.yaml`                |
| Windows  | `%USERPROFILE%\\.config\\namelens\\config.yaml` |

You can override the base directory with `XDG_CONFIG_HOME`.

## Example Configuration

```yaml
# ~/.config/namelens/config.yaml (Linux/macOS default)
# $XDG_CONFIG_HOME/namelens/config.yaml (override)

# Server settings (for MCP mode)
server:
  host: localhost
  port: 8080

# Logging
logging:
  level: info # trace, debug, info, warn, error
  profile: SIMPLE # SIMPLE, STRUCTURED, ENTERPRISE

# Domain fallback for TLDs without RDAP
domain:
  whois_fallback:
    enabled: true
    require_explicit: false # false = allow all TLDs, true = only listed TLDs
    tlds: [] # explicit TLD list (empty = all if require_explicit is false)
    timeout: 10s
    cache_ttl: 6h
  dns_fallback:
    enabled: false
    timeout: 5s
    cache_ttl: 30m

# AILink providers
ailink:
  default_provider: namelens-xai
  default_timeout: 60s
  cache_ttl: 24h
  providers:
    namelens-xai:
      enabled: true
      ai_provider: xai
      base_url: https://api.x.ai/v1
      models:
        default: grok-4-1-fast-reasoning
      selection_policy: priority
      credentials:
        - label: default
          priority: 0
          api_key: "" # Better to use NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY env var

# Expert output (routes through AILink)
expert:
  enabled: true
  default_prompt: name-availability
  role: name-availability

# Rate limiting overrides
# Keys must match actual endpoint hostnames (e.g., rdap.verisign.com, whois.whois.nic.io)
# Overrides are per minute. Defaults may use longer windows (e.g., WHOIS is 30/hour).
rate_limits:
  rdap.verisign.com: 60 # override default 30/min for .com/.net RDAP
  whois.whois.nic.io: 1 # override default 30/hour for .io whois

# Cache TTLs
cache:
  available_ttl: 5m
  taken_ttl: 1h
  error_ttl: 30s
```

## Environment Variables

All configuration can be overridden via environment variables with the
`NAMELENS_` prefix.

### Server Configuration

| Variable                 | Default     | Description         |
| ------------------------ | ----------- | ------------------- |
| `NAMELENS_HOST`          | `localhost` | Server bind address |
| `NAMELENS_PORT`          | `8080`      | Server port         |
| `NAMELENS_READ_TIMEOUT`  | `30s`       | HTTP read timeout   |
| `NAMELENS_WRITE_TIMEOUT` | `30s`       | HTTP write timeout  |

### Database Configuration

| Variable                 | Default                               | Description         |
| ------------------------ | ------------------------------------- | ------------------- |
| `NAMELENS_DB_DRIVER`     | `libsql`                              | Database driver     |
| `NAMELENS_DB_PATH`       | `$XDG_DATA_HOME/namelens/namelens.db` | Local database path |
| `NAMELENS_DB_URL`        |                                       | Turso cloud URL     |
| `NAMELENS_DB_AUTH_TOKEN` |                                       | Turso auth token    |

By default, NameLens stores data in `$XDG_DATA_HOME/namelens/namelens.db`
(usually `~/.local/share/namelens/namelens.db`). Set `NAMELENS_DB_URL` to use a
remote libsql/Turso database instead of a local file.

### Domain Fallback Configuration

| Variable                                          | Default | Description                  |
| ------------------------------------------------- | ------- | ---------------------------- |
| `NAMELENS_DOMAIN_WHOIS_FALLBACK_ENABLED`          | `false` | Enable whois fallback        |
| `NAMELENS_DOMAIN_WHOIS_FALLBACK_REQUIRE_EXPLICIT` | `true`  | Require TLD in explicit list |
| `NAMELENS_DOMAIN_WHOIS_FALLBACK_TLDS`             |         | Comma-separated TLD list     |
| `NAMELENS_DOMAIN_WHOIS_FALLBACK_TIMEOUT`          | `10s`   | Whois query timeout          |
| `NAMELENS_DOMAIN_WHOIS_FALLBACK_CACHE_TTL`        | `6h`    | Cache duration               |
| `NAMELENS_DOMAIN_DNS_FALLBACK_ENABLED`            | `false` | Enable DNS fallback          |
| `NAMELENS_DOMAIN_DNS_FALLBACK_TIMEOUT`            | `5s`    | DNS query timeout            |

### AILink Provider Configuration

AILink providers are configured as **named instances** under `ailink.providers`.
The instance id is a slug (e.g. `namelens-xai`, `namelens-openai`).

NameLens ships with an `xai` driver (for Grok via x.ai) and supports an `openai` driver for OpenAI-hosted models.

Important terminology note:

- `ai_provider: xai` targets x.ai (Grok). The API is “OpenAI-compatible”, but the model is not an OpenAI model.
- `ai_provider: openai` targets OpenAI (GPT models).

| Variable                           | Default        | Description               |
| ---------------------------------- | -------------- | ------------------------- |
| `NAMELENS_AILINK_DEFAULT_PROVIDER` | `namelens-xai` | Default provider id       |
| `NAMELENS_AILINK_DEFAULT_TIMEOUT`  | `60s`          | Default request timeout   |
| `NAMELENS_AILINK_CACHE_TTL`        | `24h`          | Result cache TTL          |
| `NAMELENS_AILINK_PROMPTS_DIR`      |                | Optional prompt overrides |

Provider instances can be overridden via env vars using this pattern
(underscores become hyphens in the instance id):

- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_AI_PROVIDER`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_BASE_URL`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_MODELS_DEFAULT`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_MODELS_REASONING`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_MODELS_FAST`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_MODELS_IMAGE`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_SELECTION_POLICY`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_DEFAULT_CREDENTIAL`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_CREDENTIALS_0_API_KEY`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_CREDENTIALS_0_PRIORITY`

Role routing can be set with:

- `NAMELENS_AILINK_ROUTING_<ROLE>=<provider-id>`

### Expert Feature Configuration

NameLens “expert” features are prompt-driven; provider selection is handled by
AILink.

| Variable                         | Default             | Description                        |
| -------------------------------- | ------------------- | ---------------------------------- |
| `NAMELENS_EXPERT_ENABLED`        | `false`             | Enable expert output               |
| `NAMELENS_EXPERT_ROLE`           |                     | Role key used for provider routing |
| `NAMELENS_EXPERT_DEFAULT_PROMPT` | `name-availability` | Default prompt slug                |

### Logging Configuration

| Variable               | Default  | Description     |
| ---------------------- | -------- | --------------- |
| `NAMELENS_LOG_LEVEL`   | `info`   | Log level       |
| `NAMELENS_LOG_PROFILE` | `SIMPLE` | Logging profile |

## Schema Validation

Configuration is validated against a JSON Schema at load time:

```
schemas/namelens/v0/config.schema.json
```

Validation errors are logged but don't prevent startup (to maintain flexibility
during development).

## Viewing Effective Configuration

Use `envinfo` to see the current effective configuration.

Use `doctor ailink` to debug AILink role/prompt routing and credential
selection:

```bash
namelens doctor ailink
namelens doctor ailink name-availability
namelens doctor ailink name-availability --role=name-availability
```

```bash
namelens envinfo
```

This shows:

- Application version and build info
- Runtime environment (Go version, OS, arch)
- Server configuration
- Domain fallback settings
- Expert search configuration

## Config Initialization and Status

Use `doctor` to initialize a config file and inspect system state:

```bash
namelens doctor init
namelens doctor config
```

`doctor init` creates a minimal config with `expert.enabled: true`,
`domain.whois_fallback.enabled: true`, and an `ailink.providers.namelens-xai`
stub. If no API key is set, expert requests are skipped until you provide
`NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY` (or set
`ailink.providers.namelens-xai.credentials[0].api_key` in config).

`doctor config` reports config paths, database status, and whether the RDAP
bootstrap cache is initialized.

## Troubleshooting

### Config file not found

Check the XDG path. By default, config lives at `~/.config/namelens/config.yaml`
unless you set `XDG_CONFIG_HOME`.

### Environment variable not working

1. Ensure the variable name is correct (case-sensitive)
2. Check the variable is exported: `export NAMELENS_...`
3. Use `namelens envinfo` to verify the setting

### Whois fallback not triggering

1. Set `NAMELENS_DOMAIN_WHOIS_FALLBACK_ENABLED=true`
2. Set `NAMELENS_DOMAIN_WHOIS_FALLBACK_REQUIRE_EXPLICIT=false` (or add TLDs to
   list)
3. Rebuild if using local development: `make build`

### RDAP results missing

RDAP lookups require the bootstrap cache. If `doctor config` reports the
bootstrap cache is empty, run:

```bash
namelens bootstrap update
```
