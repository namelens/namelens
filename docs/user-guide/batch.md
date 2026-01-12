# Batch Processing

Check multiple names efficiently for comparison and decision-making.

## Basic Batch Check

Create a file with one name per line:

```bash
cat > candidates.txt <<EOF
acmecorp
stellaplex
fluxio
codepulse
devspark
EOF

# Check all names
namelens batch candidates.txt
```

## Output Formats

### Table (Default)

Side-by-side comparison in ASCII table:

```bash
namelens batch candidates.txt --output-format=table
```

Output example:

```
╭─────────────┬─────────┬─────────┬──────────────────────────────╮
│ NAME        │ AVAIL   │ TOTAL   │ BREAKDOWN                    │
├─────────────┼─────────┼─────────┼──────────────────────────────┤
│ acmecorp    │ 4/8     │ 8       │ .com: taken, .io: avail      │
│ stellaplex  │ 6/8     │ 8       │ .com: avail, .io: avail      │
│ fluxio      │ 2/8     │ 8       │ .com: taken, .io: taken      │
╰─────────────┴─────────┴─────────┴──────────────────────────────╯
```

### JSON

Machine-readable output for scripts and reporting:

```bash
namelens batch candidates.txt --output-format=json --out results.json
```

JSON structure:

```json
[
  {
    "name": "acmecorp",
    "score": 4,
    "total": 8,
    "results": [
      {
        "name": "acmecorp",
        "check_type": "domain",
        "tld": "com",
        "available": false,
        "extra_data": {"expires": "2026-11-17"},
        "provenance": {"...": "..."}
      }
    ]
  }
]
```

### Markdown

Formatted for presentations and AI chat:

```bash
namelens batch candidates.txt --output-format=markdown --out comparison.md
```

## Filtering

### Available Only

Show only candidates with reasonable availability:

```bash
namelens batch candidates.txt --available-only
```

### Minimum Score Threshold

Only show names meeting your minimum availability score:

```bash
namelens batch candidates.txt --min-score=5
```

Example: In an 8-check profile, only show names with 5+ available.

## With Profiles

Apply a check profile to all candidates:

```bash
namelens batch candidates.txt --profile=startup
```

Common profiles:

| Profile   | Checks                           | Use Case     |
| --------- | -------------------------------- | ------------ |
| `startup` | 3 TLDs + 2 registries + 1 handle | New ventures |
| `minimal` | .com only                        | Quick scans  |
| `web3`    | .xyz, .io, .gg + npm + github    | Web3/crypto  |

## Custom Checks

Specify custom TLDs, registries, or handles:

```bash
namelens batch candidates.txt --tlds=com,io,net --registries=npm --handles=github
```

## Workflow: Candidate Comparison

### Step 1: Generate Long List

Start with 10-20 candidates from brainstorming or AI generation:

```bash
namelens generate "AI productivity tool" > candidates.txt
```

### Step 2: Quick Batch Scan

Initial availability check:

```bash
namelens batch candidates.txt --profile=minimal --available-only
```

This eliminates obvious conflicts quickly.

### Step 3: Deep Batch Analysis

For remaining 3-5 finalists, run deep analysis:

```bash
namelens batch finalists.txt \
  --profile=startup \
  --expert \
  --phonetics \
  --suitability \
  --output-format=json --out deep-analysis.json
```

### Step 4: Create Comparison Table

Extract key metrics from JSON:

```bash
jq -r '.results[] | "\(.name): \(.score)/\(.total) available"' deep-analysis.json
```

Output:

```
acmecorp: 6/8 available
stellaplex: 7/8 available
fluxio: 4/8 available
```

### Step 5: Stakeholder Presentation

Use markdown output for presentations:

```bash
namelens batch finalists.txt \
  --profile=startup \
  --expert \
  --output-format=markdown --out candidate-comparison.md
```

Include in slide decks or share via team chat.

## Workflow: Competitor Analysis

### Step 1: List Competitors

```bash
cat > competitors.txt <<EOF
competitor-a
competitor-b
competitor-c
EOF
```

### Step 2: Batch Check with Expert Analysis

```bash
namelens batch competitors.txt \
  --profile=startup \
  --expert \
  --expert-prompt=domain-content \
  --output-format=json --out competitor-analysis.json
```

### Step 3: Analyze Patterns

Extract insights:

- Which competitors own multiple TLDs?
- Which have strong social presence?
- Which domains are parked vs active?

## Integration Examples

### CI/CD: Prevent Breaking Changes

If you're checking that internal project names don't conflict with public
assets:

```bash
#!/bin/bash
# check-names.sh

namelens batch project-names.txt --output-format=json | \
  jq -e '.results[] | select(.score < 5) | .name' && \
  echo "ERROR: Some projects have availability issues" && \
  exit 1

echo "All projects clear of conflicts"
```

### Automation: Weekly Portfolio Check

Check your brand portfolio weekly:

```bash
#!/bin/bash
# weekly-check.sh

DATE=$(date +%Y-%m-%d)
OUTPUT_DIR="brand-audits/$DATE"
mkdir -p "$OUTPUT_DIR"

namelens batch brand-portfolio.txt \
  --profile=startup \
  --expert \
  --output-format=json --out "$OUTPUT_DIR/portfolio.json"

namelens batch brand-portfolio.txt \
  --profile=startup \
  --expert \
  --output-format=markdown --out "$OUTPUT_DIR/portfolio.md"

# Send notification if issues found
ISSUES=$(jq '[.results[] | select(.score < 5)] | length' "$OUTPUT_DIR/portfolio.json")
if [ "$ISSUES" -gt 0 ]; then
  echo "Found $ISSUES names with availability issues" | \
    mail -s "Brand Portfolio Alert" team@example.com
fi
```

### Spreadsheet Export

Convert JSON to CSV for Excel/Google Sheets:

```bash
namelens batch candidates.txt --output-format=json | \
  jq -r '.results[] | [.name, .score, .total] | @csv' > candidates.csv
```

Open in spreadsheet for visual comparison and filtering.

## Tips

1. **Use meaningful filenames** — `candidates-round1.txt`, `finalists-v2.txt`
   for version control
2. **Store JSON for audit trails** — Keep batch outputs for reference and
   compliance
3. **Profile first, customize later** — Start with profiles, then add custom
   flags if needed
4. **Pipe to `jq` for filtering** — Extract specific data without writing
   scripts
5. **Use `--available-only` for quick scans** — Filter noise when you just need
   viable options
6. **Batch expert sparingly** — Expert analysis is slower; use on 3-5 finalists,
   not 20 candidates

## Performance

| Scenario                        | Estimated Time |
| ------------------------------- | -------------- |
| 10 candidates, minimal profile  | 5-10 seconds   |
| 10 candidates, startup profile  | 15-30 seconds  |
| 5 candidates, startup + expert  | 60-120 seconds |
| 20 candidates, startup + expert | 4-8 minutes    |

Expert analysis is slower due to AI search queries. Batch large lists without
`--expert` first to filter.

## Need Help?

```bash
namelens batch --help
namelens doctor  # Troubleshooting
```

See also:

- [Quick Availability Check](quick-start.md) — Single-name checks
- [Startup Naming Guide](startup-guide.md) — Full naming workflow
- [Expert Analysis](expert-search.md) — Deep analysis options
