# ADR-0004: AILink Prompt Authoring Standards

**Status**: Accepted **Date**: 2026-01-04 **Deciders**: @3leapsdave

## Context

NameLens uses AILink prompts to communicate with LLM providers. These prompts
are markdown files with YAML frontmatter containing structured examples
(JSON/YAML) that define expected response formats.

During v0.1.0 development, we discovered that prettier (via goneat format) was
mangling unfenced JSON examples in prompt files. The JSON was treated as prose
and either:

- Line-wrapped at 80 characters, breaking structure
- Had indentation stripped entirely

This created a maintenance burden and risked corrupting prompts during routine
formatting.

## Decision

### 1. All structured examples must use fenced code blocks

JSON and YAML examples in prompt files MUST be wrapped in fenced code blocks
with language hints:

**Correct:**

````markdown
Respond in this JSON structure:

```json
{
  "name": "example",
  "score": 85
}
```
````

**Incorrect:**

```markdown
Respond in this JSON structure:

{
  "name": "example",
  "score": 85
}
```

### 2. Fenced blocks do not affect LLM behavior

Analysis confirms that fenced code blocks in prompts:

- Do not cause LLMs to include fences in responses (governed by response
  instructions)
- Improve prompt clarity by explicitly marking example structures
- Follow standard markdown conventions LLMs understand from training data
- Enable proper syntax highlighting in editors and documentation

### 3. Response format instructions remain explicit

Even with fenced examples, prompts should include explicit response format
instructions:

````markdown
Respond EXCLUSIVELY in this JSON structure (no markdown, no extra text):

```json
{ ... }
````

````

The instruction governs response format; the fenced block governs example presentation.

### 4. Prompt file structure

Standard prompt file structure:

```markdown
---
slug: prompt-name
name: Human-readable name
description: What this prompt does
version: 1.0.0
# ... other frontmatter
response_schema:
  $ref: "ailink/v0/response-schema"
---

System instructions and context.

Input variables: {{variable}}

Guidelines:
- Bullet points for behavior guidance

Response format instruction:

```json
{
  "structured": "example"
}
````

````

### 5. Prettier configuration

The project `.prettierrc` includes `embeddedLanguageFormatting: "off"` to prevent prettier from reformatting code within fenced blocks. This preserves intentional formatting in examples.

## Consequences

### Positive

- Prompts survive automated formatting without corruption
- Clear visual separation between prose and structured examples
- Consistent authoring standard across all AILink prompts
- Better editor support (syntax highlighting, folding)
- Examples render correctly in documentation

### Negative

- Slightly more verbose prompt files (fence markers)
- Authors must remember to fence examples (enforced by review)

## Implementation

All prompt files in `internal/ailink/prompt/prompts/` updated to use fenced code blocks for JSON response examples.

## Validation

Prompt standards are enforced via `make check-prompts`, which verifies:

1. **Format idempotency**: Running `goneat format --check` produces no changes
2. **Fenced block presence**: All prompt files contain ` ```json ` markers

```makefile
check-prompts: ## Validate AILink prompt formatting and structure
	@goneat format --check internal/ailink/prompt/prompts/
	@for f in internal/ailink/prompt/prompts/*.md; do \
		grep -q '```json' "$$f" || (echo "Missing fenced JSON: $$f" && exit 1); \
	done
````

Future enhancements may include:

- JSON validity checking (extract fenced blocks, validate with `jq`)
- Schema conformance (validate examples against declared `response_schema`)
- Frontmatter validation (required fields, valid YAML)

## References

- Prettier embedded language formatting:
  https://prettier.io/docs/en/options.html#embedded-language-formatting
- CommonMark fenced code blocks:
  https://spec.commonmark.org/0.30/#fenced-code-blocks
