# Domain Fallback Guide

When RDAP (Registration Data Access Protocol) is unavailable for a TLD, NameLens
can fall back to alternative lookup methods. For details on how RDAP
availability is determined, see `docs/operations/rdap-availability.md`.

## Fallback Chain

```
RDAP (primary) → Whois (fallback) → DNS (last resort)
```

| Method | Reliability       | Speed  | Rate Limits     |
| ------ | ----------------- | ------ | --------------- |
| RDAP   | Authoritative     | Fast   | Per-server      |
| Whois  | Authoritative     | Slower | 30/hour default |
| DNS    | Non-authoritative | Fast   | None            |

## TLDs Requiring Fallback

Some TLDs don't have public RDAP servers:

| TLD   | Whois Server | Notes                          |
| ----- | ------------ | ------------------------------ |
| `.io` | whois.nic.io | British Indian Ocean Territory |
| `.sh` | whois.nic.sh | Saint Helena                   |
| `.co` | whois.nic.co | Colombia                       |
| `.me` | whois.nic.me | Montenegro                     |

Some TLDs have neither RDAP nor public WHOIS servers (e.g., `.dev`, `.app`).
These return `unknown` status until RDAP support is added.

## Enabling Whois Fallback

Whois fallback is enabled by default so common TLDs without RDAP (like `.io`)
return a result instead of `unsupported`. You can disable it if you only want
authoritative RDAP checks.

### Via Environment Variables

```bash
export NAMELENS_DOMAIN_WHOIS_FALLBACK_ENABLED=false
export NAMELENS_DOMAIN_WHOIS_FALLBACK_REQUIRE_EXPLICIT=true
```

### Via Configuration File

```yaml
# ~/.config/namelens/config.yaml
domain:
  whois_fallback:
    enabled: true
    require_explicit: false
    timeout: 10s
    cache_ttl: 6h
```

## Configuration Options

### require_explicit

Controls which TLDs can use whois fallback:

| Value   | Behavior                                    |
| ------- | ------------------------------------------- |
| `false` | Allow whois for any TLD without RDAP        |
| `true`  | Only allow TLDs in the explicit `tlds` list |

### tlds

Explicit list of TLDs allowed for whois fallback (only used when
`require_explicit: true`):

```yaml
domain:
  whois_fallback:
    enabled: true
    require_explicit: true
    tlds:
      - io
      - sh
      - co
```

### Custom Whois Servers

Override default whois servers for specific TLDs:

```yaml
domain:
  whois_fallback:
    servers:
      io: whois.nic.io
      sh: whois.nic.sh
```

## Rate Limiting

Queries are rate-limited to avoid abuse. Each endpoint type has its own window:

| Endpoint Type | Default | Window     | Example Keys                               |
| ------------- | ------- | ---------- | ------------------------------------------ |
| RDAP servers  | 30/min  | Per minute | `rdap.verisign.com`, `rdap.nic.io`         |
| Whois servers | 30/hour | Per hour   | `whois.whois.nic.io`, `whois.whois.nic.sh` |

Override defaults via `rate_limits.<endpoint>` where keys match actual server
hostnames. Overrides are per minute even when defaults use longer windows:

```yaml
rate_limits:
  rdap.verisign.com: 60 # per-minute override
  whois.whois.nic.io: 1 # per-minute override (default is 30/hour)
```

When rate-limited, results show `rate limited` status with retry time.

## DNS Fallback

DNS fallback is a last resort when both RDAP and whois are unavailable:

```yaml
domain:
  dns_fallback:
    enabled: true
    timeout: 5s
    cache_ttl: 30m
```

DNS results are **non-authoritative** - they indicate whether DNS records exist
but cannot confirm registration status.

| DNS Result        | Meaning                                  |
| ----------------- | ---------------------------------------- |
| `nxdomain`        | Domain likely available (no DNS records) |
| `records_present` | Domain likely taken (has DNS records)    |

## Availability States

NameLens uses tri-state availability reporting:

| Status      | Meaning                                              |
| ----------- | ---------------------------------------------------- |
| `available` | Domain confirmed available via RDAP/WHOIS            |
| `taken`     | Domain confirmed registered                          |
| `unknown`   | No authoritative data (no RDAP/WHOIS server for TLD) |

The summary excludes `unknown` results from the denominator: "6/6 available, 2
unknown" means 6 of 6 checkable domains are available, plus 2 could not be
verified.

## Example Output

```bash
$ namelens check myproject --tlds=com,io,sh

│ domain │ myproject.com │ taken     │ exp: 2026-01-15; registrar: GoDaddy │
│ domain │ myproject.io  │ taken     │ source: whois                       │
│ domain │ myproject.sh  │ available │ source: whois                       │
```

## Troubleshooting

### "unsupported" status for .io/.sh

If you disabled whois fallback, re-enable it:

```bash
export NAMELENS_DOMAIN_WHOIS_FALLBACK_ENABLED=true
export NAMELENS_DOMAIN_WHOIS_FALLBACK_REQUIRE_EXPLICIT=false
```

### "unknown" status for .dev/.app

These TLDs don't have public WHOIS servers. RDAP support is planned for a future
release.

### "rate limited" status

You've exceeded the rate limit for an endpoint. Wait or increase the limit:

```yaml
rate_limits:
  whois.whois.nic.io: 1 # per-minute override (default is 30/hour)
```

### Inconsistent results

Whois parsing relies on pattern matching. Results are cached for 6 hours by
default. Clear cache or wait for expiry.
