# Configuration Guide

NameLens uses a three-layer configuration system following the Fulmen Forge
Workhorse Standard.

## Configuration Layers

Configuration is merged in order of precedence (lowest to highest):

| Layer | Source      | Description                                      |
| ----- | ----------- | ------------------------------------------------ |
| 1     | Defaults    | `config/namelens/v0/namelens-defaults.yaml`      |
| 2     | User Config | Platform-specific config file (see paths below)  |
| 3     | `.env` File | Auto-loaded or specified with `--env-file`       |
| 4     | Environment | `NAMELENS_*` environment variables               |
| 5     | CLI Flags   | Command-line flags (`--port`, `--verbose`, etc.) |

Higher layers override lower layers. CLI flags and explicit environment
variables always win.

## Configuration File Paths

NameLens uses XDG paths for user configuration. By default:

| Platform | Path                                            |
| -------- | ----------------------------------------------- |
| macOS    | `~/.config/namelens/config.yaml`                |
| Linux    | `~/.config/namelens/config.yaml`                |
| Windows  | `%USERPROFILE%\\.config\\namelens\\config.yaml` |

You can override the base directory with `XDG_CONFIG_HOME`.

## `.env` File Support

NameLens supports `.env` files for environment variable configuration. This is
convenient for local development and avoids polluting your shell profile.

### Auto-Loading

When running `namelens serve`, `.env` files are loaded automatically in this
order:

1. `$XDG_CONFIG_HOME/namelens/.env` (usually `~/.config/namelens/.env`)
2. `./.env` in the current working directory (overrides XDG values)

### Explicit File

Use `--env-file` to load a specific file (disables auto-loading):

```bash
namelens serve --env-file /path/to/my.env
```

### Getting Started with `.env`

The repository includes a `.env.example` with all available settings:

```bash
# Copy the example and customize
cp .env.example .env

# Or place in your XDG config directory
cp .env.example ~/.config/namelens/.env
```

Edit the file and uncomment the variables you need. The `.env` file is
gitignored and will not be committed.

> **Note**: `.env` files are loaded by `namelens serve` only. For CLI commands
> like `namelens check`, use exported environment variables or the config file.

## Example Configuration

