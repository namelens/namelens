# Release Notes

This file keeps notes for the latest three releases in reverse chronological
order.

## v0.2.4 (2026-05-05)

Stability hardening + small CLI ergonomics + supply-chain hygiene. Originally
prepared in March around two bug fixes (B13/B14); the release picked up material
from three subsequent dogfood-driven sessions before tagging.

Highlights:

- **B13 fix**: `check --expert` works outside git repos via embedded schema
  fallback
- **B14 fix**: expert-bulk rate-limit burst mitigated with serialized fallback,
  cooldown spacing, and exponential backoff
- **`check --provider`**: provider override flag now matches the equivalent on
  `generate` — no more env-var workaround for A/B'ing AI backends
- **Vulnerability dependencies cleared**: Go toolchain pinned to 1.26.2 and
  `golang.org/x/image` bumped — `goneat dependencies --vuln` returns 0 findings
- **YAML tooling alignment**: `.yamlfmt` and `.yamllint` added at repo root to
  stop the goneat ↔ yamlfmt comment-padding oscillation
- **CI infra**: runner image bumped to `goneat-tools-runner-glibc:v0.4.0` (Go
  1.26.2 + bundled `golangci-lint v2.12.1`); redundant CI install step removed
- **`make pr-final`** strict-local-gate target so contributors can verify the
  same checks CI runs before pushing PR updates

### `check --provider`

Switch AI providers per-invocation on `check` without editing config or
exporting env vars:

```bash
namelens check acmecorp --expert --provider namelens-anthropic
namelens check acmecorp --expert --provider namelens-xai --expert-depth deep
```

Same validation as `generate`'s flag — unknown or disabled provider keys fail
with the list of valid IDs.

### `--expert` provider capability — read this if you switch providers

xAI's Grok models perform real-time web search by default; Anthropic's Claude
does **not**. For `check --expert` the difference is material — a user
defaulting to Claude can get confident-but-shallow verdicts that miss recent or
niche conflicts. New "Choosing a provider for `--expert`" section in
`docs/ailink/README.md` covers this; cross-checking with multiple providers is
also a valid signal that something deserves a human look.

### Vulnerability cleanup

`goneat dependencies --vuln` reported 12 findings (1 critical + 6 high + 5
medium) on the previous tip:

- 11 in Go stdlib at `go1.26.1` — cleared by pinning `toolchain go1.26.2` in
  `go.mod`. Go auto-downloads the toolchain on demand
- 1 in `golang.org/x/image v0.35.0` (`GHSA-44p7-9xx4-hf2g`) — cleared by bumping
  to `v0.38.0` (pulls `golang.org/x/text` `v0.33.0` → `v0.35.0` transitively)

Post-bump scan returns 0 findings.

### Expert prompt loading (B13)

Running `check --expert` from outside a git repository (e.g. from a
`make install` binary in `/usr/local/bin`) failed with "failed to load prompts"
because both `catalogForSchemas()` and `buildSchemaCatalog()` required a
repository root. Fixed by embedding prompt and response schemas via `go:embed`
with temp-directory extraction fallback.

### Expert bulk rate-limit burst (B14)

The first name in a multi-name `--expert` batch occasionally got "provider
request failed" due to rate-limit burst immediately after the bulk request
completed. Fixed with layered mitigations: post-bulk fallback serialized with 2s
initial cooldown / 1.5s spacing, mutex-serialized execution, and 429-triggered
exponential backoff (2s/4s/8s) with deterministic per-name SHA256 jitter.

### Process and infra

- Local hooks regenerated **without** the guardian intercept — moving from
  microteam direct-push to a PR-based workflow with branch protection on the
  remote replacing the local approval prompt. User-level guardian config
  (`~/.goneat/guardian/config.yaml`) is unchanged so other repos keep their
  existing protections
- `make api-generate` is now idempotent: re-injects the `// #nosec G101`
  suppression that oapi-codegen drops on regen, so `check-api` stays clean
  across regenerations
- `.yamlfmt` and `.yamllint` added with `pad_line_comments: 2` set _explicitly_
  — yamlfmt's default of 1 fights goneat's pin of 2 and oscillates in CI if the
  value isn't pinned (see goneat appnote `yaml-format-lint-alignment`)
- New `make pr-final` target — same shape as `prepush` but stricter
  (`--fail-on medium`); documented in `AGENTS.md` and `RELEASE_CHECKLIST.md`

### Code quality

- errcheck cleanup in `internal/cmd/generate.go`: ~80 unchecked `fmt.Fprintf` /
  `Fprintln` calls in the result renderers refactored through a small
  `errWriter` helper. Public signatures unchanged so the existing tests still
  drive the renderers via `bytes.Buffer`
- QF1012 lint cleanup: `WriteString(fmt.Sprintf(...))` → `fmt.Fprintf(...)`
  across output formatting code
- G115/G101 security hardening: integer overflow bounds checks on uint64→int64,
  int→uint16, int→uint32 conversions in daemon code

### Upgrade notes

No breaking changes. All v0.2.3 configurations remain valid.

If your CI references `goneat-tools-runner-glibc:v0.3.5` or earlier, bump to
`:v0.4.0` and remove any separate `golangci-lint` install step — the runner now
ships a verified binary built against its pinned Go.

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
