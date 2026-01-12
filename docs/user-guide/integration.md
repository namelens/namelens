# Integration & Automation

MCP server, API, and CI/CD integration for Namelens.

---

## MCP Server

Namelens provides an MCP (Model Context Protocol) server for integration with AI
assistants like Claude, OpenCode, and other MCP-compatible tools.

### Start the Server

```bash
namelens serve
# Or with custom port
namelens serve --port 9000
```

The server starts on `http://localhost:8080` (or custom port) with MCP endpoints
mounted under `/mcp`.

### MCP Tools

The following tools are exposed:

| Tool                      | Description                                          |
| ------------------------- | ---------------------------------------------------- |
| `check_name_availability` | Check a single name across TLDs, registries, handles |
| `batch_check`             | Check multiple names from a list                     |

#### check_name_availability

Parameters:

- `name` (required) — Name to check
- `profile` (optional) — Predefined profile: `startup`, `minimal`, `web3`
- `tlds` (optional) — Specific TLDs: `["com", "io", "dev"]`
- `registries` (optional) — Package registries: `["npm", "pypi"]`
- `handles` (optional) — Social handles: `["github"]`

#### batch_check

Parameters:

- `names` (required) — Array of names to check
- `profile` (optional) — Predefined profile
- `tlds` (optional) — Specific TLDs

### AI Provider Transparency

When using MCP tools with expert analysis enabled, Namelens maintains its direct HTTP
interface to AI providers. This means:

- **No SDK mediation** — Requests go directly from Namelens to AI providers
- **Full auditability** — Enable debug logging to see complete request/response:
  ```bash
  NAMELENS_LOG_LEVEL=debug namelens serve
  ```
- **Controlled pipeline** — You define prompts, timeouts, and caching behavior
- **No hidden telemetry** — No data sent without your explicit configuration

See [Expert Search Guide](expert-search.md) for details on direct provider
architecture.

### Configure with Claude Desktop

