# Compare Command

Side-by-side comparison of candidate names for quick screening before deep
analysis.

> **Note**: Example names in this guide (fulgate, toolcrux, acmecorp, etc.) are
> for illustration only and do not represent trademark claims.

---

## When to Use Compare

Use `compare` when you have 2-10 candidate names and need to quickly filter them
before running expensive brand analysis with `review`.

| Scenario                        | Use Compare?                        |
| ------------------------------- | ----------------------------------- |
| Generated 8 names, need top 3-4 | Yes - use `--mode=quick` to screen  |
| Narrowed to 3 finalists         | Yes - full mode for phonetics/suit. |
| Have 50+ names to check         | No - use `batch` first              |
| Need trademark/brand deep dive  | No - use `review` after compare     |

**Typical workflow**: `generate` -> `compare --mode=quick` -> `compare` (full)
-> `review`

---

## Quick Mode (Screening)

Fast availability-only comparison for initial screening:

```bash
namelens compare fulgate agentanvil fulnexus toolcrux --mode=quick
```

Output:

```
╭────────────┬──────────────┬────────╮
│ NAME       │ AVAILABILITY │ LENGTH │
├────────────┼──────────────┼────────┤
│ fulgate    │ 7/7          │      7 │
│ agentanvil │ 7/7          │     10 │
│ fulnexus   │ 7/7          │      8 │
│ toolcrux   │ 7/7          │      8 │
╰────────────┴──────────────┴────────╯
```

Quick mode skips AI-powered phonetics and suitability analysis, completing in
~10 seconds for 4-5 names.

---

## Full Mode (Analysis)

Include phonetics and cultural suitability scores:

```bash
namelens compare fulgate toolcrux
```

Output:

```
╭──────────┬──────────────┬──────┬───────────┬─────────────┬────────╮
│ NAME     │ AVAILABILITY │ RISK │ PHONETICS │ SUITABILITY │ LENGTH │
├──────────┼──────────────┼──────┼───────────┼─────────────┼────────┤
│ fulgate  │ 7/7          │ low  │ 83        │ 95          │      7 │
│ toolcrux │ 7/7          │ low  │ 81        │ 95          │      8 │
╰──────────┴──────────────┴──────┴───────────┴─────────────┴────────╯
```

**Columns explained**:

| Column       | Description                                        |
| ------------ | -------------------------------------------------- |
| Availability | Available checks / Total checks (based on profile) |
| Risk         | low/medium/high - derived from .com and key assets |
| Phonetics    | Combined score for pronunciation and typeability   |
| Suitability  | Cultural appropriateness across markets            |
| Length       | Character count (shorter = better for CLI tools)   |

Full mode takes ~30-60 seconds for 2-4 names due to AI analysis.

---

## Output Formats

### Table (Default)

```bash
namelens compare fulgate toolcrux --output-format=table
```

### JSON

Machine-readable for automation and scripts:

```bash
namelens compare fulgate toolcrux --output-format=json
```

```json
[
  {
    "name": "fulgate",
    "length": 7,
    "availability": {
      "score": 7,
      "total": 7,
      "unknown": 0
    },
    "risk_level": "low",
    "phonetics": {
      "overall_score": 83,
      "typeability_score": 82,
      "cli_suitability": 90
    },
    "suitability": {
      "overall_score": 95,
      "rating": "suitable"
    }
  }
]
```

### Markdown

For documentation and stakeholder presentations:

```bash
namelens compare fulgate toolcrux --mode=quick --output-format=markdown
```

```markdown
| Name     | Availability | Length |
| -------- | ------------ | ------ |
| fulgate  | 7/7          | 7      |
| toolcrux | 7/7          | 8      |
```

---

## Flags Reference

| Flag              | Default   | Description                                |
| ----------------- | --------- | ------------------------------------------ |
| `--mode`          | (full)    | `quick` for availability only              |
| `--profile`       | `startup` | Availability profile (domains, registries) |
| `--output-format` | `table`   | Output format: table, json, markdown       |
| `--out`           | stdout    | Write output to file                       |
| `--no-cache`      | false     | Skip cache, force fresh lookups            |

---

## Compare vs Batch

| Feature               | `compare`             | `batch`                |
| --------------------- | --------------------- | ---------------------- |
| Input                 | CLI args (2-20 names) | File (unlimited names) |
| Phonetics/Suitability | Yes (full mode)       | No                     |
| Risk scoring          | Yes                   | No                     |
| Best for              | Finalist screening    | Initial bulk filtering |
| Speed                 | ~10s quick, ~60s full | ~2s per name           |

**Workflow**: Use `batch` for 20+ candidates, then `compare` for top 5-10.

---

## Examples

### Screen generated candidates

```bash
# Generate ideas
namelens generate "agent gateway for MCP tools" --out candidates.txt

# Quick screen the top candidates
namelens compare agentgate toolbridge mcprelay fulgate --mode=quick

# Full analysis on survivors
namelens compare fulgate toolbridge --output-format=markdown --out finalists.md
```

### Compare with different profiles

```bash
# Startup profile (domains + npm + pypi + github)
namelens compare myapp yourapp --profile=startup

# Minimal profile (.com only)
namelens compare myapp yourapp --profile=minimal
```

### Export for stakeholders

```bash
namelens compare alpha beta gamma \
  --output-format=markdown \
  --out naming-comparison.md
```

---

## See Also

- [Batch Processing](batch.md) - Bulk checking from files
- [Startup Guide](startup-guide.md) - End-to-end naming workflow
- [Expert Analysis](expert-search.md) - Deep brand research with `review`
