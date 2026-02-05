# Integration & Automation

HTTP API, CLI integration, and CI/CD patterns for Namelens.

---

## HTTP API (Control Plane)

Namelens provides an HTTP API for integration with AI agents, automation tools,
and custom applications.

### Start the Server

```bash
namelens serve
# Or with custom bind address
namelens serve --bind=127.0.0.1:9000
```

The server starts on `http://localhost:8080` by default with API endpoints under
`/v1/`.

### Security

**Localhost-only by default**: The server binds to `127.0.0.1` unless you
explicitly specify a different address.

**API key authentication**: For non-localhost access, configure an API key:

```bash
# Generate a key
namelens serve --generate-key
# Output: Generated API key: nlcp_xxxxxxxxxxxx

# Set via environment
export NAMELENS_NAMELENS_CONTROL_PLANE_API_KEY=nlcp_xxxxxxxxxxxx

# Include in requests
curl -H "X-API-Key: nlcp_xxxxxxxxxxxx" http://localhost:8080/v1/check ...
```

**Network exposure**: If you need to expose the API to the network:

```bash
namelens serve --bind=0.0.0.0:8080
# WARNING: Server bound to 0.0.0.0:8080 - exposed to network
#          Use a reverse proxy (nginx, caddy, cloudflared) for production
```

Use a reverse proxy for TLS termination and additional authentication in
production deployments.

### API Endpoints

| Method | Endpoint       | Purpose                         |
| ------ | -------------- | ------------------------------- |
| `GET`  | `/health`      | Health check (no auth required) |
| `POST` | `/v1/check`    | Check a single name             |
| `POST` | `/v1/compare`  | Compare multiple candidates     |
| `POST` | `/v1/generate` | Generate name alternatives      |
| `POST` | `/v1/review`   | Deep brand review               |
| `GET`  | `/v1/profiles` | List available profiles         |
| `GET`  | `/v1/status`   | Rate limit and provider status  |

### OpenAPI Specification

The API is defined by an OpenAPI 3.1 spec at the repository root:

```
/openapi.yaml
```

Use this spec to:

- Generate clients in any language
- Import into Postman, Insomnia, or similar tools
- Validate requests/responses programmatically

### Example: Check Name

```bash
curl -X POST http://localhost:8080/v1/check \
  -H "Content-Type: application/json" \
  -d '{
    "name": "acmecorp",
    "profile": "startup",
    "expert": true
  }'
```

Response:

```json
{
  "name": "acmecorp",
  "results": [
    {
      "type": "domain",
      "name": "acmecorp.com",
      "status": "taken",
      "notes": "exp: 2026-11-17; registrar: GoDaddy"
    },
    {
      "type": "npm",
      "name": "acmecorp",
      "status": "available",
      "notes": ""
    }
  ],
  "summary": {
    "available": 5,
    "total": 8,
    "risk_level": "medium"
  },
  "expert": {
    "risk": "low",
    "analysis": "No direct trademark conflicts found..."
  }
}
```

### Example: Compare Candidates

```bash
curl -X POST http://localhost:8080/v1/compare \
  -H "Content-Type: application/json" \
  -d '{
    "names": ["acmecorp", "acmeio", "acmehq"],
    "profile": "startup",
    "phonetics": true
  }'
```

### Example: Generate Names

```bash
curl -X POST http://localhost:8080/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "concept": "static analyzer for shell scripts",
    "count": 10
  }'
```

### AI Provider Transparency

When using expert analysis via the API, Namelens maintains its direct HTTP
interface to AI providers. This means:

- **No SDK mediation** - Requests go directly from Namelens to AI providers
- **Full auditability** - Enable debug logging to see complete request/response:
  ```bash
  NAMELENS_NAMELENS_LOG_LEVEL=debug namelens serve
  ```
- **Controlled pipeline** - You define prompts, timeouts, and caching behavior
- **No hidden telemetry** - No data sent without your explicit configuration

See [Expert Search Guide](expert-search.md) for details on direct provider
architecture.

---

## CLI Integration

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
  echo "All clear - $NAME is fully available"
