# AILink (internal)

AILink is NameLens' provider-facing AI connector.

## Core Philosophy: Direct HTTP Connections

AILink uses **direct HTTP connections to AI providers**—no intermediary SDKs, no
prompt libraries, no third-party services filtering or modifying content.

### Why This Matters

| Problem with SDKs               | AILink Solution                                          |
| ------------------------------- | -------------------------------------------------------- |
| **Opaque request/response**     | Full visibility into every byte sent/received            |
| **Hidden telemetry**            | No data sent without your explicit configuration         |
| **Filtering or transformation** | Content passes through unchanged                         |
| **Dependency bloat**            | Pure Go `net/http` implementations                       |
| **Version conflicts**           | No SDK version management overhead                       |
| **Unclear behavior**            | Explicit request construction, explicit response parsing |

### Request Flow

```
User Request
    ↓
AILink Configuration (prompt + provider + credentials)
    ↓
Direct HTTP POST to Provider API
    ↓
Provider Response (passed through unchanged)
    ↓
JSON Schema Validation
    ↓
Formatted Output
```

Every step is auditable via debug logging:

```bash
NAMELENS_LOG_LEVEL=debug namelens check acmecorp --expert
```

## Goals

- **Direct-to-provider HTTP drivers** — No SDKs, no intermediaries, pure
  net/http
- **Prompt-driven behavior** — Apps own prompts; AILink enforces prompt rules
- **Multi-provider routing** — Prompt/role → provider instance with credential
  failover
- **Full transparency** — Request/response pipeline fully visible and auditable

## Configuration model

AILink providers are configured as **named instances** under `ailink.providers`.

- The map key is a user-defined id (slug-like), e.g. `namelens-xai`.
- Each instance declares its underlying provider type via `ai_provider`.
- Each instance can hold multiple credentials under `credentials`.
- Credentials are selected via `selection_policy` (default: `priority`) and
  optional `default_credential`.

Example:

```yaml
ailink:
  default_provider: namelens-xai
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
          api_key: "${NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY}"
  routing:
    name-availability: namelens-xai
```

Routing is role-based. In NameLens, the default role is the prompt slug (e.g.
`name-availability`).

## Prompt rules

Prompts are stored in `internal/ailink/prompt/prompts/*.md` and validated at
load time against `schemas/ailink/v0/prompt.schema.json`.

Prompt formatting rules (including fenced JSON examples) are described in
`docs/architecture/decisions/ADR-0004-prompt-standards.md` and enforced via
`make check-prompts`.

## Future direction

AILink is intended to be extracted into a standalone library (e.g.
`fulmenhq/ailink`).

The extracted library will maintain the same direct HTTP philosophy—no SDKs,
full transparency, user-controlled pipeline.

Potential upcoming work:

- Additional drivers (`openai`, `anthropic`, image generation)
- Capability-aware routing and failover
- JSONSchema-first prompt formats (in addition to markdown + frontmatter)
- Stream response support for real-time generation

See
[ADR-0002: AILink Expert Search Architecture](../architecture/decisions/ADR-0002-ailink-architecture.md)
for full design rationale on SDK-free architecture.
