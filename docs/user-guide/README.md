# NameLens User Guide

> **Disclaimer**: This tool is for informational purposes only and does not
> constitute legal or professional advice. See
> [USAGE-NOTICE.md](../USAGE-NOTICE.md) before making business decisions based
> on results.

---

## Get Started Quickly

| Your Situation                              | Where to start                             |
| ------------------------------------------- | ------------------------------------------ |
| "I have a name, is it available?"           | [Quick Availability Check](quick-start.md) |
| "I need name ideas, not just checks"        | [Name Generation](generate.md)             |
| "I'm serious about this name—deep research" | [Expert Analysis](expert-search.md)        |
| "I want logo/mark directions for my name"   | [Brand Mark Generation](mark.md)           |
| "I run a startup, need the full picture"    | [Startup Naming Guide](startup-guide.md)   |
| "I'm integrating into my workflow"          | [Configuration](configuration.md)          |

---

## Core Workflows

### Workflow 1: Quick Availability Check

The 5-second check for developers and rapid prototyping.

```bash
namelens check myproject
```

**When to use:**

- Early-stage ideation
- Domain squat checks
- CI/CD pipeline validation
- Batch checking multiple candidates

See: [Quick Availability Check](quick-start.md)

---

### Workflow 2: Deep Brand Analysis

Full research for projects you're serious about.

```bash
namelens check acmecorp --expert --phonetics --suitability
```

**What you get:**

- Trademark conflict risk scoring
- Social media sentiment analysis
- Web search for existing products
- Pronunciation and typeability scores
- Cultural suitability across markets
- Clear proceed/caution/stop recommendation

**When to use:**

- Before registering domains
- Before printing marketing materials
- Before incorporating your business
- Before launching public campaigns

See: [Expert Analysis](expert-search.md)

---

### Workflow 3: Name Generation

Generate candidates from your product concept.

```bash
namelens generate "static analyzer for shell scripts"
```

**When to use:**

- Brainstorming phase
- You have features but no name
- Need fresh perspectives
- Want to explore naming patterns

See: [Name Generation](generate.md)

---

### Workflow 4: Compare Finalists

Compare 2-10 candidates with phonetics and suitability scores.

```bash
namelens compare acmecorp stellaplex fluxio
```

**When to use:**

- Shortlisted 3-5 names from batch or generate
- Need phonetics and cultural suitability scores
- Stakeholder presentations
- Decision matrices

See: [Compare Command](compare.md)

---

### Workflow 5: Brand Mark Generation

Generate logo/mark directions for finalist names.

```bash
namelens mark "myproject" --out-dir ./marks --color brand
```

**What you get:**

- 3+ distinct logo mark directions
- AI-generated images for each concept
- Configurable color palettes
- Transparent background support

**When to use:**

- After selecting 1-3 finalist names
- Before engaging a designer
- Internal presentations
- Quick visual exploration

See: [Brand Mark Generation](mark.md)

---

### Workflow 6: Batch Screening

Check many names from a file for initial filtering.

```bash
namelens batch candidates.txt --output-format=table
```

**When to use:**

- 20+ candidates to filter
- Initial availability screening
- Bulk checking before compare

See: [Batch Processing](batch.md)

---

## Specialized Guides

### Startup Naming Guide

End-to-end guidance for new ventures.

[Read the Startup Naming Guide →](startup-guide.md)

### Marketing & Brand Alignment

Using Namelens for competitive research and brand gap analysis.

[Read the Brand Guide →](brand-guide.md)

### Integration & Automation

MCP server, API, and CI/CD integration.

[Read the Integration Guide →](integration.md)

---

## Reference

| Document                              | Description                       |
| ------------------------------------- | --------------------------------- |
| [Brand Mark Generation](mark.md)      | Logo/mark directions and images   |
| [Compare Command](compare.md)         | Side-by-side finalist comparison  |
| [Configuration](configuration.md)     | Profiles, env vars, customization |
| [Domain Fallback](domain-fallback.md) | WHOIS and DNS fallback for TLDs   |
| [Expert Prompts](expert-prompts.md)   | Available AI analysis prompts     |

---

## Real-World Examples

| Example                                                             | What You'll Learn                   |
| ------------------------------------------------------------------- | ----------------------------------- |
| [How Namelens Named Itself](../examples/namelens-origin-story.md)   | Full journey from codename to brand |
| [The Tesla Trademark Lesson](../examples/tesla-trademark-lesson.md) | Why domain availability ≠ usability |

See [examples/README.md](../examples/README.md) for all examples and how to use
them as learning tools.

---

## Need Help?

```bash
namelens --help
namelens generate --help
namelens check --help
namelens doctor
```

## Related Documentation

- [AGENTS.md](../../AGENTS.md) - AI agent guidelines
- [Architecture](../architecture/) - System design decisions
- [Operations](../operations/) - Build and deployment
- [AILink](../ailink/) - AI provider integration (internal)
