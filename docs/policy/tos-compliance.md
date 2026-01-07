# NameLens TOS Compliance Policy

## Purpose

NameLens performs availability checks across third-party services. We commit to
respecting each provider's Terms of Service (TOS) and acceptable use policies.
We will take commercially reasonable actions to ensure our code does that.

## Policy

- Use official, documented APIs when available.
- Avoid scraping or undocumented endpoints unless explicit written permission is
  obtained.
- Support user-provided credentials where required to comply with rate limits
  and access controls.
- Implement rate limiting, caching, and backoff to honor service constraints.
- Apply a safety margin to published limits (default 90%) unless explicitly
  overridden for a justified use case.
- Document data sources and request behavior in code comments and docs.
- Review and update integrations when providers change terms or endpoints. This
  review is a required step when adding or modifying integrations.

## Operational Observability

- Log check throughput (total checks, elapsed time, checks/sec) for visibility.

## Phase 7 Compliance Notes (Registries)

These checks will be implemented using official APIs:

- npm: `https://registry.npmjs.org/<name>` (auth optional for higher limits).
- PyPI: `https://pypi.org/pypi/<name>/json`.
- GitHub handles: GitHub REST API with PAT for reliable limits (public
  endpoints).

## ADR/SOP Requirement

We will maintain explicit ADRs and/or SOPs in this repository describing how
each integration complies with provider TOS and what safeguards are implemented
(rate limits, credential handling, and fallback behavior).

Planned follow-up: add a Phase 7 ADR describing registry availability checks and
their compliance constraints.
