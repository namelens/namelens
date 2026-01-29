# Context Command

The `context` command scans a directory and produces a structured corpus for AI
prompts. This enables inspection, editing, and reuse of context before
generation.

## Overview

Instead of passing directories directly to `generate`, the context command lets
you:

1. **Inspect** - See exactly what files will be included
2. **Edit** - Modify the corpus before generation
3. **Iterate** - Refine context gathering independently
4. **Compose** - Pipe output to other tools or save for reuse

## Usage

### Basic Usage

```bash
# Generate JSON corpus from a directory
namelens context ./planning

# Generate markdown for human review
namelens context ./planning --output=markdown

# See what would be included without content
namelens context ./planning --manifest-only
```

### Pipeline Usage

```bash
# Pipe directly to generate
namelens context ./planning | namelens generate "my product" --corpus=-

# Save and reuse
namelens context ./planning > corpus.json
namelens generate "my product" --corpus=corpus.json
```

## File Classification

The context command classifies files by type and allocates budget accordingly.
Matching is case-insensitive (`README.md` and `readme.md` both work).

| Class | Patterns | Budget | Extract Mode |
|-------|----------|--------|--------------|
| readme | README.md, README.txt | 25% | Full content |
| architecture | ARCHITECTURE.md, DESIGN.md, STRUCTURE.md | 20% | Full content |
| decisions | DECISIONS.md, ADR-*.md | 20% | Full content |
| planning | BOOTSTRAP.md, PLAN.md, CONCEPT.md, VISION.md | 15% | Full content |
| project_metadata | package.json, Cargo.toml, go.mod, pyproject.toml | 5% | Metadata only |
| general_docs | docs/*.md, *.md | 15% | Full content |
| code | *.go, *.rs, *.py, *.ts, *.js | 0% | Excluded |

### Budget Allocation

Files are included by class priority, not first-come-first-served. A README
always gets included before general docs, even if discovered later. Each class
has a maximum budget share to ensure diverse representation.

### Metadata Extraction

For project files (package.json, Cargo.toml, etc.), only key metadata is
extracted:

- Name, version, description
- Keywords, license
- **Not** dependencies (too verbose, changes frequently)

## Output Formats

### JSON (Default)

Schema-backed format for programmatic use:

```json
{
  "$schema": "https://schemas.namelens.dev/context/v1.0.0.schema.json",
  "version": "1.0.0",
  "generated_at": "2026-01-29T08:00:00Z",
  "source": {
    "type": "directory",
    "path": "./planning"
  },
  "budget": {
    "max_chars": 32000,
    "used_chars": 18416
  },
  "manifest": {
    "total_files_scanned": 5,
    "files_included": 3,
    "files_excluded": 2
  },
  "files": [...],
  "content": [...]
}
```

### Markdown

Human-readable format with tables and formatted content:

```bash
namelens context ./planning --output=markdown
```

### Prompt

Optimized format for AI prompt inclusion:

```bash
namelens context ./planning --output=prompt
```

## Flags

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--output` | `-o` | string | Output format: `json` (default), `markdown`, `prompt` |
| `--budget` | | int | Max characters to include (default: 32000) |
| `--manifest-only` | | bool | Output manifest without content |
| `--include` | | strings | Additional glob patterns to include |
| `--exclude` | | strings | Glob patterns to exclude (planned) |

## Integration with Generate

The `generate` command accepts pre-generated corpus via `--corpus`:

```bash
# From file
namelens generate "my product" --corpus=corpus.json

# From stdin
namelens context ./planning | namelens generate "my product" --corpus=-

# Format auto-detected (JSON or markdown)
```

Priority for description sources:
1. `--description` (inline)
2. `--corpus` (pre-generated)
3. `--description-file` (single file)
4. `--scan-dir` (live scan)

## Examples

### Inspect Before Generate

```bash
# See what files would be included
namelens context ./my-project --manifest-only

# Output:
# {
#   "manifest": {
#     "total_files_scanned": 12,
#     "files_included": 4,
#     "files_excluded": 8
#   },
#   "files": [
#     {"path": "README.md", "class": "readme", "chars": 2400},
#     {"path": "DESIGN.md", "class": "architecture", "chars": 5600},
#     ...
#   ]
# }
```

### Review in Markdown

```bash
# Generate human-readable summary
namelens context ./my-project --output=markdown > context-review.md

# Review and edit if needed
vim context-review.md

# Then use edited version
namelens generate "my product" --corpus=context-review.md
```

### Adjust Budget

```bash
# Small context for quick iteration
namelens context ./my-project --budget=8000

# Large context for comprehensive analysis
namelens context ./my-project --budget=64000
```

### Add Additional Files

```bash
# Include requirements files
namelens context ./my-project --include="requirements*.txt"

# Include specific config
namelens context ./my-project --include="*.yaml"
```

## Related

- [Generate](generate.md) - Use context in name generation
- [Configuration](configuration.md) - Environment and config file setup
