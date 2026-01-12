# Output & File Writing Standard

This project separates **formatting** (how results are rendered) from **destinations** (where results are written).

This keeps CLI output consistent across commands and makes it easy to move between:
- interactive terminal use
- CI/automation
- generated reports saved to files

## Terms

- **Output format**: the serialization/renderer to use (`table`, `json`, `markdown`).
- **Out**: write the primary output to a single file (or stdout).
- **Out dir**: write per-name artifacts to a directory (plus an index file).

## Flags

### `--output-format`

Controls output formatting.

- `--output-format=table` (default): console-friendly.
- `--output-format=json`: machine-friendly.
- `--output-format=markdown`: report-friendly.

### `--out`

Write the primary rendered output to a single file.

- If omitted: write to stdout.
- If `--out=-`: write to stdout.

Examples:

```bash
namelens check namelens --output-format=json --out namelens.check.json
namelens review namelens --output-format=markdown --out namelens.review.md
```

### `--out-dir`

Write per-name artifacts to a directory (plus an index file). Useful for multi-name runs.

- `--out` and `--out-dir` are mutually exclusive.

Index and per-name file naming:

- `check`:
  - `check.index.<ext>`
  - `<name>.check.<ext>`
- `review`:
  - `review.index.<ext>`
  - `<name>.review.<ext>`
- `batch`:
  - `batch.index.<ext>`
  - `<name>.batch.<ext>`

Where `<ext>` depends on format:
- `json` → `json`
- `markdown` → `md`
- `table` → `txt`

## Multi-name Inputs

Commands that accept multiple names support:

- Positional names: `namelens check foo bar baz`
- File input: `--names-file candidates.txt`
- Stdin input: `--names-file -`

## Filename Sanitization

When writing per-name files, names are sanitized:
- lowercased
- characters outside `[a-z0-9._-]` replaced with `-`
- leading/trailing `.` and `-` removed

If a name becomes empty after sanitization, it falls back to `output`.

## Historical Releases

Release notes in `docs/releases/` describe the CLI surface at the time of that release.
They should not be “modernized” when flags change later.
