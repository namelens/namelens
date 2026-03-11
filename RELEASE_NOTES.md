# Release Notes

This file keeps notes for the latest three releases in reverse chronological
order.

## v0.2.4 (2026-03-11)

Stability hardening: two bug fixes from standalone deployment and expert bulk
analysis, plus lint and security cleanup.

Highlights:

- **B13 fix**: Expert prompt loading now works outside git repos via embedded
  schema fallback
- **B14 fix**: Expert bulk rate-limit burst mitigated with serialized fallback,
  cooldown spacing, and exponential backoff
- **Code quality**: QF1012 lint cleanup and G115/G101 security hardening

### Expert Prompt Loading (B13)

Running `check --expert` from outside a git repository (e.g. from a `make
install` binary in `/usr/local/bin`) failed with "failed to load prompts"
because both `catalogForSchemas()` and `buildSchemaCatalog()` required a
repository root to locate schema files.

Fixed by embedding prompt and response schemas via `go:embed` with
temp-directory extraction fallback. Both the prompt loader
(`internal/ailink/prompt/loader.go`) and the ailink helpers
(`internal/cmd/ailink_helpers.go`) now fall back to embedded schemas when repo
root discovery fails.

Makefile `sync-embedded-config` and `verify-embedded-config` targets extended to
keep embedded copies in sync with source-of-truth schemas.

### Expert Bulk Rate-Limit Burst (B14)

The first name in a multi-name `--expert` batch occasionally got "provider
request failed" due to rate-limit burst immediately after the bulk request
completed.

Fixed with layered mitigations:

- Post-bulk fallback requests serialized with 2s initial cooldown and 1.5s
  spacing, scheduled from bulk completion time
- Fallback execution mutex-serialized to prevent concurrent provider calls
- Rate-limited (429) responses trigger exponential backoff retry (2s/4s/8s base,
  up to 3 attempts) with deterministic per-name SHA256 jitter

### Code Quality

- **QF1012**: Replaced `WriteString(fmt.Sprintf(...))` with `fmt.Fprintf()`
  across `internal/ailink/context/corpus.go`, `internal/output/analysis.go`, and
  `internal/output/markdown.go`
- **G115**: Added integer overflow bounds checks on uint64→int64 timestamp
  conversions, int→uint16 port conversions, and int→uint32 PID conversions in
  daemon discovery and lifecycle code
- **G101**: Suppressed false positive on generated `ApiKeyScopes` constant in
  `internal/api/openapi.gen.go`

### Upgrade Notes

No breaking changes. All v0.2.3 configurations remain valid.

---

## v0.2.3 (2026-02-23)

Dogfooding polish: bug fixes from the v0.2.2 naming exercise, brand workflow
improvements, and provider routing UX.

Highlights:

- **Brand context for review**: `--context-file` and `--scan-dir` pass product
  context through to brand analyses — transforms generic AI guesses into
  context-aware brand assessments
- **Provider override for generate**: `--provider` enables invocation-scoped
  provider selection for A/B testing across AI backends
- **OSS profile**: Registry-and-handle-only checks for open-source projects
- **Seven bug fixes** from dogfooding, including expert output cleanup and
  schema validation corrections

### Brand Context for Review

Brand analyses without product context produce generic assessments. Now you can
pass context directly:

```bash
# From a file (truncated to 2000 chars)
namelens review myproject --mode=brand --depth=deep \
  --context-file ./VISION.md

# Scan a directory for context files
namelens review myproject --mode=brand --depth=deep \
  --scan-dir ./docs
```

The context is injected into `brand-plan` and `brand-proposal` prompts,
producing assessments that reflect your actual product positioning, target
audience, and competitive landscape.

### Provider Override for Generate

Switch AI providers per-invocation without editing config:

```bash
# Force Anthropic for this run only
namelens generate "agent gateway" --provider namelens-anthropic

# Force OpenAI with deep depth
namelens generate "agent gateway" --depth deep --provider namelens-openai
```

Provider precedence: `--provider` flag > routing table > provider roles >
default provider > single-provider fallback.

Validation gives clear errors:

- Unknown provider:
  `unknown provider "foo" (valid: namelens-anthropic, namelens-openai, namelens-xai)`
- Disabled provider: `provider "foo" is disabled`

### OSS Profile

Open-source projects rarely need domain checks. The new `oss` profile checks
only package registries and social handles:

```bash
namelens check mylib --profile=oss
# Checks: npm, pypi, cargo, github (no domains)
```

### Bug Fixes

- **Expert output (B3, B11)**: NAME column in `check --expert` shows the checked
  name, not "ailink" — fixed in two passes across `output/notes.go` and
  `check.go`
- **xAI markup (B5)**: Internal Grok citation markup (`<grok:render>`,
  `<argument>` tags) stripped from responses before display
- **Schema validation (B12)**: `brand-plan` prompt no longer fails schema
  validation when called via `generate` or `review` — response schema matching
  corrected for non-search prompts
- **Check defaults (B6)**: `namelens check <name>` without `--profile` now
  covers .com/.dev/.io/.app, npm/pypi/cargo, and GitHub
- **Standalone binary**: CLI works correctly when run from outside the
  repository directory — embedded config defaults used as fallback
- **Deep generation**: `--depth=deep` now uses broader naming strategies for
  less correlated candidates
- **Profile slice aliasing**: Returned profile objects no longer share backing
  arrays with global built-in definitions

### Review Phonetics

International phonetic analysis can now be scoped:

```bash
namelens review myname --mode=core --locales en,de,ja --keyboards qwerty,qwertz
```

### Upgrade Notes

No breaking changes. All v0.2.2 configurations remain valid.

The default check behavior has changed: bare `namelens check <name>` now checks
more targets than before (4 TLDs + 3 registries + GitHub vs. just .com). Use
`--profile=minimal` for the previous behavior.

---

## v0.2.2 (2026-02-20)

Model refresh: updated Anthropic and OpenAI model tiers to current releases.

Highlights:

- **Anthropic models updated**: Default and reasoning tiers now use
  `claude-sonnet-4-6`; fast tier updated to `claude-haiku-4-5-20251001`
- **OpenAI reasoning tier added**: `o3` configured as the `reasoning` model for
  `--depth=deep` workloads — OpenAI's dedicated reasoning model delivers
  significantly higher quality for deep brand analysis

### Model Updates

| Provider  | Tier        | Before                     | After                     |
| --------- | ----------- | -------------------------- | ------------------------- |
| Anthropic | `default`   | claude-sonnet-4-5-20250929 | claude-sonnet-4-6         |
| Anthropic | `reasoning` | claude-sonnet-4-5-20250929 | claude-sonnet-4-6         |
| Anthropic | `fast`      | claude-3-5-haiku-20241022  | claude-haiku-4-5-20251001 |
| OpenAI    | `reasoning` | (not set)                  | o3                        |

To update an existing config, re-run the setup wizard or edit
`~/.config/namelens/config.yaml` directly.

### Upgrade Notes

No breaking changes. Existing configs with the old model names remain valid —
the models are still available from Anthropic. Update at your convenience.

---

## v0.2.1 (2026-02-14)

Agent-ready deployment: headless server API, guided setup, and safety
guardrails.

See [v0.2.1 full release notes](docs/releases/v0.2.1.md) for details.
