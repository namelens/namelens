# AILink (internal)

AILink is NameLens’ provider-facing AI connector.

Goals:

- Direct-to-provider HTTP drivers (no intermediary prompt libraries)
- Prompt-driven behavior (apps own prompts; AILink enforces prompt rules)
- Multi-provider, multi-credential routing (prompt/role → provider instance)

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

Potential upcoming work:

- Additional drivers (`openai`, `anthropic`, image generation)
- Capability-aware routing and failover
- JSONSchema-first prompt formats (in addition to markdown + frontmatter)
