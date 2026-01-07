# Expert Search Guide

> **Important**: Expert search results are AI-generated and provided for
> informational purposes only. They do not constitute legal or professional
> advice. See [USAGE-NOTICE.md](../USAGE-NOTICE.md) for full disclaimer.

Expert search uses AI to analyze brand name availability beyond simple
domain/registry checks.

## Overview

The `--expert` flag triggers an AI-powered analysis that searches:

- **X (Twitter)** - Handles, mentions, existing projects
- **Web** - Trademarks, startups, news, domains
- **Sentiment** - Positive/negative associations

## Enabling Expert Search

### Via Environment Variables

```bash
export NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=your-api-key
```

### Via Configuration File

```yaml
ailink:
  default_provider: namelens-xai
  providers:
    namelens-xai:
      enabled: true
      ai_provider: xai
      base_url: https://api.x.ai/v1
      models:
        default: grok-4-1-fast-reasoning
      selection_policy: priority
      credentials:
        - label: default
          priority: 0
          api_key: your-api-key # Better to use env var

expert:
  enabled: true
  default_prompt: name-availability
  role: name-availability
```

## Usage

```bash
# Basic expert check
namelens check myproject --expert

# With specific TLDs
namelens check myproject --tlds=com,io --expert

# JSON output with full analysis
namelens check myproject --expert --output=json
```

## Output Format

### Text Output

```
│ expert │ ailink │ risk: low │ No direct conflicts found; name appears available │
```

### JSON Output

```json
{
  "ailink": {
    "summary": "No direct conflicts found",
    "likely_available": true,
    "risk_level": "low",
    "insights": [
      "No existing projects with this name on X",
      "No trademark conflicts found"
    ],
    "mentions": [
      {
        "source": "X",
        "description": "No exact matches found",
        "relevance": "low",
        "sentiment": "neutral"
      }
    ],
    "recommendations": [
      "Proceed confidently",
      "Secure social handles immediately"
    ]
  }
}
```

## Risk Levels

| Level      | Meaning                                          |
| ---------- | ------------------------------------------------ |
| `low`      | No significant conflicts found                   |
| `medium`   | Some partial matches or similar names exist      |
| `high`     | Active projects or trademarks with similar names |
| `critical` | Direct conflicts that would prevent use          |

## Supported Providers

Currently supported:

| Provider    | Base URL              | Models                              |
| ----------- | --------------------- | ----------------------------------- |
| x.ai (Grok) | `https://api.x.ai/v1` | `grok-4-1-fast-reasoning`, `grok-4` |

The driver uses OpenAI-compatible API format with x.ai's `search_parameters`
extension for live search.

## Prompts

Expert search uses configurable prompts located in:

```
internal/ailink/prompt/prompts/name-availability.md
```

Custom prompts can be placed in a directory specified by
`NAMELENS_AILINK_PROMPTS_DIR`.

### Prompt Slugs

Available prompt slugs:

- `name-availability` (default) - comprehensive availability analysis with
  real-time search
- `brand-proposal` - generate a branding proposal for a candidate name
- `brand-plan` - generate a detailed branding and launch plan
- `domain-content` - analyze what's on a taken domain (parked, placeholder, or
  active product)
- `name-alternatives` - generate naming candidates from a product concept (used
  by `generate` command)
- `name-phonetics` - analyze pronunciation, typeability, and CLI suitability
- `name-suitability` - analyze cultural appropriateness across locales

Example usage:

```bash
# Generate a branding proposal
namelens check namelens --expert --expert-prompt=brand-proposal --expert-depth=deep

# Analyze domain content for conflict assessment
namelens check namelens --tlds=com --expert --expert-prompt=domain-content
```

## Phonetics and Suitability Analysis

The `--phonetics` and `--suitability` flags provide specialized analysis for
naming decisions.

### Phonetics Analysis

Analyzes pronunciation, typeability, and CLI-friendliness:

```bash
namelens check myproject --phonetics
```

Output includes:

- **Syllable breakdown** - phonetic structure
- **Typeability score** - how easy to type (0-100)
- **CLI suitability** - command-line friendliness (0-100)
- **Potential issues** - consonant clusters, ambiguous spellings

### Suitability Analysis

Analyzes cultural appropriateness across target locales:

```bash
namelens check myproject --suitability --locales=en-US,de-DE,ja-JP
```

Output includes:

- **Overall score** - cultural appropriateness (0-100)
- **Risk categories** - offensive, religious, political, legal
- **Locale-specific concerns** - per-locale analysis

### Combined Analysis

```bash
# Full analysis with phonetics and suitability
namelens check myproject --phonetics --suitability --locales=en-US,es-ES,zh-CN
```

## Rate Limiting and Costs

- x.ai Agent Tools API is currently free (as of Dec 2025)
- Live Search costs $25 per 1,000 sources
- Results are cached for 24 hours by default

## Troubleshooting

### "expert api key not configured"

Set the API key:

```bash
export NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=your-key
```

### "expert request failed"

Check:

1. API key is valid
2. Network connectivity to api.x.ai
3. Model name is correct (`grok-4-1-fast-reasoning` recommended)

### Slow response times

Expert search can take 10-30 seconds as the AI performs multiple searches.
Increase timeout if needed:

```yaml
expert:
  timeout: 90s
```
