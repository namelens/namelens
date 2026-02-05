---
description: HTTP API reference for programmatic name checking and generation
---

# HTTP API Quick Reference

**Base URL**: `http://localhost:8080` (default, configurable)

NameLens exposes a REST API for programmatic access to name checking and
generation. Useful for CI/CD pipelines, automated naming workflows, and
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
```

### Foreground Mode (Development)

```bash
# Run in foreground (logs to console)
namelens serve

# With custom port
namelens serve --port 3000
```

### Configuration

Edit `~/.config/namelens/config.yaml`:

```yaml
server:
  port: 8080              # HTTP API port
  host: localhost         # Bind address (0.0.0.0 for all interfaces)
  read_timeout: 30s
  write_timeout: 30s
```

Or use environment variables:

```bash
export NAMELENS_PORT=3000
export NAMELENS_HOST=0.0.0.0
```

## API Endpoints

### Health Check

```bash
GET /health
```

**Response**:

```json
{
  "status": "healthy",
  "version": "0.2.1",
  "timestamp": "2026-02-03T14:17:42Z"
}
```

### Check Name Availability

```bash
POST /api/v1/check
Content-Type: application/json

{
  "name": "myproject",
  "tlds": ["com", "io", "dev"],
  "registries": ["npm", "pypi"],
  "handles": ["github"],
  "expert": false
}
```

**Response**:

```json
{
  "name": "myproject",
  "results": [
    {
      "type": "domain",
      "name": "myproject.com",
      "available": false,
      "details": "exp: 2026-06-15; registrar: Example Inc."
    },
    {
      "type": "domain",
      "name": "myproject.io",
      "available": true
    },
    {
      "type": "npm",
      "name": "myproject",
      "available": true
    }
  ],
  "summary": {
    "total": 5,
    "available": 3,
    "taken": 2
  }
}
```

### Generate Name Alternatives

```bash
POST /api/v1/generate
Content-Type: application/json

{
  "concept": "API gateway for microservices",
  "description": "Unified control plane for managing services across multiple cloud providers",
  "depth": "quick",
  "constraints": "Prefer short names under 8 characters"
}
```

**Response**:

```json
{
  "concept_analysis": {
    "core_function": "...",
    "key_themes": ["unification", "orchestration", "multi-cloud"],
    "target_audience": "Platform engineers"
  },
  "candidates": [
    {
      "name": "omnigate",
      "strategy": "compound",
      "strength": "strong",
      "rationale": "Combines 'omni' (all/universal) with 'gate' (gateway)"
    }
  ]
}
```

## Common Use Cases

### CI/CD Pipeline Check

Check proposed project names before creating repos:

```bash
#!/bin/bash
# .github/workflows/check-name.yml

PROJECT_NAME=${GITHUB_REPOSITORY##*/}

# Check availability
curl -s -X POST http://localhost:8080/api/v1/check \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$PROJECT_NAME\", \"expert\": true}" | \
  jq -e '.summary.available > 0' || {
    echo "❌ Project name '$PROJECT_NAME' has conflicts"
    exit 1
  }

echo "✅ Project name looks good"
```

### Pre-Commit Hook

Validate package names before publishing:

```bash
#!/bin/bash
# .git/hooks/pre-commit

PACKAGE_NAME=$(jq -r '.name' package.json)

# Quick check (no expert for speed)
RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/check \
  -H "Content-Type: application/json" \
  -d "{\"name\": \"$PACKAGE_NAME\", \"tlds\": [\"io\"], \"registries\": [\"npm\"]}")

if echo "$RESPONSE" | jq -e '.results[] | select(.type == "npm" and .available == false)' > /dev/null; then
  echo "❌ npm package name '$PACKAGE_NAME' is already taken"
  exit 1
fi
```

### Bulk Generation via Script

Generate names for a new feature and automatically check them:

```bash
#!/usr/bin/env bash

CONCEPT="$1"

# Generate candidates
GENERATE_RESPONSE=$(curl -s -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d "{\"concept\": \"$CONCEPT\", \"depth\": \"quick\"}")

# Extract top 3 names
NAMES=$(echo "$GENERATE_RESPONSE" | jq -r '.candidates[0:3].name')

# Check all 3 in one call
echo "Checking: $NAMES"
curl -s -X POST http://localhost:8080/api/v1/check/bulk \
  -H "Content-Type: application/json" \
  -d "{\"names\": [$(echo "$NAMES" | tr '\n' ',' | sed 's/,$//')], \"expert\": true}"
```

## Error Handling

### Common HTTP Status Codes

| Code | Meaning             | Action                    |
| ---- | ------------------- | ------------------------- |
| 200  | Success             | Process response          |
| 400  | Bad Request         | Check request JSON format |
| 404  | Not Found           | Endpoint doesn't exist    |
| 429  | Rate Limited        | Slow down requests        |
| 500  | Server Error        | Check server logs         |
| 503  | Service Unavailable | Server may be starting up |

### Error Response Format

```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Name parameter is required",
    "details": {
      "field": "name",
      "reason": "missing"
    }
  }
}
```

## Performance Tips

1. **Use bulk endpoints** for multiple names:
   - `POST /api/v1/check/bulk` instead of multiple individual checks
   - Reduces AI costs by 60-80%

2. **Cache results** for repeated checks:
   - NameLens has built-in caching (24h default)
   - Use `--no-cache` only when you need fresh data

3. **Match depth to urgency**:
   - `depth: quick` for initial screening (faster, cheaper)
   - `depth: deep` for finalists (thorough analysis)

4. **Parallelize non-AI checks**:
   - Domain/registry checks are fast and parallelized
   - Only use `expert: true` for names that pass basic checks

## Security Considerations

- **Default binding**: `localhost:8080` (local only)
- **To expose externally**: Set `host: 0.0.0.0` and use a reverse proxy
- **No authentication by default**: Add proxy auth if exposing to internet
- **Rate limiting**: Built-in (configurable in config.yaml)

## Troubleshooting

### Server won't start

```bash
# Check if port is in use
namelens serve cleanup

# Or check manually
lsof -i :8080

# Kill existing process
namelens serve stop --force
```

### Connection refused

```bash
# Check server is running
namelens serve status

# Check health endpoint
curl http://localhost:8080/health

# Check port configuration
namelens envinfo | grep port
```

### API returns errors

```bash
# Check server logs
# (When running in foreground, logs go to console)

# Test with simple request
curl -v http://localhost:8080/health
```

## Further Reading

- [Workflows & Best Practices](workflows.md) - Choosing providers and optimizing
  costs
- [Configuration Guide](configuration.md) - Server settings and provider routing
- [Quick Start Guide](quick-start.md) - CLI usage and basic commands

---

_API documentation for namelens v0.2.1. Server must be running for API calls to
succeed._
