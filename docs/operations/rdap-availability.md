# RDAP Availability Detection

This note explains how NameLens determines whether RDAP is available for a TLD.

## Source of Truth

NameLens uses the IANA RDAP bootstrap registry:

- URL: `https://data.iana.org/rdap/dns.json`
- Format: mapping of TLDs to RDAP server URLs

This registry is the authoritative source for which TLDs publish RDAP endpoints.

## How NameLens Uses It

1. `namelens bootstrap update` downloads `dns.json`.
2. The bootstrap service stores the TLD → RDAP URL list in the database.
3. The domain checker looks up the requested TLD in the cached bootstrap data.
4. If a TLD has no RDAP URLs, NameLens treats RDAP as unavailable and falls back
   to WHOIS/DNS (if enabled).

## Refresh Cadence

The bootstrap data is cached locally. Refresh it periodically:

```bash
namelens bootstrap update
```

## Notes on ccTLDs

Many ccTLDs are not required to support RDAP. This is why popular developer TLDs
like `.io` lack RDAP and require WHOIS fallback.

## Troubleshooting

- If a TLD should have RDAP but shows as unsupported, run
  `namelens bootstrap update` and re-check.
- Use `namelens bootstrap status` to see the cached bootstrap metadata.
