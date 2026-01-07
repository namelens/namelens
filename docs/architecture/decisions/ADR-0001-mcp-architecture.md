# ADR-0001: MCP Server Architecture

**Status**: Accepted  
**Date**: 2024-12-27  
**Deciders**: @3leapsdave

## Context

NameLens needs MCP (Model Context Protocol) support to enable AI assistant
integration. MCP allows AI tools like Claude Code, OpenCode, and Codex to invoke
NameLens's name availability checking directly.

Key decisions required:

1. Which Go MCP library to use
2. Which transport(s) to support
3. How to structure for future extraction to gofulmen

## Decision

### 1. Use mark3labs/mcp-go as the MCP SDK

**Chosen**: `github.com/mark3labs/mcp-go`

**Alternatives considered**:

| Library                       | Pros                                         | Cons                                   |
| ----------------------------- | -------------------------------------------- | -------------------------------------- |
| `modelcontextprotocol/go-sdk` | Official Anthropic SDK                       | Newer, less mature, API still evolving |
| `mark3labs/mcp-go`            | Mature, widely adopted, clean API, good docs | Third-party                            |
| `metoro-io/mcp-golang`        | Gin integration                              | Less active, smaller community         |

**Rationale**:

- `mcp-go` has explicit transport support: `ServeStdio()`, `NewSSEServer()`,
  `NewStreamableHTTPServer()`
- Clean tool registration API with generics
- Battle-tested in production MCP servers
- Active maintenance and community
- MIT licensed

### 2. HTTP-first with stdio wrapper

**Primary transport**: HTTP with SSE (Server-Sent Events) - legacy 2024-11-05
spec

**Secondary transport**: stdio (for Claude Desktop and VS Code extensions)

**Rationale**:

- NameLens already has a Chi HTTP server with `/health`, `/version`, `/metrics`
- Adding MCP as an HTTP endpoint is natural extension
- stdio has stdout purity challenges (any stray print breaks JSON-RPC)
- HTTP allows remote deployment, team sharing, service mode
- Claude Code CLI supports both; VS Code extensions often require stdio

**Transport matrix**:

| Client             | HTTP/SSE  | stdio     | Notes                                     |
| ------------------ | --------- | --------- | ----------------------------------------- |
| Claude Code CLI    | Preferred | Supported | `claude mcp add --transport sse`          |
| OpenCode           | Preferred | Supported | Prefers StreamableHTTP, falls back to SSE |
| Codex CLI          | Supported | Supported | Config-based                              |
| Claude Desktop     | Limited   | Preferred | May require stdio                         |
| VS Code extensions | No        | Required  | Cline, Roo, Continue                      |

**Deferred**: Streamable HTTP (2025-03-26 spec) - not yet widely supported by
clients.

### 3. ToolService interface pattern for gofulmen extraction

The MCP implementation uses a `ToolService` interface to decouple business logic
from MCP transport:

```go
// ToolService defines the contract between MCP handler and business logic.
// This interface stays in namelens; gofulmen provides MCP plumbing.
type ToolService interface {
    CheckName(ctx context.Context, req CheckRequest) (*CheckResponse, error)
    BatchCheck(ctx context.Context, req BatchRequest) (*BatchResponse, error)
}
```

**What extracts to gofulmen later**:

```
gofulmen/mcp/
├── server.go        # MCP server wrapper
├── transport/
│   ├── sse.go       # SSE transport (Chi middleware)
│   └── stdio.go     # stdio transport
└── types.go         # MCP protocol types
```

**What stays in namelens**:

```
internal/mcp/
├── server.go    # MCP server setup
├── tools.go     # Tool definitions
├── service.go   # ToolService implementation
└── format.go    # Response formatting
```

### 4. SSE endpoint structure

Follow SSE transport conventions:

- `GET /mcp/sse` - SSE event stream (server-to-client)
- `POST /mcp/message` - JSON-RPC messages (client-to-server)

Mounted on existing Chi router via `r.Mount("/mcp", sseServer.Handler())`.

## Consequences

### Positive

- Clean separation allows future gofulmen extraction
- HTTP-first avoids stdout purity debugging
- Works with major MCP clients out of the box
- Leverages existing HTTP server infrastructure

### Negative

- Third-party SDK dependency (mcp-go)
- stdio mode still needed for some clients (VS Code extensions)
- Two transports to maintain

### Risks

- mcp-go API changes (mitigated: stable v0.28+, semantic versioning)
- MCP spec evolution (mitigated: SSE is stable legacy spec)

## References

- [MCP Specification](https://modelcontextprotocol.io/specification)
- [mark3labs/mcp-go](https://github.com/mark3labs/mcp-go)
- [MCP Transports](https://modelcontextprotocol.io/specification/basic/transports)
