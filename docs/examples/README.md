---
last_updated: 2026-01-25
tool_version: 0.1.4
---

# Examples

Real-world examples demonstrating namelens capabilities.

## How to Use These Examples

Each example is a complete case study you can follow:

1. **Read the narrative** — Understand the problem and context
2. **See the commands** — Copy the exact commands used
3. **Analyze the output** — Interpret what Namelens found
4. **Apply the lessons** — Use these insights for your own naming

**Example as a teaching tool:**

The [Tesla Trademark Lesson](tesla-trademark-lesson.md) shows why "domain
available" ≠ "name safe." Run it yourself:

```bash
namelens check tesla --tlds=com --expert --expert-prompt=domain-content
```

Compare the results with the documented output. See what the AI caught that a
simple availability check missed.

## Available Examples

| Example                                                   | Description                                     | Features Used                                                                       |
| --------------------------------------------------------- | ----------------------------------------------- | ----------------------------------------------------------------------------------- |
| [Tesla Trademark Lesson](tesla-trademark-lesson.md)       | Why domain availability doesn't equal usability | name-availability, domain-content                                                   |
| [namelens Origin Story](namelens-origin-story.md)         | How this tool named itself (with 2026 update)   | name-availability, domain-content, brand-proposal, name-phonetics, name-suitability |
| [namelens Brand Mark Story](namelens-brand-mark-story.md) | How this tool created its own icon              | namelens mark, --color, --description, --audience, TinEye validation                |

## Example Categories

### Trademark & Legal Risk

- **Tesla Trademark Lesson** - Expert analysis catching trademark conflicts even
  when domains are technically available

### Naming Strategy

- **namelens Origin Story** - Full naming journey from codename discovery
  through candidate evaluation to final brand selection

### Brand Identity

- **namelens Brand Mark Story** - Generating, iterating, and validating a brand
  mark using the mark command and TinEye similarity checking

## Running These Examples

All examples use the standard namelens CLI:

```bash
# Basic check
namelens check <name> --tlds=com,io,dev

# With expert analysis (requires API key)
namelens check <name> --tlds=com,io,dev --expert

# Full analysis with phonetics and suitability
namelens check <name> --profile=developer --expert --phonetics --suitability

# Check social handles
namelens check <name> --handles=github

# Use a specific expert prompt (e.g., brand-proposal, name-phonetics)
namelens check <name> --expert --expert-prompt=brand-proposal
```

See the [User Guide](../user-guide/README.md) for full documentation.

## Contributing Examples

Examples should include:

- Actual command output (markdown-fenced)
- Clear explanation of what the tool found
- Lessons learned or key takeaways
- Tool version used for reproducibility
