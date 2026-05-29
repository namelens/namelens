# Release Notes

This file keeps notes for the latest three releases in reverse chronological
order.

## v0.2.5 (2026-05-29)

**Make `--expert` trustworthy.** JSON parity with markdown, calls that don't
time out under realistic loads, bulk-fallback that actually delivers expert
data, and `--timeout` coverage on every user-invoked AI call. Surfaced by the
2026-05-08 recipekit dogfood session; landed across four PRs.

Highlights:

- **`--timeout` flag uniform across all five AI-invoking commands**: `check`,
  `generate`, `mark` (text + image legs), `compare`, and `review` all accept
  `--timeout <duration>` (e.g. `--timeout 240s`, `--timeout 4m`); overrides
  `ailink.default_timeout` for that invocation
- **`ailink.default_timeout` raised `60s` → `180s`**: matches realistic
  Anthropic deep `generate` and multi-name `--expert-bulk` budgets observed in
  dogfooding. Driver-layer ceiling fix lets per-call `--timeout` flow through to
  the underlying HTTP client without re-clamping to the config default
- **JSON `expert` field on `check` output**: `--output-format json` now includes
  the synthesized expert digest (`risk`, `notes`, raw payload). JSON-consuming
  agents and pipelines no longer blind to trademark/conflict intelligence
- **`--expert-bulk` reliability fixes**: centralized name canonicalization
  closes a silent-merge-miss on mixed-case input; `waitForFallbackSlot` closes
  the ctx-cancel slot-drop hole so every requested name gets a populated batch
  slot

### `--timeout` flag uniform across commands

Every AI-invoking command honors `--timeout` with the same semantics:

```bash
namelens check acmecorp --expert --timeout 240s
namelens generate "agent gateway" --depth deep --timeout 4m
namelens mark acmecorp --color brand --timeout 3m
namelens compare alpha beta gamma --timeout 180s
namelens review acmecorp --mode brand --timeout 240s
```

`--timeout` overrides `ailink.default_timeout` for that single call. `0` = use
config default. Same help-text shape across all five commands.

This closes the **make `--expert` trustworthy** theme — every user-invoked AI
call can now be bounded with a known deadline. The 2026-05-08 dogfood session
reported timeouts at 60s on Anthropic deep `generate` (5KB brief) and on
`--expert-bulk` over 10 names. Both flows now complete cleanly at the new `180s`
default; per-invocation `--timeout` is available when you need more.

`mark`'s `--timeout` flag was added earlier in the v0.2.5 cycle (PR-2 / #6) and
initially only bounded the image-generation call. PR-4 (#8) threads the same
flag through the shared `runReviewGenerate` helper so it now bounds the
mark-prompt text leg too.

### JSON `expert` field parity

`check --expert --output-format json` previously omitted the synthesized expert
digest that the markdown renderer surfaced — JSON consumers had to parse
`ailink`/`ailink_error` themselves and re-implement the digest. v0.2.5 adds an
`Expert *ExpertDigest` field on `core.BatchResult`, populated by a single
render-time synthesizer (`internal/core/expert.go` `NewExpertDigest`) shared by
both the markdown and JSON paths.

The synthesizer is a pure pass-through — it reads `risk_level` and `notes` from
the AI response and surfaces them without transformation. Same payload in both
`--output-format markdown` and `--output-format json`.

### `--expert-bulk` keying + cancel-hole fixes

Two bugs surfaced in dogfooding:

1. **Silent merge miss on mixed-case names.** `lookup` used raw input, the live
   bulk map used lower/trimmed item names, and the cached bulk map used raw item
   names — three different keys in three places. `expert: null` could appear for
   a name that had a successful fallback. Centralized canonicalization via
   `expertBulkKey` / `bulkItemsToMap` closes the gap
2. **Ctx-cancel slot-drop.** Ctrl-C during the fallback cooldown produced a
   missing batch slot rather than an error-tagged entry. Downstream consumers
   that assumed every requested name had a slot broke. `waitForFallbackSlot` now
   returns `AILINK_FALLBACK_CANCELED` and falls through to the standard
   `summarizeResults` write path

Seven new tests in `internal/cmd/expert_bulk_test.go` exercise mixed-case input
through the bulk + fallback path; the cancel-hole test exercises context
cancellation during cooldown.

### Upgrade notes

No breaking changes. All v0.2.4 configurations remain valid.

If you've pinned `ailink.default_timeout` to `60s` explicitly, no change — your
pin holds. If you relied on the implicit `60s` default, you'll get `180s` now;
on observable behavior, this means previously-timing-out calls will complete
rather than fail. If you specifically want the old default back, pin
`ailink.default_timeout: 60s` in your config.

JSON consumers of `check` output will see a new `expert` field on results that
have expert payload. Existing fields (`ailink`, `ailink_error`) are unchanged —
the new field is additive.

---

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

Model refresh: Anthropic default/reasoning to `claude-sonnet-4-6`, fast to
`claude-haiku-4-5-20251001`; OpenAI reasoning tier added (`o3`).

See [v0.2.2 full release notes](docs/releases/v0.2.2.md) for details.
