# ADR-0002: AILink Expert Search Architecture

**Status**: Accepted  
**Date**: 2024-12-27  
**Deciders**: @3leapsdave

## Context

NameLens needs AI-powered expert analysis to supplement deterministic
availability checks. This capability should:

1. Provide real-time research on name conflicts, reputation risks, and social
   mentions
2. Support multiple AI providers (Grok, Claude, future models)
3. Be structured for eventual extraction to gofulmen as a reusable module
4. Allow user-customizable prompts

Key decisions required:

1. SDK vs direct HTTP integration
2. Multi-provider abstraction strategy
3. Prompt configuration and registry pattern
4. Schema lifecycle and validation

## Decision

### 1. Direct AI backends only (no SDKs)

**Decision**: All AI provider integrations use direct HTTP calls, not vendor
SDKs.

**Rationale**:

- **Auditability**: Direct HTTP is transparent; request/response fully visible
- **Portability**: No SDK dependency chains or version conflicts
- **Control**: Fine-grained timeout, retry, and error handling
- **Extraction**: Easier to move to gofulmen without SDK baggage
- **Security**: No hidden telemetry or behavior from SDKs

**Implementation**:

- Each driver implements the `Driver` interface via `net/http`
- Request/response types are explicit Go structs
- No `go-openai`, `anthropic-go`, or similar libraries

**Trade-offs accepted**:

- More boilerplate for each provider
- Must track API changes manually
- No SDK conveniences (auto-retry, token counting helpers)

This policy applies to all current and future drivers.

### 2. Driver interface for multi-provider support

**Decision**: Define a provider-agnostic `Driver` interface that each backend
implements.

```go
type Driver interface {
    Complete(ctx context.Context, req *Request) (*Response, error)
    Name() string
    Capabilities() Capabilities
}
```

**Rationale**:

- Clean separation between AILink service logic and provider specifics
- Easy to add new providers (Anthropic, OpenAI planned for future releases)
- Testable with mock drivers

**Drivers**:

| Driver      | API                      | Status      |
| ----------- | ------------------------ | ----------- |
| `xai`       | OpenAI-compatible (Grok) | Implemented |
| `openai`    | OpenAI API               | Planned     |
| `anthropic` | Anthropic Messages API   | Planned     |

### 3. Prompt registry with slug-based lookup

**Decision**: Prompts are YAML files with frontmatter, identified by slug,
loaded from embedded defaults + optional user directory.

**Loading priority**:

1. Embedded defaults (`//go:embed prompts/*.yaml`)
2. User prompts_dir (overrides by slug match)

**Slug validation**: DNS-label-like (`^[a-z0-9][a-z0-9-]*[a-z0-9]$`, 1-64 chars)

**Rationale**:

- YAML is readable for multi-line system prompts
- Frontmatter separates metadata from prompt content
- Slugs provide stable identifiers for caching and CLI
- User override enables customization without forking

### 4. Schema-first validation

**Decision**: Use JSONSchema for prompt configs and response validation,
leveraging gofulmen/schema.

**Schema locations** (anticipating crucible SSOT):

```
schemas/ailink/v0/
├── prompt.schema.json
└── search-response.schema.json
```

**Lifecycle**:

1. crucible (fulmenhq/crucible) is SSOT for schemas
2. gofulmen imports and implements ailink package
3. Apps consume via gofulmen

For v0.1.0, namelens has local schemas. These will be upstreamed to crucible.

### 5. Content types for multimodal support

**Decision**: Define content blocks with IANA media types, anticipating image
support.

```go
type ContentBlock struct {
    Type    ContentType `json:"type"`     // "text/plain", "image/png", etc.
    Text    string      `json:"text,omitempty"`
    Data    []byte      `json:"data,omitempty"`
    DataURL string      `json:"data_url,omitempty"`
}
```

**Rationale**:

- IANA types are standard and extensible
- Structured blocks (vs raw strings) support mixed content
- Image support deferred to v0.2.0 but schema-ready now

### 6. Extraction-ready architecture

**Decision**: Structure the ailink package for future extraction to gofulmen.

**What moves to gofulmen**:

```
gofulmen/ailink/
├── driver/       # Driver interface + implementations
├── content/      # ContentBlock, Message types
├── prompt/       # Schema validation, registry interface
└── types.go      # Core types
```

**What stays in namelens**:

```
internal/ailink/
├── prompts/      # App-specific prompt definitions
├── search.go     # SearchService
└── handlers.go   # App-specific response handling
```

### 7. Error handling strategy

**Decision**: On ailink failure, return deterministic results + structured
error.

```go
type CheckOutput struct {
    Results     []*core.CheckResult `json:"results"`
    AILink      *SearchResponse     `json:"ailink,omitempty"`
    AILinkError *SearchError        `json:"ailink_error,omitempty"`
}

type SearchError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Details string `json:"details,omitempty"`
}
```

**Rationale**:

- Never fail the entire command due to ailink issues
- Structured error enables programmatic handling
- Graceful degradation to deterministic-only results

### 8. Timeout limits

**Decision**: 60s default, 5 minute hard maximum.

```go
const (
    DefaultTimeout = 60 * time.Second
    MaxTimeout     = 5 * time.Minute
)
```

**Rationale**:

- Tool-using models (Grok with web_search) can take 10-30s
- 5 minute cap prevents runaway costs
- Configurable for deep analysis use cases

## Consequences

### Positive

- Clean, auditable integration with no SDK magic
- Easy to add new AI providers
- Prompt customization without code changes
- Schema validation catches issues early
- Extraction to gofulmen will be straightforward

### Negative

- More boilerplate per provider than SDK approach
- Must manually track provider API changes
- Two-phase extraction (local first, then gofulmen)

### Risks

- Provider API changes (mitigated: direct HTTP is explicit, changes visible)
- Prompt engineering complexity (mitigated: embedded defaults work out of box)
- Cost runaway (mitigated: timeout limits, cache TTL)

## References

- [xAI Grok API](https://docs.x.ai/api)
- [Anthropic Messages API](https://docs.anthropic.com/en/api/messages)
- [OpenAI-compatible Chat Completions](https://platform.openai.com/docs/api-reference/chat)
- [gofulmen/schema](https://github.com/fulmenhq/gofulmen/tree/main/schema)
