---
description: HTTP API reference for programmatic name checking
---

# HTTP API Reference

**Base URL**: `http://localhost:8080` (default, configurable)

NameLens exposes a REST API for programmatic access to name availability
checking. Useful for CI/CD pipelines, automated naming workflows, and
integration with other tools.

## Starting the Server

### Daemon Mode (Background)

```bash
# Start server in background
namelens serve --daemon

# Check status
namelens serve status

# Stop gracefully
namelens serve stop

# Force kill if needed
namelens serve stop --force

# Clean up stale PID files
namelens serve cleanup
```

### Foreground Mode (Development)

```bash
# Run in foreground (logs to console)
namelens serve

# With custom port
namelens serve --port 3000
```

### Configuration

Server settings can be set in the config file, environment variables, or a
`.env` file. See the [Configuration Guide](configuration.md) for full details.

Edit `~/.config/namelens/config.yaml`:

```yaml
server:
  port: 8080 # HTTP API port
  host: localhost # Bind address — keep as localhost unless you need network access
  read_timeout: 30s
  write_timeout: 30s
```

Or use environment variables:

```bash
export NAMELENS_PORT=3000
export NAMELENS_HOST=localhost  # Default; change only if you need network access
```

Or use a `.env` file (copy from `.env.example` in the repo):

```bash
namelens serve --env-file ~/.config/namelens/.env
```

> **Recommendation**: Keep the default `localhost` bind address unless you have
> a specific need for network access. If you do bind to `0.0.0.0`, configure an
> API key and use a reverse proxy.

## Authentication

The Control Plane API uses API key authentication for securing access beyond
localhost.

> **v0.2.1 security model**: The current authentication is designed for
> localhost and trusted-network deployments. It provides a shared API key via
> the `X-API-Key` header. It does not yet support per-user keys, OAuth, mTLS, or
> automatic key rotation. For public-facing deployments, always place namelens
> behind a reverse proxy with its own TLS and authentication layer.

### Generating an API Key

```bash
# Generate a new API key
namelens serve --generate-key
# Output: nlcp_a1b2c3d4e5f6...
```

### Starting with Authentication

```bash
# Pass key directly
namelens serve --daemon --api-key nlcp_a1b2c3d4e5f6...

# Or use environment variable
export NAMELENS_CONTROL_PLANE_API_KEY=nlcp_a1b2c3d4e5f6...
namelens serve --daemon

# Or use .env file
namelens serve --daemon --env-file ~/.config/namelens/.env
```

### Making Authenticated Requests

```bash
curl -H "X-API-Key: nlcp_a1b2c3d4e5f6..." \
  http://localhost:8080/v1/check \
  -d '{"name": "myproject"}'
```

### Authentication Behavior

| Scenario                            | Result                                |
| ----------------------------------- | ------------------------------------- |
| No key configured                   | All requests allowed                  |
| Key configured + localhost (no key) | Allowed (dev convenience)             |
| Key configured + localhost + key    | Key validated (catches config errors) |
| Key configured + remote (no key)    | 401 Unauthorized                      |
| Key configured + wrong key          | 401 Unauthorized                      |

## API Endpoints

### Health Check

```
GET /health
```

Returns aggregate server health status. Suitable for Kubernetes probes and
monitoring.

**Response** (200 OK):

```json
{
  "status": "healthy",
  "version": "0.2.1",
  "checks": {
    "app_identity": { "status": "pass" },
    "signal_handlers": { "status": "pass" },
    "telemetry": { "status": "pass" }
  }
}
```

**Status values**: `healthy`, `degraded`, `unhealthy`  
**Check status values**: `pass`, `fail`, `warn`

### Kubernetes Probes

```
GET /health/live     # Liveness probe
GET /health/ready    # Readiness probe
GET /health/startup  # Startup probe
```

**Response** (200 OK):

```json
{
  "status": "healthy",
  "timestamp": "2026-02-05T10:30:00Z"
}
```

### Server Status

```
GET /v1/status
```

Returns provider status and rate limit headroom.

**Response** (200 OK):

```json
{
  "providers": {
    "domain": { "available": true },
    "npm": { "available": true },
    "github": { "available": true }
  }
}
```

### List Profiles

