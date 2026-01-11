# Startup Naming Guide

End-to-end guidance for naming your venture using Namelens.

---

## Phase 1: Ideation (0-2 days)

### Generate Candidates

Start with your product concept. Let Namelens generate initial ideas:

```bash
namelens generate "AI-powered developer productivity tool"
```

This returns 10-20 name candidates across different naming patterns
(descriptive, metaphorical, abstract, compound).

### Expand Your List

Add your own ideas to the mix. Aim for 15-30 candidates total:

```bash
# Create a candidates file
cat > candidates.txt <<EOF
acmecorp
stellaplex
fluxio
codepulse
devspark
EOF
```

### Initial Batch Check

Quick availability scan to eliminate obvious conflicts:

```bash
namelens batch candidates.txt --output=table --available-only
```

Focus on candidates with reasonable availability scores.

---

## Phase 2: Deep Research (1-3 days)

### Select 3-5 Finalists

From your initial batch, choose 3-5 strong candidates for deep analysis.

### Run Expert Analysis

For each finalist, run comprehensive checks:

```bash
namelens check acmecorp \
  --profile=startup \
  --expert \
  --phonetics \
  --suitability \
  --locales=en-US,es-ES,de-DE
```

**What you're looking for:**

- **Low trademark risk** — "risk: low" in expert output
- **Available .com** — Or clearly acquirable (parked)
- **Strong scores** — Phonetics 85+, Suitability 90+ across target markets
- **Clean web presence** — No negative sentiment or existing products

### Analyze Competitors

Use the expert analysis to understand competitive landscape:

```bash
# Check what's actually on taken .com domains
namelens check acmecorp.com --expert --expert-prompt=domain-content
```

This tells you if a taken domain is:

- **Parked** — Usually acquirable for reasonable price
- **Active product** — Conflict; move on
- **Placeholder** — Low-value use; negotiate or move on

---

## Phase 3: Decision (1 day)

### Compare Finalists

Create a comparison table:

| Candidate  | .com Status         | Trademark Risk | Phonetics | Suitability | Notes            |
| ---------- | ------------------- | -------------- | --------- | ----------- | ---------------- |
| acmecorp   | Parked (acquirable) | Low            | 92/100    | 96/100      | Strong contender |
| stellaplex | Available           | Low            | 88/100    | 94/100      | Easy to type     |
| fluxio     | Active conflict     | High           | 85/100    | 82/100      | Risky            |

### Get Brand Proposal

For your top choice, request a brand proposal:

```bash
namelens check acmecorp --expert --expert-prompt=brand-proposal --expert-depth=deep
```

This provides:

- Brand positioning recommendations
- Target audience analysis
- Messaging angles
- Launch considerations

### Make Your Choice

Based on:

1. **Availability** — Can you secure the domains and handles?
2. **Risk level** — Are there trademark or competitive conflicts?
3. **Fit** — Does the name resonate with your audience?
4. **Growth potential** — Can this name scale with your product?

---

## Phase 4: Execution (Day 1-7)

### Secure Domains Immediately

```bash
# Verify one last time
namelens check acmecorp --tlds=com,io,dev,app --profile=startup

# Then register through your preferred registrar
```

**Recommended order:**

1. .com (priority)
2. .io or .dev (secondary)
3. .app (if app-focused)
4. Any misspelling variants (prevent typosquatting)

### Claim Social Handles

```bash
namelens check acmecorp --handles=github
```

More platforms coming soon. For now, manually claim:

- GitHub org
- X/Twitter
- LinkedIn company page
- Product Hunt (if applicable)

### File Trademark (Optional but Recommended)

Namelens does not constitute legal advice. Consult trademark counsel for:

- USPTO registration
- International trademark protection
- Service mark coverage

**Pro tip**: Namelens can help identify potential conflicts _before_ you file,
saving attorney time and filing fees.

### Internal Alignment

Document your naming decision:

```
Name: Acme Corp
Domains: acmecorp.com (parked, negotiating), acmecorp.io (registered)
Handles: @acmecorp (GitHub, X)
Trademark search: Low risk identified via Namelens
Decision rationale: [from brand-proposal]
Next steps: [registration timeline, launch plan]
```

Share this with co-founders, legal counsel, and marketing team.

---

## Phase 5: Launch Prep

### Verify Everything Again

```bash
namelens check acmecorp --profile=startup --expert
```

Final verification before:

- Printing business cards
- Launching website
- Incorporating business
- Announcing publicly

### Monitor Your Name (Coming Soon)

Future versions of Namelens will support:

- Domain expiration alerts
- Competitor name monitoring
- Trademark filing notifications
- Social handle squat detection

## Timeline Summary

| Phase     | Duration      | Key Actions                                   |
| --------- | ------------- | --------------------------------------------- |
| Ideation  | 0-2 days      | Generate, batch check, select candidates      |
| Research  | 1-3 days      | Expert analysis, competitor research          |
| Decision  | 1 day         | Compare finalists, brand proposal             |
| Execution | 1-7 days      | Secure domains, claim handles, file trademark |
| **Total** | **3-14 days** | From concept to secured name                  |

---

## Common Startup Pitfalls

### Pitfall 1: Falling in Love Too Early

**Don't**: Commit to a name before checking availability.

**Do**: Start with 15-30 candidates; let availability guide your choice.

### Pitfall 2: Ignoring .com

**Don't**: Assume .io or .dev is sufficient.

**Do**: .com is expected by investors and customers. If taken, evaluate if it's
parked (acquirable) or an active conflict (move on).

### Pitfall 3: Skipping Trademark Research

**Don't**: Assume "available" means "legally safe."

**Do**: Run `--expert` to identify trademark conflicts before investing.

### Pitfall 4: Forgetting Social Handles

**Don't**: Lock in the name before checking @handle availability.

**Do**: Check GitHub, X, and LinkedIn during research phase.

### Pitfall 5: Waiting Until Too Late

**Don't**: Wait until printing business cards or filing LLC paperwork.

**Do**: Run Namelens checks during ideation, not execution.

---

## Real-World Example

[Read the full Namelens origin story](../examples/namelens-origin-story.md) to
see how we applied this process to name this tool—including the critical
conflict we caught on our original codename.

---

## Need Help?

```bash
namelens generate --help
namelens check --help
namelens batch --help
namelens doctor  # Troubleshooting
```

See also:

- [Quick Availability Check](quick-start.md) — Basic checks
- [Expert Analysis](expert-search.md) — Deep research
- [Batch Processing](batch.md) — Compare candidates
