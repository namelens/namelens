# Release Notes

This file keeps notes for the latest three releases in reverse chronological
order.

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

Highlights:

- **Setup wizard**: 30 seconds from install to full AI capability with
  `namelens setup`
- **Daemon mode**: Run server in background with `--daemon` flag
- **Control Plane API**: REST API for remote name checking and comparison
- **Anthropic Claude**: Third AI provider option alongside xAI and OpenAI
- **Expert mode guidance**: Safety warnings prevent false confidence from
  availability-only results
- **Environment files**: Auto-load `.env` from XDG config or specify with
  `--env-file`

### Upgrade Notes

- PID files moved from `~/.namelens/` to `~/.local/share/namelens/run/`
- Old `~/.namelens/` directory can be safely removed

See [v0.2.1 full release notes](docs/releases/v0.2.1.md) for details.