```
GET /v1/profiles
```

Returns all available check profiles (built-in and custom).

**Response** (200 OK):

```json
{
  "profiles": [
    {
      "name": "startup",
      "description": "Common startup TLDs and registries",
      "tlds": ["com", "io", "dev", "co"],
      "registries": ["npm", "pypi"],
      "handles": ["github"],
      "is_builtin": true
    },
    {
      "name": "developer",
      "description": "Developer-focused TLDs and all registries",
      "tlds": ["dev", "io", "sh", "run"],
      "registries": ["npm", "pypi", "cargo"],
      "handles": ["github"],
      "is_builtin": true
    }
  ]
}
```

### Check Name Availability

```
POST /v1/check
Content-Type: application/json
```

Check availability of a name across domains, registries, and handles.

**Request Body**:

```json
{
  "name": "myproject",
  "profile": "startup",
  "expert": false,
  "tlds": ["com", "io", "dev"],
  "registries": ["npm"],
  "handles": ["github"]
}
```

| Field        | Type     | Required | Description                                              |
| ------------ | -------- | -------- | -------------------------------------------------------- |
| `name`       | string   | Yes      | Name to check (1-63 chars)                               |
| `profile`    | string   | No       | Profile: startup, developer, oss, minimal, website, web3 |
| `expert`     | boolean  | No       | Enable AI brand safety analysis                          |
| `tlds`       | string[] | No       | Custom TLDs (overrides profile)                          |
| `registries` | string[] | No       | Custom registries: npm, pypi, cargo                      |
| `handles`    | string[] | No       | Custom handles: github                                   |

**Response** (200 OK):

```json
{
  "name": "myproject",
  "results": [
    {
      "name": "myproject.com",
      "check_type": "domain",
      "tld": "com",
      "available": "taken",
      "provenance": {
        "from_cache": false,
        "source": "rdap",
        "server": "https://rdap.verisign.com/com/v1"
      }
    },
    {
      "name": "myproject.io",
      "check_type": "domain",
      "tld": "io",
      "available": "available",
      "provenance": {
        "from_cache": true,
        "source": "rdap"
      }
    },
    {
      "name": "myproject",
      "check_type": "npm",
      "available": "available",
      "provenance": {
        "from_cache": false,
        "source": "registry"
      }
    }
  ],
  "summary": {
    "total": 3,
    "available": 2,
    "taken": 1,
    "unknown": 0,
    "risk_level": "high"
  }
}
```

**Availability values**: `available`, `taken`, `unknown`, `error`,
`rate_limited`, `unsupported`

**Risk levels**: `low`, `medium`, `high` (high if .com is taken)

### Compare Multiple Names

```
POST /v1/compare
Content-Type: application/json
```

Compare multiple name candidates side-by-side.

**Request Body**:

```json
{
  "names": ["acmecorp", "acmeio", "acmehq"],
  "profile": "startup",
  "expert": false
}
```

| Field        | Type     | Required | Description                     |
| ------------ | -------- | -------- | ------------------------------- |
| `names`      | string[] | Yes      | 2-10 names to compare           |
| `profile`    | string   | No       | Check profile to use            |
| `expert`     | boolean  | No       | Enable AI analysis per name     |
| `tlds`       | string[] | No       | Custom TLDs (overrides profile) |
| `registries` | string[] | No       | Custom registries               |
| `handles`    | string[] | No       | Custom handles                  |

**Response** (200 OK):

```json
{
  "candidates": [
    {
      "name": "acmecorp",
      "results": [
        {
          "name": "acmecorp.com",
          "check_type": "domain",
          "tld": "com",
          "available": "taken"
        }
      ],
      "summary": {
        "total": 5,
        "available": 3,
        "taken": 2,
        "unknown": 0,
        "risk_level": "high"
      }
    },
    {
      "name": "acmeio",
      "results": [...],
      "summary": {
        "total": 5,
        "available": 5,
        "taken": 0,
        "unknown": 0,
        "risk_level": "low"
      }
    }
  ]
}
```

## Error Handling

### HTTP Status Codes

