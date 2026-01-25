---
last_updated: 2026-01-25
tool_version: 0.1.4
model: OpenAI GPT Image 1.5
---

# How namelens Got Its Icon

**A tool that created its own brand mark**

> In v0.1.0, we used namelens to name itself. In v0.1.4, we used namelens to
> create its own icon. The "network viewfinder" mark was generated, selected,
> and validated entirely through the tool's brand mark pipeline.

---

## The Challenge

After using namelens to find its own name, we needed a visual identity. The
requirements:

- Represent what namelens does (checking availability across connected sources)
- Work as a favicon, README badge, and social icon
- Avoid resemblance to existing trademarks
- Fit the developer/startup audience

## The Process

### Round 1-4: Finding the Right Model

We started with DALL-E 3, which produced decent results but had issues:
- Occasional text leakage ("COMPRNY" appearing in one mark)
- Inconsistent instruction following

We switched to **GPT Image 1.5**, which produced cleaner output with better
instruction following.

### Round 5-8: Adding Color Control

Early rounds produced greyscale output. We added `--color` flag support:

```bash
namelens mark "namelens" --out-dir ./marks --color brand
```

The `brand` mode produces a consistent teal/navy/coral tech palette.

### Round 9-10: Adding Context

Without context, the model inferred "camera lens" imagery from the name alone.
We added `--description` and `--audience` flags:

```bash
namelens mark "namelens" \
  --description "CLI tool for checking brand name availability" \
  --audience "developers and startups" \
  --color brand
```

This shifted output toward network/connectivity concepts.

### Round 11: The Winning Concept

With a more explicit description, we got exactly what we wanted:

```bash
namelens mark "namelens" --out-dir ./marks \
  --description "A viewfinder or focus bracket frame containing a network of connected nodes inside - representing inspecting brand availability across multiple connected sources" \
  --audience "developers and startups" \
  --color brand \
  --background transparent \
  --format png \
  --quality high
```

**Result**: "network viewfinder" - viewfinder brackets containing connected
nodes. This represents:

- **Viewfinder/brackets**: The "lens" inspection metaphor
- **Connected nodes**: Multi-source checking (domains, registries, trademarks)
- **Teal/navy palette**: Modern tech aesthetic

<img src="https://raw.githubusercontent.com/namelens/.github/main/assets/namelens-icon-256.png" width="128" alt="namelens icon">

## Validation

Before adopting the mark, we ran it through TinEye:

```
TinEye searched over 80.9 billion images but didn't find any matches.
```

**0 matches** - the generated icon is unique and doesn't conflict with existing
images on the web.

## The Final Asset

The icon is stored in the [namelens/.github](https://github.com/namelens/.github)
repository with full provenance:

| File | Size | Use |
|------|------|-----|
| `namelens-icon-original.png` | 1024×1024 | Master |
| `namelens-icon-512.png` | 512×512 | High-res |
| `namelens-icon-256.png` | 256×256 | Web |
| `namelens-icon-128.png` | 128×128 | README |
| `namelens-icon-64.png` | 64×64 | Badges |
| `namelens-icon-32.png` | 32×32 | Favicon |

See [PROVENANCE.md](https://github.com/namelens/.github/blob/main/assets/PROVENANCE.md)
for full generation details.

## Lessons Learned

1. **Context matters**: Without `--description`, the model infers from the name
   alone. "namelens" → camera imagery. With context → network/validation imagery.

2. **Color needs explicit control**: Default output was greyscale. The `--color`
   flag ensures consistent brand palette.

3. **Iterate on prompts**: It took 11 rounds to get the right concept. The
   `namelens mark` command makes iteration fast.

4. **Validate before adopting**: TinEye check confirmed uniqueness. Always
   verify generated marks don't resemble existing trademarks.

---

## Try It Yourself

```bash
# Generate marks for your project
namelens mark "yourproject" --out-dir ./marks \
  --description "What your project does" \
  --audience "Who it's for" \
  --color brand

# Create thumbnails for sharing
namelens image thumb --in-dir ./marks --format jpeg
```

See [Brand Mark Generation](../user-guide/mark.md) for full documentation.
