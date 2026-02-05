# ADR-0002: Control Plane API (Supersedes MCP)

**Status**: Accepted  
**Date**: 2026-02-02  
**Deciders**: @3leapsdave  
**Supersedes**: [ADR-0001](ADR-0001-mcp-architecture.md)

## Context

NameLens needs a programmatic interface for AI agents and automation. The
original plan (ADR-0001) proposed MCP (Model Context Protocol) for AI assistant
integration.

After evaluating real-world usage patterns, we've determined that:

1. **Agents are the primary consumers** - AI agents increasingly drive namelens
   workflows, not humans in Claude Desktop
2. **HTTP is universal** - Every agent framework speaks HTTP; MCP is
   Anthropic-specific
3. **Remote deployment is required** - Containerized/cloud deployment needs
   network API, not stdio
4. **OpenAPI enables tooling** - Auto-generated clients, documentation,
   validation, mocks

MCP adds complexity (two transports, stdio purity issues, SDK dependency)
without proportional benefit when a well-designed HTTP API serves all use cases
better.

## Decision

### 1. Build Control Plane HTTP API, not MCP

**Chosen**: Schema-first REST API with OpenAPI 3.1 specification

**Rationale**:

- Universal compatibility (any HTTP client, any language)
- OpenAPI spec enables client generation, documentation, validation
- Simpler implementation (one transport, standard HTTP semantics)
- Container/serverless deployment patterns work naturally
- If MCP demand emerges, adapters exist (OpenAPI-to-MCP bridges)

### 2. Schema-First with Enforced Validation

**Chosen**: OpenAPI spec is source of truth, enforced at compile and runtime

The spec lives at repository root for visibility:

```
/openapi.yaml          # Source of truth
```

#### Enforcement Toolchain

| Stage              | Tool             | Purpose                                  |
| ------------------ | ---------------- | ---------------------------------------- |
| Spec validation    | `vacuum`         | Lint OpenAPI spec for errors and style   |
| Code generation    | `oapi-codegen`   | Generate Go types and server interface   |
| Runtime validation | `kin-openapi`    | Validate requests/responses against spec |
| CI gate            | `make check-api` | Block merge if spec/code diverge         |

#### Enforcement Method

1. **Build-time**: `oapi-codegen` generates:
   - Request/response structs from schema components
   - Server interface that handlers must implement
   - Strict types (no `interface{}` escape hatches)

2. **Runtime**: `kin-openapi` middleware validates:
   - Incoming requests against spec (400 on violation)
   - Outgoing responses against spec (500 + alert on violation in dev)

3. **CI gate**: `make check-api` runs:
   ```bash
   # Regenerate from spec
   oapi-codegen -generate types,chi-server -package api openapi.yaml > internal/api/openapi.gen.go
   # Fail if generated code differs from committed
   git diff --exit-code internal/api/openapi.gen.go
   ```

This ensures the spec cannot drift from implementation - any schema change
requires regenerating code, and any code change without spec update fails CI.

#### Makefile Targets

```makefile
.PHONY: api-lint api-generate check-api

api-lint:
	vacuum lint openapi.yaml

api-generate:
	oapi-codegen -generate types,chi-server -package api \
		-o internal/api/openapi.gen.go openapi.yaml

check-api: api-lint api-generate
	git diff --exit-code internal/api/openapi.gen.go || \
		(echo "API spec and generated code are out of sync" && exit 1)
```

### 3. Localhost-Only by Default

**Security posture**: Bind to `127.0.0.1` unless explicitly overridden

```bash
# Default (safe)
namelens serve

# Explicit network exposure (warns user)
namelens serve --bind=0.0.0.0:8080
```

Network binding emits warning:

```
WARNING: Server bound to 0.0.0.0:8080 - exposed to network
         Use a reverse proxy (nginx, caddy, cloudflared) for production
```

### 4. Simple API Key Authentication

**Chosen**: `X-API-Key` header, optional for localhost

- No OAuth complexity
- No JWT token management
- Works with any HTTP client
- Localhost requests allowed without key (development mode)

```bash
# Generate a key
namelens serve --generate-key
# Output: Generated API key: nlcp_xxxxxxxxxxxx

# Configure
export NAMELENS_NAMELENS_CONTROL_PLANE_API_KEY=nlcp_xxxxxxxxxxxx
```

### 5. Single Port, Unified Server

Control plane runs on the same server as health/metrics endpoints:

| Endpoint            | Auth  | Purpose                 |
| ------------------- | ----- | ----------------------- |
| `GET /health`       | No    | Kubernetes probes       |
| `GET /metrics`      | No    | Prometheus metrics      |
| `GET /version`      | No    | Version info            |
| `POST /v1/check`    | Yes\* | Name availability check |
| `POST /v1/compare`  | Yes\* | Compare candidates      |
| `POST /v1/generate` | Yes\* | Generate alternatives   |
| `GET /v1/profiles`  | Yes\* | List profiles           |
| `GET /v1/status`    | Yes\* | Rate limit status       |

\*Auth required for non-localhost requests when API key configured

## Consequences

### Positive

- Universal agent compatibility (HTTP everywhere)
- Schema-first prevents API drift
- Simpler codebase (no MCP SDK, no stdio transport)
- Standard deployment patterns (containers, proxies, load balancers)
- Generated clients for any language from OpenAPI spec

### Negative

- No native Claude Desktop integration (would need MCP adapter)
- Must maintain OpenAPI spec discipline

### Risks

- OpenAPI tooling churn (mitigated: oapi-codegen and kin-openapi are mature)
- Schema-first discipline requires team buy-in (mitigated: CI enforcement)

## Migration

ADR-0001 planned MCP but no code was implemented. No migration required - this
ADR establishes the path forward.

Documentation referencing MCP will be updated to describe the Control Plane API.

## References

- [OpenAPI 3.1 Specification](https://spec.openapis.org/oas/v3.1.0)
- [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) - Go code
  generator
- [kin-openapi](https://github.com/getkin/kin-openapi) - Runtime validation
- [vacuum](https://quobix.com/vacuum/) - OpenAPI linter
- ADR-0001: MCP Server Architecture (superseded)
