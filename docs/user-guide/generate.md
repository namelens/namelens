# Name Generation Guide

> **Important**: Generated name suggestions are AI-powered and provided for
> informational purposes only. They do not constitute legal or professional
> advice. See [USAGE-NOTICE.md](../USAGE-NOTICE.md) for full disclaimer.

The `generate` command creates naming candidates from a product concept using
AI.

## Overview

While `check` validates a specific name, `generate` brainstorms candidates from
a concept description. This enables the full naming workflow:

1. **Generate** - brainstorm candidates from concept/description
2. **Check** - validate availability of promising candidates
3. **Expert analysis** - deep-dive on finalists

## Prerequisites

Requires an AI backend with API key configured. OpenAI is recommended for
generate due to fast, reliable structured outputs:

```bash
# OpenAI (recommended for generate)
export NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_CREDENTIALS_0_API_KEY=your-api-key
export NAMELENS_AILINK_DEFAULT_PROVIDER=namelens-openai

# Or xAI/Grok
export NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=your-api-key
```

See [Configuration](configuration.md) for full provider setup.

## Usage

### Basic Usage

```bash
# Generate names from a concept
namelens generate "static analyzer for shell scripts"
```

### With Context

```bash
# With current working name
namelens generate "static analyzer for shell scripts" --current-name shellsafe

# With tagline
namelens generate "static analyzer for shell scripts" --tagline "The pause before the pipe"

# With inline description
namelens generate "static analyzer for shell scripts" \
  --description "Lightweight CLI that assesses shell scripts for risky patterns"

# With description from file
namelens generate "static analyzer for shell scripts" --description-file ./README.md

# With pre-generated corpus (recommended for complex projects)
namelens context ./my-project > corpus.json
namelens generate "static analyzer for shell scripts" --corpus=corpus.json

# Pipeline: scan and generate in one step
namelens context ./my-project | namelens generate "my product" --corpus=-

# Quick scan (deprecated, use context command instead)
namelens generate "my product" --scan-dir=./my-project
```

### With Constraints

```bash
# Specify naming constraints
namelens generate "static analyzer for shell scripts" \
  --constraints "must be CLI-friendly, avoid 'safe' suffix due to go-shellsafe conflict"
```

### Generation Depth

```bash
# Quick generation (default) - uses default model
namelens generate "agent gateway" --depth quick

# Fast generation - uses fast model tier (gpt-4o-mini for OpenAI)
namelens generate "agent gateway" --depth fast

# Deep generation - uses reasoning model (gpt-5.1 for OpenAI)
namelens generate "agent gateway" --depth deep
```

### Output Formats

```bash
# Human-readable output (default)
namelens generate "process utilities library"

# JSON output for scripting
namelens generate "process utilities library" --json
```

## Flags

| Flag                 | Short | Type   | Description                                          |
| -------------------- | ----- | ------ | ---------------------------------------------------- |
| `--current-name`     | `-n`  | string | Current working name seeking alternatives            |
| `--tagline`          | `-t`  | string | Product tagline/slogan                               |
| `--description`      | `-d`  | string | Inline product description                           |
| `--description-file` | `-f`  | path   | Read description from file (truncated to 2000 chars) |
| `--corpus`           |       | path   | Use pre-generated corpus file (JSON/markdown, `-` for stdin) |
| `--scan-dir`         | `-s`  | path   | Scan directory for context files (prefer `--corpus`) |
| `--scan-budget`      |       | int    | Max chars from scanned files (default: 32000)        |
| `--constraints`      | `-c`  | string | Naming constraints/requirements                      |
| `--depth`            |       | string | `quick` (default), `fast`, or `deep`                 |
| `--json`             |       | bool   | Output raw JSON response                             |
| `--model`            |       | string | Model override                                       |
| `--prompt`           |       | string | Prompt slug (default: `name-alternatives`)           |

## Output Format

### Human-Readable

```
Generating name alternatives for: static analyzer for shell scripts

Concept Analysis:
  Core function: Pre-execution static analysis of shell scripts
  Key themes: safety, vetting, inspection, pre-run
  Target audience: DevOps, security engineers, CI/CD pipelines

Top Recommendations:
  1. scriptvet - "Vet your scripts before running" - Strong fit, clean availability
  2. pipesafe - Similar to shellsafe without go-shellsafe confusion
  3. runvet - "Vet before you run" - Action-oriented

All Candidates:
  NAME           STRATEGY     STRENGTH   CONFLICTS
  scriptvet      compound     strong     None found
  pipesafe       compound     strong     None found
  runvet         compound     moderate   None found
  shaudit        compound     moderate   PyPI: security headers tool
  prerun         descriptive  weak       PyPI: prerun package

Themes explored: safety, vetting, shell, script, run, guard

Run 'namelens check <name>' to verify availability.
```

### JSON

Raw response from the AI backend, conforming to the `name-alternatives` prompt
schema.

## Workflow Examples

### Generate Then Check

```bash
# Generate candidates
namelens generate "shell script analyzer" --json | jq -r '.top_recommendations[].name'

# Check top candidates
namelens generate "shell script analyzer" --json | \
  jq -r '.top_recommendations[].name' | \
  xargs -I {} namelens check {}
```

### Full Analysis Pipeline

```bash
# 1. Generate candidates
namelens generate "agent gateway for AI services" \
  --constraints "short, memorable, tech-forward" \
  --depth deep

# 2. Check availability of favorites
namelens check fulsigil --expert

# 3. Analyze phonetics and suitability
namelens check fulsigil --phonetics --suitability --locales=en-US,de-DE,ja-JP
```

## Troubleshooting

### "model not configured"

Set the model via environment or flag:

```bash
# For OpenAI
export NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_MODELS_DEFAULT=gpt-4o

# For xAI
export NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_MODELS_DEFAULT=grok-4-1-fast-reasoning

# Or override per-command
namelens generate "my concept" --model gpt-4o
```

### "API key not configured"

Set the API key for your provider:

```bash
# For OpenAI
export NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_CREDENTIALS_0_API_KEY=your-key

# For xAI
export NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=your-key
```

### Slow Response

OpenAI typically responds in 1-5 seconds. xAI can take 10-30 seconds due to web
search.

Use `--depth quick` for faster results:

```bash
namelens generate "my concept" --depth quick
```

Use `--depth fast` with OpenAI for fastest results (uses gpt-4o-mini):

```bash
namelens generate "my concept" --depth fast
```

## Related

- [Context](context.md) - Generate and inspect context corpus
- [Expert Search](expert-search.md) - AI-powered availability analysis
- [Configuration](configuration.md) - Environment and config file setup
