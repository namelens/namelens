# Release Notes

This file keeps notes for the latest three releases in reverse chronological
order.

## Unreleased

## v0.1.4 (2026-01-24)

Brand mark generation and image pipeline.

Highlights:

- **Brand mark command**: New `namelens mark` generates logo/mark directions and
  images for finalist names
- **Thumbnail utility**: New `namelens image thumb` creates agent-friendly
  thumbnails for sharing and AI workflows
- **Color control**: `--color` flag with `monochrome`, `brand`, and `vibrant`
  modes for consistent brand palettes
- **GPT Image support**: OpenAI GPT Image models recommended for best results

### Brand Mark Generation

Generate early-stage logo directions for your finalist names:

```bash
# Basic mark generation
namelens mark "myproject" --out-dir ./marks --count 3

# With context, color mode and transparent background
namelens mark "myproject" --out-dir ./marks \
  --description "Static analysis tool for Go" \
  --audience "developers" \
  --color brand \
  --background transparent \
  --format png \
  --quality high
```

Output:

```
./marks/
├── myproject_brand-mark_01.png
├── myproject_brand-mark_02.png
└── myproject_brand-mark_03.png
```

### Color Modes

Control the color palette of generated marks:

| Mode         | Description                                     |
| ------------ | ----------------------------------------------- |
| `brand`      | Modern tech palette (teal, navy, coral accents) |
| `monochrome` | Black, white, and greys only                    |
| `vibrant`    | Bold, saturated startup colors                  |

```bash
# Tech-forward brand colors (recommended)
namelens mark "myproject" --out-dir ./marks --color brand

# Classic monochrome
namelens mark "myproject" --out-dir ./marks --color monochrome
```

### Thumbnail Generation

Create smaller versions for sharing and AI agent ingestion:

```bash
# Generate JPEG thumbnails (256px max)
namelens image thumb --in-dir ./marks --max-size 256 --format jpeg

# Custom size and output location
namelens image thumb --in-dir ./marks --out-dir ./thumbs --max-size 128
```

Output:

```
./marks/
├── myproject_brand-mark_01.png
├── myproject_brand-mark_01.thumbnail.jpg
├── myproject_brand-mark_02.png
├── myproject_brand-mark_02.thumbnail.jpg
└── ...
```

### Configuration

Route text and image generation to separate providers:

```bash
# .env configuration
NAMELENS_AILINK_ROUTING_BRAND_MARK=namelens-openai
NAMELENS_AILINK_ROUTING_BRAND_MARK_IMAGE=namelens-openai-image

# Dedicated image provider
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_IMAGE_ENABLED=true
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_IMAGE_AI_PROVIDER=openai
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_IMAGE_MODELS_IMAGE=gpt-image-1.5
```

### Provider Recommendations

| Provider         | Status       | Notes                                       |
| ---------------- | ------------ | ------------------------------------------- |
| OpenAI GPT Image | Recommended  | Best instruction following, no text leakage |
| xAI grok-2-image | Experimental | No size control, photorealistic bias        |

See [known-issues.md](docs/known-issues.md) for xAI limitations.

## v0.1.3 (2026-01-14)

Candidate comparison and Rust ecosystem support.

Highlights:

- **Compare command**: New `namelens compare` for side-by-side candidate
  screening with availability, risk, phonetics, and suitability scores
- **Cargo/crates.io checker**: Rust crate availability now included in registry
  checks
- **Review quick mode**: New `--mode=quick` flag for fast screening workflows
- **Improved reliability**: .app/.dev domains now route via Google RDAP

### Compare Command

Screen 2-10 candidates before running expensive brand analysis:

```bash
# Quick availability screening
namelens compare fulgate agentanvil toolcrux --mode=quick

# Full analysis with phonetics and suitability
namelens compare fulgate toolcrux

# Export for stakeholders
namelens compare alpha beta gamma --output-format=markdown --out comparison.md
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

### Cargo/crates.io Support

Check Rust crate availability:

```bash
# Check specific registry
namelens check mycrate --registries=cargo

# Developer profile now includes cargo
namelens check myproject --profile=developer
```

### Review Quick Mode

Fast screening without full brand analysis:

```bash
namelens review myproject --mode=quick
```

## v0.1.2 (2026-01-11)

Release workflow fix.

Highlights:

- **CI fix**: Add explicit `GITHUB_TOKEN` to all `softprops/action-gh-release`
  steps in `release.yml` for reliable artifact uploads in new GitHub
  organizations