```yaml
# ~/.config/namelens/config.yaml (Linux/macOS default)
# $XDG_CONFIG_HOME/namelens/config.yaml (override)

# Server settings (for HTTP API)
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
  default_provider: namelens-xai # or namelens-openai
  default_timeout: 60s
  cache_ttl: 24h
  providers:
    # xAI provider - recommended for expert search (web intelligence)
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
          api_key: "" # Use NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY env var

    # OpenAI provider - recommended for structured analysis
    namelens-openai:
      enabled: true
      ai_provider: openai
      base_url: https://api.openai.com/v1
      models:
        default: gpt-4o
        reasoning: gpt-5.1
        fast: gpt-4o-mini
      credentials:
        - label: default
          priority: 0
          api_key: "" # Use NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_CREDENTIALS_0_API_KEY env var

    # Anthropic provider - deep analysis and conflict-aware generation
    namelens-anthropic:
      enabled: true
      ai_provider: anthropic
      base_url: https://api.anthropic.com/v1
      models:
        default: claude-sonnet-4-6
      credentials:
        - label: default
          priority: 0
          api_key: "" # Use NAMELENS_AILINK_PROVIDERS_NAMELENS_ANTHROPIC_CREDENTIALS_0_API_KEY env var

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

| Variable                         | Default     | Description                   |
| -------------------------------- | ----------- | ----------------------------- |
| `NAMELENS_HOST`                  | `localhost` | Server bind address           |
| `NAMELENS_PORT`                  | `8080`      | Server port                   |
| `NAMELENS_READ_TIMEOUT`          | `30s`       | HTTP read timeout             |
| `NAMELENS_WRITE_TIMEOUT`         | `30s`       | HTTP write timeout            |
| `NAMELENS_CONTROL_PLANE_API_KEY` |             | API key for `/v1/*` endpoints |

> **Security note**: When no API key is configured, the control plane API allows
> all requests from localhost. Configure a key when exposing the server beyond
> localhost. See [HTTP API Reference](http-api.md) for authentication details.

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

NameLens ships with three drivers: `xai` (for Grok via x.ai), `openai` (for
OpenAI-hosted models), and `anthropic` (for Claude via Anthropic's Messages
API).

Important terminology note:

- `ai_provider: xai` targets x.ai (Grok). The API is "OpenAI-compatible", but
  the model is not an OpenAI model.
- `ai_provider: openai` targets OpenAI (GPT models).

#### Provider Recommendations

| Use Case                      | Recommended Provider | Reason                                             |
| ----------------------------- | -------------------- | -------------------------------------------------- |
| Expert search (`--expert`)    | xAI (Grok)           | Real-time web intelligence via live search         |
| Phonetics (`--phonetics`)     | Any                  | All three produce comparable results               |
| Suitability (`--suitability`) | xAI or Anthropic     | xAI catches more via web; Anthropic reasons deeply |
| Generate (`generate`)         | Anthropic or OpenAI  | Anthropic for depth; OpenAI for speed              |
| Review (`review`)             | Any                  | Works on all providers                             |

**Key differences:** xAI/Grok has built-in web search capabilities for real-time
internet intelligence. Anthropic excels at structured reasoning and conflict
awareness. OpenAI is fastest for structured analysis.

#### OpenAI Provider Setup

```bash
# Enable OpenAI provider
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_ENABLED=true
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_AI_PROVIDER=openai
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_BASE_URL=https://api.openai.com/v1
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_CREDENTIALS_0_API_KEY=sk-your-key

# Model tiers (recommended)
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_MODELS_DEFAULT=gpt-4o
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_MODELS_REASONING=o3
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_MODELS_FAST=gpt-4o-mini

# To use OpenAI as default provider
NAMELENS_AILINK_DEFAULT_PROVIDER=namelens-openai
```

#### Anthropic Provider Setup

```bash
# Enable Anthropic provider
NAMELENS_AILINK_PROVIDERS_NAMELENS_ANTHROPIC_ENABLED=true
NAMELENS_AILINK_PROVIDERS_NAMELENS_ANTHROPIC_AI_PROVIDER=anthropic
NAMELENS_AILINK_PROVIDERS_NAMELENS_ANTHROPIC_BASE_URL=https://api.anthropic.com/v1
NAMELENS_AILINK_PROVIDERS_NAMELENS_ANTHROPIC_CREDENTIALS_0_API_KEY=sk-ant-your-key

# Model configuration
NAMELENS_AILINK_PROVIDERS_NAMELENS_ANTHROPIC_MODELS_DEFAULT=claude-sonnet-4-6

# To use Anthropic as default provider
NAMELENS_AILINK_DEFAULT_PROVIDER=namelens-anthropic
```

#### Model Tiers

NameLens supports model tiers for different workloads:

| Tier        | OpenAI Model | xAI Model               | Used When                     |
| ----------- | ------------ | ----------------------- | ----------------------------- |
| `default`   | gpt-4o       | grok-4-1-fast-reasoning | Most prompts, `--depth=quick` |
| `reasoning` | o3           | -                       | Deep analysis, `--depth=deep` |
| `fast`      | gpt-4o-mini  | -                       | Quick triage, `--depth=fast`  |

**Note:** For fast tier, use `gpt-4o-mini`. Smaller models like `gpt-5-mini` may
fail schema validation on structured prompts.

#### OpenAI Structured Outputs

NameLens automatically uses OpenAI's native `json_schema` structured outputs
with `strict: true` when the prompt defines a response schema. This ensures
reliable JSON output that validates against the schema. If a model doesn't
support `json_schema`, AILink falls back to `json_object` mode.

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
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_MODELS_FAST` (recommend `gpt-4o-mini`;
  some small models may fail schema validation)
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_MODELS_IMAGE`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_SELECTION_POLICY`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_DEFAULT_CREDENTIAL`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_CREDENTIALS_0_API_KEY`
- `NAMELENS_AILINK_PROVIDERS_<INSTANCE>_CREDENTIALS_0_PRIORITY`

Role routing can be set with:

- `NAMELENS_AILINK_ROUTING_<ROLE>=<provider-id>`

For one-off generation runs, you can override routing at invocation time:

```bash
namelens generate "my concept" --provider namelens-anthropic
```

Provider precedence for `generate` is:

1. `--provider` flag (if set)
2. `ailink.routing[<prompt-slug>]`
3. Provider `roles` match
4. `ailink.default_provider`
5. Single enabled provider fallback

Validation behavior for `--provider`:

- unknown id: `unknown provider "<id>" (valid: ...)`
- configured but disabled: `provider "<id>" is disabled`

For `namelens mark`, use separate roles for text vs image generation:

- `brand-mark` (mark directions + image prompts)
- `brand-mark-image` (image generation)

Example:

```bash
NAMELENS_AILINK_ROUTING_BRAND_MARK=namelens-openai
NAMELENS_AILINK_ROUTING_BRAND_MARK_IMAGE=namelens-openai-image
```

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

The **recommended** way to configure an AI provider is the setup wizard:

```bash
namelens setup
```

This interactively selects a provider, securely reads your API key, tests the
connection, and writes the config file. For non-interactive environments:

```bash
namelens setup --provider xai --api-key $XAI_KEY --no-test
```

For manual initialization, use `doctor`:

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