else
  echo "Conflicts found: $SCORE/$TOTAL available"
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
      - 'package.json'

jobs:
  check-name:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Namelens
        run: |
          wget https://github.com/namelens/namelens/releases/latest/download/namelens-linux-amd64
          chmod +x namelens-linux-amd64

      - name: Check Project Name
        run: |
          NAME=$(jq -r '.name' package.json)
          ./namelens-linux-amd64 check "$NAME" --profile=startup --output-format=json --out result.json

          CONFLICT=$(jq '[.results[] | select(
            (.check_type == "domain" and .tld == "com" and .available == false) or
            (.check_type == "npm" and .available == false)
          )] | length' result.json)

          if [ "$CONFLICT" -gt 0 ]; then
            echo "Name conflicts found"
            cat result.json
            exit 1
          fi
```

### Pre-Commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

if git diff --cached --name-only | grep -q 'package.json'; then
  NAME=$(jq -r '.name' package.json)
  echo "Checking availability for: $NAME"

  RESULT=$(namelens check "$NAME" --profile=minimal --output-format=json)
  COM_AVAIL=$(jq '.results[] | select(.check_type == "domain" and .tld == "com") | .available' <<< "$RESULT")

  if [ "$COM_AVAIL" == "false" ]; then
    echo "Warning: $NAME.com is taken"
    read -p "Continue commit? (y/n) " response
    [ "$response" != "y" ] && exit 1
  fi
fi
```

---

## Docker Integration

### Dockerfile

```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN CGO_ENABLED=1 make build

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/bin/namelens /usr/local/bin/
COPY --from=builder /app/openapi.yaml /etc/namelens/
EXPOSE 8080
ENTRYPOINT ["namelens", "serve", "--bind=0.0.0.0:8080"]
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
      - NAMELENS_NAMELENS_LOG_LEVEL=info
      - NAMELENS_NAMELENS_CONTROL_PLANE_API_KEY=${NAMELENS_API_KEY}
      - NAMELENS_NAMELENS_DB_PATH=/data/namelens.db
    volumes:
      - namelens-data:/data

volumes:
  namelens-data:
```

### Run in Docker

```bash
docker-compose up -d

curl -X POST http://localhost:8080/v1/check \
  -H "Content-Type: application/json" \
  -H "X-API-Key: ${NAMELENS_API_KEY}" \
  -d '{"name": "acmecorp", "profile": "startup"}'
```

---

## Environment Configuration

### Development

```bash
export NAMELENS_NAMELENS_LOG_LEVEL=debug
export NAMELENS_NAMELENS_DB_PATH=./dev.db

namelens serve
```

### Production

```bash
export NAMELENS_NAMELENS_LOG_LEVEL=info
export NAMELENS_NAMELENS_DB_PATH=/var/lib/namelens/namelens.db
export NAMELENS_NAMELENS_CONTROL_PLANE_API_KEY=nlcp_your_secret_key

# Use Turso for shared state
export NAMELENS_NAMELENS_DB_URL=libsql://your-db.turso.io
export NAMELENS_NAMELENS_DB_AUTH_TOKEN=your-auth-token

namelens serve --bind=0.0.0.0:8080
```

---

## Best Practices

1. **Use API keys** - Always configure API key authentication for non-localhost
   deployments

2. **Use reverse proxy** - For production, put namelens behind nginx, caddy, or
   cloudflared for TLS and rate limiting

3. **Cache appropriately** - Namelens caches results by default; respect cache
   TTL for rate limit compliance

4. **Handle errors** - Implement retry logic with exponential backoff for
   transient failures

5. **Version pinning** - Pin to specific Namelens versions in CI/CD

6. **Monitor health** - Use `/health` endpoint for readiness/liveness probes

---

## Need Help?

```bash
namelens serve --help
namelens doctor
```

See also:

- [Configuration](configuration.md) - Profiles and env vars
- [Quick Availability Check](quick-start.md) - Basic usage
- [Startup Naming Guide](startup-guide.md) - Full naming workflow
