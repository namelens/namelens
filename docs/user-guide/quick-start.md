# Quick Availability Check

The fastest way to check if a name is available across domains, registries, and
social handles.

## Basic Check

```bash
namelens check myproject
```

By default, this checks `.com` only.

Use `--profile=startup` for a broader scan across `.com`, `.io`, `.dev`, `.app`
plus npm/PyPI and GitHub.

## Check Specific TLDs

```bash
namelens check myproject --tlds=com,io,sh,net
```

## Check with Profile

Profiles bundle common checks together:

```bash
namelens check acmecorp --profile=startup
```

Available profiles:

| Profile     | Domains                           | Registries       | Handles |
| ----------- | --------------------------------- | ---------------- | ------- |
| `startup`   | .com, .io, .dev, .app             | npm, pypi        | github  |
| `developer` | .com, .io, .dev, .app, .sh, .org, .net | npm, pypi, cargo | github  |
| `minimal`   | .com                              | -                | -       |
| `website`   | .com, .org, .net                  | -                | -       |
| `web3`      | .xyz, .io, .gg                    | npm              | github  |

## Output Formats

**Default (table):**

```
╭────────┬──────────┬──────────┬────────────────────────────╮
│ TYPE   │ NAME     │ STATUS   │ NOTES                      │
├────────┼──────────┼──────────┼────────────────────────────┤
│ domain │ myproject.com │ taken │ exp: 2026-06-15           │
│ domain │ myproject.io  │ avail │                           │
│ npm    │ myproject     │ avail │                           │
╰────────┴──────────┴──────────┴────────────────────────────╯
```

**JSON:**

```bash
namelens check myproject --output-format=json
```

**Markdown (for AI chat pasting):**

```bash
namelens check myproject --output-format=markdown
```

## Social Handle Check

```bash
namelens check myproject --handles=github
```

More social platforms coming in future releases.

## Check Package Registries Only

```bash
namelens check myproject --registries=npm,pypi,cargo
```

## Multiple Names (Batch)

```bash
# Create a file with one name per line
echo -e "acmecorp\nstellaplex\nfluxio" > candidates.txt

# Batch check
namelens batch candidates.txt

# With specific output
namelens batch candidates.txt --output-format=table
namelens batch candidates.txt --output-format=json --out results.json
```

## CI/CD Integration

Add name availability checks to your pipeline:

```bash
# In CI: Fail if key domains are taken
namelens check myproject --tlds=com,io --output-format=json | \
  jq -e '.results[] | select(.check_type == "domain" and .tld == "com" and .available == false)' && \
  echo "ERROR: .com domain not available" && exit 1
```

## Tips

- **Check early, check often** — Availability changes fast; check before
  investing
- **Batch your candidates** — Use `batch` to compare 3-5 names side-by-side
- **Profile for your stage** — `minimal` for quick checks, `startup` for deeper
- **Cache is your friend** — Results are cached; re-running is fast

## What's Not Included

The quick check does **not** include:

- Trademark searches (use `--expert`)
- Phonetic or cultural analysis (use `--phonetics --suitability`)
- Social sentiment analysis (use `--expert`)

**About expert features**: Namelens uses direct HTTP connections to AI providers
(no SDKs or intermediaries), giving you full transparency into the request/
response pipeline. Enable debug logging to inspect:

```bash
NAMELENS_LOG_LEVEL=debug namelens check myproject --expert
```

For projects you're serious about, see [Expert Analysis](expert-search.md).
