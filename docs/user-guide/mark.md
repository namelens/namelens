# Brand Mark Generation

Generate early-stage logo mark concepts and images for your finalist names.

---

## Overview

The `namelens mark` command generates brand mark directions and images for names
you're serious about. Use it after narrowing candidates with `check`, `compare`,
or `review`.

For best results, provide a short description and audience so the marks reflect
what you’re building (not just the name string).

**What you get:**

- 3+ distinct logo mark directions
- AI-generated images for each direction
- Configurable color palettes
- Transparent background support for compositing

**When to use:**

- After selecting 1-3 finalist names
- Before engaging a designer (get directional concepts first)
- For internal presentations and stakeholder alignment
- Quick visual exploration of brand directions

---

## Quick Start

```bash
# Basic: Generate 3 marks for your name
namelens mark "myproject" --out-dir ./marks

# Recommended: With context, color mode, and transparent background
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

---

## Color Modes

Control the color palette with `--color`:

| Mode         | Description                             | Best For                      |
| ------------ | --------------------------------------- | ----------------------------- |
| `brand`      | Modern tech palette (teal, navy, coral) | SaaS, dev tools, startups     |
| `monochrome` | Black, white, and greys                 | Classic brands, print-first   |
| `vibrant`    | Bold, saturated colors                  | Consumer apps, creative tools |
| _(default)_  | Tech-forward palette                    | General use                   |

```bash
# Tech-forward brand colors (recommended for most projects)
namelens mark "myproject" --out-dir ./marks --color brand

# Classic monochrome for versatility
namelens mark "myproject" --out-dir ./marks --color monochrome

# Bold colors for consumer-facing products
namelens mark "myproject" --out-dir ./marks --color vibrant
```

---

## Output Formats

### Image Format

Choose output format with `--format`:

| Format | Transparency | File Size | Best For            |
| ------ | ------------ | --------- | ------------------- |
| `png`  | Yes          | Larger    | Print, design tools |
| `webp` | Yes          | Smaller   | Web, sharing        |
| `jpeg` | No           | Smallest  | Quick previews      |

```bash
# PNG for design work
namelens mark "myproject" --out-dir ./marks --format png

# WebP for smaller files
namelens mark "myproject" --out-dir ./marks --format webp
```

### Background

Control background with `--background`:

- `transparent` — For compositing onto other backgrounds
- `opaque` — Solid background (model decides color)
- `auto` — Let the model decide (default)

```bash
# Transparent for flexibility
namelens mark "myproject" --out-dir ./marks --background transparent
```

---

## Generating Thumbnails

Create smaller versions for sharing, presentations, or AI agent workflows:

```bash
# Generate JPEG thumbnails (256px max dimension)
namelens image thumb --in-dir ./marks --max-size 256 --format jpeg

# Custom size
namelens image thumb --in-dir ./marks --max-size 128

# Output to separate directory
namelens image thumb --in-dir ./marks --out-dir ./thumbs
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

Thumbnails are useful for:

- Slack/Discord sharing
- Email attachments
- AI agent context (smaller token cost)
- Quick stakeholder reviews

---

## Configuration

### Provider Setup

For best results, configure a dedicated image provider:

```bash
# .env

# Text generation (mark directions)
NAMELENS_AILINK_ROUTING_BRAND_MARK=namelens-openai

# Image generation (separate provider recommended)
NAMELENS_AILINK_ROUTING_BRAND_MARK_IMAGE=namelens-openai-image

# Dedicated OpenAI image provider
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_IMAGE_ENABLED=true
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_IMAGE_AI_PROVIDER=openai
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_IMAGE_BASE_URL=https://api.openai.com/v1
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_IMAGE_MODELS_IMAGE=gpt-image-1.5
NAMELENS_AILINK_PROVIDERS_NAMELENS_OPENAI_IMAGE_CREDENTIALS_0_API_KEY=sk-...
```

### Provider Recommendations

| Provider             | Status       | Notes                                         |
| -------------------- | ------------ | --------------------------------------------- |
| **OpenAI GPT Image** | Recommended  | Best instruction following, no text in images |
| xAI grok-2-image     | Experimental | No size/aspect control, photorealistic style  |

See [known-issues.md](../known-issues.md) for detailed provider limitations.

---

## Workflow Examples

### Startup Brand Exploration

```bash
# 1. Generate name candidates
namelens generate "AI code review tool" --count 10

# 2. Check availability
namelens compare codereview aicodex codelyze reviewbot

# 3. Generate marks for top 2 finalists
namelens mark "aicodex" --out-dir ./marks/aicodex --color brand
namelens mark "codelyze" --out-dir ./marks/codelyze --color brand

# 4. Create thumbnails for stakeholder review
namelens image thumb --in-dir ./marks/aicodex --format jpeg
namelens image thumb --in-dir ./marks/codelyze --format jpeg
```

### Design Handoff Preparation

```bash
# Generate high-quality PNG with transparent background
namelens mark "finalname" --out-dir ./design-handoff \
  --color brand \
  --background transparent \
  --format png \
  --quality high \
  --count 5  # More options for designer

# Include in design brief alongside:
# - Brand positioning from namelens review
# - Phonetics analysis
# - Competitor research
```

### Monochrome for Print

```bash
# Classic black and white for business cards, letterhead
namelens mark "corpname" --out-dir ./print \
  --color monochrome \
  --format png \
  --quality high
```

---

## Command Reference

```
namelens mark <name> [flags]

Flags:
      --audience string     Target audience (e.g. developers, startups)
      --background string   Background: auto, transparent, opaque (default "auto")
      --color string        Color mode: monochrome, brand, vibrant
      --count int           Number of mark images to generate (default 3)
      --description string  One-line product description (helps image direction)
      --depth string        Generation depth: quick, deep (default "quick")
      --format string       Output format: png, jpeg, webp (default "png")
      --out-dir string      Write images to a directory (required)
      --quality string      Quality: auto, low, medium, high (default "auto")
      --size string         Image size (e.g. 1024x1024) (default "1024x1024")
```

```
namelens image thumb [flags]

Flags:
      --format string      Thumbnail format: jpeg or png (default "jpeg")
      --in-dir string      Input directory containing images
      --jpeg-quality int   JPEG quality (1-100) (default 80)
      --max-size int       Max thumbnail dimension (64-1024) (default 256)
      --out-dir string     Output directory (defaults to in-dir)
      --suffix string      Filename suffix (default "thumbnail")
```

---

## Tips

1. **Use `--color brand` for most projects** — The tech palette works well for
   SaaS, dev tools, and startups.

2. **Generate more than you need** — Use `--count 5` and pick the best
   directions to share with designers.

3. **Transparent backgrounds are flexible** — Use `--background transparent` so
   marks can be placed on any background color.

4. **Thumbnails for quick feedback** — Generate thumbnails before sending full
   images; they're faster to review.

5. **Iterate on finalists only** — Don't generate marks for every candidate;
   reserve this for 2-3 finalists.

---

## Limitations

- Generated marks are **directional concepts**, not final artwork
- Results vary between runs (AI generation is non-deterministic)
- Some providers may occasionally include unwanted text despite instructions
- Vector/SVG output not yet supported (planned)

---

## See Also

- [Compare Command](compare.md) — Screen candidates before generating marks
- [Expert Analysis](expert-search.md) — Deep brand research
- [Configuration](configuration.md) — Provider setup details
- [Known Issues](../known-issues.md) — Provider limitations