Add to your Claude Desktop MCP config
(`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "namelens": {
      "command": "/path/to/namelens",
      "args": ["serve"]
    }
  }
}
```

### Configure with OpenCode

Add to your OpenCode MCP configuration:

```json
{
  "mcpServers": {
    "namelens": {
      "transport": "http",
      "url": "http://localhost:8080/mcp",
      "transportType": "sse"
    }
  }
}
```

### Example: AI Assistant Workflow

1. **Ask assistant to check names:**

   ```
   "Check if 'stellaplex' is available across domains and npm"
   ```

2. **Assistant calls MCP tool:**

   ```json
   {
     "name": "check_name_availability",
     "arguments": {
       "name": "stellaplex",
       "profile": "startup"
     }
   }
   ```

3. **Assistant interprets results and provides insights**

---

## CLI API

For direct programmatic use, Namelens provides CLI commands with structured
output.

### JSON Output

All commands support `--output-format=json`:

```bash
namelens check myproject --output-format=json
namelens batch candidates.txt --output-format=json
```

### Pipe Processing

Combine with other tools:

```bash
# Extract available TLDs
namelens check myproject --output-format=json | \
  jq '.results[] | select(.check_type == "domain" and .available == true) | .tld'

# Count total availability
namelens check myproject --output-format=json | \
  jq '.results | map(select(.available == true)) | length'
```

### Script Integration

```bash
#!/bin/bash
# check-and-register.sh

NAME=$1
RESULT=$(namelens check "$NAME" --profile=startup --output-format=json)

# Extract availability score
SCORE=$(echo "$RESULT" | jq '[.results[] | select(.available == true)] | length')
TOTAL=$(echo "$RESULT" | jq '.results | length')

if [ "$SCORE" -eq "$TOTAL" ]; then
  echo "✅ $NAME is fully available - proceed with registration"
  # Add registration logic here
else
  echo "⚠️  $NAME has $SCORE/$TOTAL available - review conflicts:"
  echo "$RESULT" | jq '.results[] | select(.available == false)'
fi
```

---

## CI/CD Integration

### GitHub Actions

Check name availability as part of your workflow:

```yaml
name: Name Availability Check

on:
  pull_request:
    paths:
      - 'package.json'  # Or files containing project name

jobs:
  check-name:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Namelens
        run: |
          wget https://github.com/3leaps/namelens/releases/latest/download/namelens-linux-amd64
          chmod +x namelens-linux-amd64

      - name: Check Project Name
        run: |
          # Extract name from package.json or config
          NAME=$(jq -r '.name' package.json)

          # Run availability check
          ./namelens-linux-amd64 check "$NAME" --profile=startup --output-format=json --out result.json

          # Fail if .com or npm is taken
          CONFLICT=$(jq '[.results[] | select(
            (.check_type == "domain" and .tld == "com" and .available == false) or
            (.check_type == "npm" and .available == false)
          )] | length')

          if [ "$CONFLICT" -gt 0 ]; then
            echo "❌ Name conflicts found:"
            cat result.json
            exit 1
          fi

          echo "✅ No conflicts found"
```

### Pre-Commit Hook

Warn before committing with potential name conflicts:

```bash
#!/bin/bash
# .git/hooks/pre-commit

# Check if package.json was modified
if git diff --cached --name-only | grep -q 'package.json'; then
  NAME=$(jq -r '.name' package.json)

  echo "Checking availability for: $NAME"
  RESULT=$(namelens check "$NAME" --profile=minimal --output-format=json)

  # Check .com availability
  COM_AVAIL=$(jq '.results[] | select(.check_type == "domain" and .tld == "com") | .available' <<< "$RESULT")

  if [ "$COM_AVAIL" == "false" ]; then
    echo "⚠️  Warning: $NAME.com is taken"
    echo "Continue commit? (y/n)"
    read -r response
    if [ "$response" != "y" ]; then
      exit 1
    fi
  fi
fi
```

### Continuous Monitoring

Run periodic checks on your brand portfolio:

```yaml
# GitHub Actions workflow
name: Brand Portfolio Monitor

on:
  schedule:
    - cron: '0 9 * * 1'  # Every Monday 9am

jobs:
  monitor:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Namelens
        run: # ... (same as above)

      - name: Check Portfolio
        run: |
          namelens batch brand-portfolio.txt \
            --profile=startup \
            --expert \
            --output-format=json --out portfolio.json

          # Report issues
          ISSUES=$(jq '[.results[] | select(.score < 5)]' portfolio.json)
          if [ -n "$ISSUES" ]; then
            echo "Portfolio issues detected"
            echo "$ISSUES" | gh issue create \
              --title "Brand Portfolio Alert - $(date +%Y-%m-%d)" \
              --body-file -
          fi
```

---

## HTTP API

For custom integrations, Namelens exposes HTTP endpoints when running in server
mode.

### Health Check

```bash
curl http://localhost:8080/health
```

Response:

```json
{
  "status": "healthy",
  "version": "0.1.2"
}
```

### Name Check Endpoint

```bash
curl -X POST http://localhost:8080/api/v1/check \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acmecorp",
    "profile": "startup"
  }'
```

Response:

```json
{
  "name": "acmecorp",
  "results": [
    {
      "check_type": "domain",
      "tld": "com",
      "available": false,
      "extra_data": {"expires": "2026-11-17"}
    }
  ],
  "score": 4,
  "total": 8
}
```

### Batch Check Endpoint

```bash
curl -X POST http://localhost:8080/api/v1/batch \
  -H "Content-Type: application/json" \
  -d '{
    "names": ["acmecorp", "stellaplex"],
    "profile": "startup"
  }'
```

---

## Environment Configuration

Configure Namelens for different environments via environment variables:

### Development

```bash
export NAMELENS_LOG_LEVEL=debug
export NAMELENS_DB_PATH=./dev.db

# Optional: Enable expert features
export NAMELENS_EXPERT_ENABLED=true
export NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=your-dev-key

namelens serve
```

### Production

```bash
export NAMELENS_LOG_LEVEL=info
export NAMELENS_DB_PATH=/var/lib/namelens/namelens.db

# Use Turso for shared state
export NAMELENS_DB_URL=libsql://your-db.turso.io
export NAMELENS_DB_AUTH_TOKEN=your-auth-token

# Expert features (if using)
export NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=your-prod-key

namelens serve --port 9000
```

---

## Docker Integration

### Dockerfile

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o namelens ./cmd/namelens

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/namelens /usr/local/bin/
EXPOSE 8080
CMD ["namelens", "serve"]
```

### Docker Compose

```yaml
version: '3.8'

services:
  namelens:
    build: .
    ports:
      - "8080:8080"
    environment:
      - NAMELENS_LOG_LEVEL=info
      - NAMELENS_PORT=8080
      - NAMELENS_DB_PATH=/data/namelens.db
    volumes:
      - namelens-data:/data

volumes:
  namelens-data:
```

### Run in Docker

```bash
docker-compose up -d

# Check availability via API
curl -X POST http://localhost:8080/api/v1/check \
  -H "Content-Type: application/json" \
  -d '{"name": "acmecorp", "profile": "startup"}'
```

---

## Best Practices

1. **Cache appropriately** — Namelens caches results by default; respect cache
   TTL for rate limit compliance

2. **Error handling** — Always handle HTTP errors and timeouts gracefully in
   integrations

3. **Rate limiting** — Implement client-side rate limiting when calling Namelens
   APIs at scale

4. **Security** — Never expose API keys or credentials in CI logs or
   repositories

5. **Version pinning** — Pin to specific Namelens versions in production
   environments

6. **Monitoring** — Set up health checks and alerts for Namelens server
   deployments

7. **AI provider transparency** — When using expert features, enable debug
   logging to inspect full request/response to AI providers:
   ```bash
   NAMELENS_LOG_LEVEL=debug namelens serve
   ```
   Namelens uses direct HTTP connections (no SDKs), giving you full visibility
   into the AI pipeline for compliance and debugging.

---

## Need Help?

```bash
namelens serve --help
namelens doctor  # Diagnostics
```

See also:

- [Configuration](configuration.md) — Profiles and env vars
- [Quick Availability Check](quick-start.md) — Basic usage
- [Startup Naming Guide](startup-guide.md) — Full naming workflow