| Code | Meaning             | Action                    |
| ---- | ------------------- | ------------------------- |
| 200  | Success             | Process response          |
| 400  | Bad Request         | Check request JSON format |
| 401  | Unauthorized        | Provide valid API key     |
| 404  | Not Found           | Endpoint doesn't exist    |
| 429  | Rate Limited        | Retry after delay         |
| 500  | Server Error        | Check server logs         |
| 503  | Service Unavailable | Server may be starting up |

### Error Response Format

All errors return a consistent JSON structure:

```json
{
  "error": {
    "code": "bad_request",
    "message": "name is required"
  }
}
```

**Error codes** (lowercase snake_case):

- `bad_request` - Invalid request parameters
- `unauthorized` - Missing or invalid API key
- `not_found` - Resource not found
- `rate_limited` - Too many requests (includes `retry_after`)
- `internal_error` - Server error

Rate limit errors include retry timing:

```json
{
  "error": {
    "code": "rate_limited",
    "message": "rate limit exceeded, retry after 60 seconds",
    "retry_after": 60
  }
}
```

## Common Use Cases

### CI/CD Pipeline Check

Check proposed project names before creating repos:

```bash
#!/bin/bash
PROJECT_NAME=${GITHUB_REPOSITORY##*/}

RESPONSE=$(curl -s -X POST http://localhost:8080/v1/check \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$PROJECT_NAME\", \"profile\": \"developer\"}")

# Check if .com is available (risk_level != high)
RISK=$(echo "$RESPONSE" | jq -r '.summary.risk_level')
if [ "$RISK" = "high" ]; then
  echo "Warning: $PROJECT_NAME.com is taken"
  exit 1
fi

echo "Name looks good"
```

### Pre-Commit Hook

Validate package names before publishing:

```bash
#!/bin/bash
PACKAGE_NAME=$(jq -r '.name' package.json)

RESPONSE=$(curl -s -X POST http://localhost:8080/v1/check \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$PACKAGE_NAME\", \"registries\": [\"npm\"]}")

NPM_TAKEN=$(echo "$RESPONSE" | jq -r '.results[] | select(.check_type == "npm" and .available == "taken")')
if [ -n "$NPM_TAKEN" ]; then
  echo "npm package '$PACKAGE_NAME' is already taken"
  exit 1
fi
```

### Compare Naming Options

Evaluate multiple candidates for a new project:

```bash
curl -s -X POST http://localhost:8080/v1/compare \
  -H "Content-Type: application/json" \
  -d '{
    "names": ["quicksync", "fastsync", "rapidsync"],
    "profile": "startup"
  }' | jq '.candidates[] | {name, risk: .summary.risk_level, available: .summary.available}'
```

## OpenAPI Specification

The complete API specification is available in OpenAPI 3.1 format:

- **File**: `openapi.yaml` in the repository root
- **Use for**: SDK generation, API documentation tools, testing

## Performance Tips

1. **Use profiles** instead of custom TLD lists for common use cases
2. **Cache results** - NameLens caches responses (24h default)
3. **Batch comparisons** - Use `/v1/compare` for multiple names
4. **Skip expert mode** for initial screening (faster, cheaper)

## Troubleshooting

### Server won't start

```bash
# Clean up stale PID files
namelens serve cleanup

# Check if port is in use
lsof -i :8080

# Stop a managed namelens server
namelens serve stop --force

# If the process isn't managed by namelens (no PID file), stop will refuse.
# Use cleanup instead to terminate any process on the port:
namelens serve cleanup --port 8080
```

### Connection refused

```bash
# Check server is running
namelens serve status

# Test health endpoint
curl http://localhost:8080/health

# Check which port is configured
namelens envinfo | grep port
```

### Authentication errors

```bash
# Verify key is set
echo $NAMELENS_CONTROL_PLANE_API_KEY

# Test with explicit key
curl -H "X-API-Key: $NAMELENS_CONTROL_PLANE_API_KEY" \
  http://localhost:8080/v1/profiles
```

## Further Reading

- [Workflows & Best Practices](workflows.md) - Choosing providers and optimizing
  costs
- [Configuration Guide](configuration.md) - Server settings and provider routing
- [Quick Start Guide](quick-start.md) - CLI usage and basic commands

---

_API documentation for namelens v0.2.1. See `openapi.yaml` for the complete
specification._
