---
last_updated: 2026-01-05
tool_version: 0.1.0
prompt_version: name-availability@1.0.0, domain-content@1.0.0, brand-proposal@1.0.0, name-phonetics@1.0.0, name-suitability@1.0.0
---

# How namelens Got Its Name

**A tool that used itself to find its own identity**

> **Living document**: This example was originally created at the "initial
> commit" state of namelens (December 2025). The January 2026 update below shows
> additional analysis capabilities now available. The original story is
> preserved as historical context.

This example documents the actual naming journey for this project - from
codename to final brand.

---

## January 2026 Update: The Story Gets Better

We originally used namelens to name itself. The tool worked - we picked the
name, registered the domains, and launched the project. Running the same check
today tells a different story:

```bash
$ namelens check namelens --profile=developer --expert --phonetics --suitability
╭────────┬──────────────┬────────────────┬───────────────────────────────────────────────────────────────────────╮
│ TYPE   │ NAME         │ STATUS         │ NOTES                                                                 │
├────────┼──────────────┼────────────────┼───────────────────────────────────────────────────────────────────────┤
│ domain │ namelens.app │ taken          │ exp: 2026-12-30; registrar: Dynadot LLC.                              │
│ domain │ namelens.com │ taken          │ exp: 2026-11-17; registrar: GoDaddy.com, LLC                          │
│ domain │ namelens.dev │ taken          │ exp: 2026-12-30; registrar: Dynadot LLC.                              │
│ domain │ namelens.io  │ taken          │ source: whois                                                         │
│ domain │ namelens.net │ available      │                                                                       │
│ domain │ namelens.org │ taken          │ exp: 2026-12-30; registrar: Dynadot Inc                               │
│ domain │ namelens.sh  │ taken          │ source: whois                                                         │
│ npm    │ namelens     │ available      │                                                                       │
│ pypi   │ namelens     │ available      │                                                                       │
│ github │ @namelens    │ taken          │ url: https://github.com/namelens                                      │
│ expert │ ailink       │ risk: low      │ No direct mentions as existing brand; highly available with low risks │
├────────┼──────────────┼────────────────┼───────────────────────────────────────────────────────────────────────┤
│        │              │ 3/10 AVAILABLE │                                                                       │
╰────────┴──────────────┴────────────────┴───────────────────────────────────────────────────────────────────────╯

Suitability Analysis:
  Overall: 98/100 (suitable)
  Risks: None identified
  Notes: Neutral to positive associations across all markets
```

**What changed**: The domains now show as "taken" - _by us_. The GitHub org is
ours. The tool worked.

### New Analysis: Phonetics Deep Dive

v0.1.0 adds AI-powered phonetics analysis. Here's what it says about our name:

```json
{
  "name": "namelens",
  "syllables": {
    "count": 2,
    "breakdown": "name-lens",
    "stress_pattern": "PRIMARY-secondary"
  },
  "pronunciation": {
    "ipa_primary": "/ˈneɪm.lɛnz/",
    "phone_test_score": 92,
    "phone_test_notes": "Easy to spell out verbally; clear and distinctive"
  },
  "typeability": {
    "overall_score": 88,
    "hand_alternation": "good",
    "muscle_memory": "easy",
    "common_typos": ["nameless (added 'a')", "namelen (dropped 's')"]
  },
  "cli_suitability": {
    "score": 90,
    "tab_completion": "distinctive",
    "notes": "Short, unique prefix 'namel'; ideal for repeated CLI use"
  },
  "overall_assessment": {
    "combined_score": 90,
    "recommendation": "Strong candidate for CLI tool naming",
    "strengths": [
      "Intuitive pronunciation",
      "Excellent typing ergonomics",
      "Memorable and brandable"
    ]
  }
}
```

**Key scores**:

- Phonetics: 92/100
- Typeability: 88/100
- CLI Suitability: 90/100
- Cultural Suitability: 98/100

The phonetics analysis confirms what we felt intuitively: `namelens` is easy to
say, easy to type, and works great as a CLI command. The "lens" suffix has
positive associations in developer tools (GitLens, etc.) without direct overlap.

### New Analysis: Brand Proposal

The `brand-proposal` prompt now provides richer context:

> "Highly recommended: 'namelens' is unique, memorable, and developer-friendly
> with no major conflicts identified, ideal for a CLI utility focused on name
> analysis or generation."

**Insights**:

- No existing developer tools or CLI utilities named 'namelens'
- GitLens prominence highlights 'lens' suffix appeal in dev tools
- Developer sentiment on X favors simple, insightful CLI tools

**Recommendations**:

- Secure namelens.dev and GitHub org 'namelens' ✅ (done)
- Run full trademark search ✅ (done)
- Claim social handles ✅ (done)

Mission accomplished. The tool that named itself continues to validate the
choice.

---

## Original Story (December 2025)

### The Codename

Every project starts with a working name. Ours was **namescout** - a
straightforward description of what the tool does: scout out whether a name is
available.

We built the MVP under this codename. It checked domains via RDAP, package
registries (npm, PyPI), and social handles (GitHub). Later we added expert AI
analysis to catch trademark conflicts that simple availability checks miss.

Then came the moment of truth: we pointed the tool at itself.

### The Discovery

```bash
$ namescout check namescout --tlds=com,io,dev --expert
╭────────┬───────────────┬───────────────┬─────────────────────────────────────────────────────────────╮
│ TYPE   │ NAME          │ STATUS        │ NOTES                                                       │
├────────┼───────────────┼───────────────┼─────────────────────────────────────────────────────────────┤
│ domain │ namescout.com │ taken         │ exp: 2026-10-01; registrar: Rebel Ltd                       │
│ domain │ namescout.dev │ available     │                                                             │
│ domain │ namescout.io  │ taken         │ source: whois                                               │
│ expert │ ailink        │ risk: low     │ No direct mentions or trademarks identified...              │
├────────┼───────────────┼───────────────┼─────────────────────────────────────────────────────────────┤
│        │               │ 1/3 AVAILABLE │                                                             │
╰────────┴───────────────┴───────────────┴─────────────────────────────────────────────────────────────╯
```

The .com was taken. But maybe it was just parked? We dug deeper with
`domain-content`:

```bash
$ namescout check namescout --tlds=com --expert --expert-prompt=domain-content
╭────────┬───────────────┬────────────────┬────────────────────────────────────────────────────────────────────────╮
│ TYPE   │ NAME          │ STATUS         │ NOTES                                                                  │
├────────┼───────────────┼────────────────┼────────────────────────────────────────────────────────────────────────┤
│ domain │ namescout.com │ taken          │ exp: 2026-10-01; registrar: Rebel Ltd                                  │
│ expert │ ailink        │ risk: critical │ namescout.com is an active domain registrar service with domain search,│
│        │               │                │ registration, and related tools; represents a strong naming conflict.  │
╰────────┴───────────────┴────────────────┴────────────────────────────────────────────────────────────────────────╯
```

**Critical risk.** Not parked - an _active domain registrar service_. A direct
competitor in the naming space.

Our own tool told us we couldn't use our own name.

### The Search

We generated alternatives. Dozens of candidates across different naming
patterns:

- **Descriptive**: checkname, namechecker, nameprobe
- **Metaphorical**: namelens, namewise, namelight
- **Abstract**: nomi, nomina, brandly
- **Compound**: namecraft, namesmith, brandscout

Each candidate went through the same gauntlet:

```bash
$ namescout check <candidate> --tlds=com,io,dev --expert
```

Most fell quickly:

- **.com taken by active products** - immediate disqualification
- **Trademark conflicts** - too risky
- **Awkward phonetics** - wouldn't work as a CLI command

Two survived: **namelens** and **namewise**.

### The Finalists

#### namelens

```bash
$ namescout check namelens --tlds=com,io,dev --expert
╭────────┬──────────────┬───────────────┬───────────────────────────────────────────────────────────────╮
│ TYPE   │ NAME         │ STATUS        │ NOTES                                                         │
├────────┼──────────────┼───────────────┼───────────────────────────────────────────────────────────────┤
│ domain │ namelens.com │ taken         │ exp: 2026-11-17; registrar: GoDaddy.com, LLC                  │
│ domain │ namelens.dev │ available     │                                                               │
│ domain │ namelens.io  │ available     │ source: whois                                                 │
│ expert │ ailink       │ risk: low     │ No direct mentions, trademarks, or social handles found.      │
├────────┼──────────────┼───────────────┼───────────────────────────────────────────────────────────────┤
│        │              │ 2/3 AVAILABLE │                                                               │
╰────────┴──────────────┴───────────────┴───────────────────────────────────────────────────────────────╯
```

The .com was taken but appeared parked:

```json
{
  "summary": "namelens.com has no indexed web content and appears to be a parked domain.",
  "likely_available": true,
  "risk_level": "low",
  "insights": [
    "Domain fits classic parked profile: registered but not developed",
    "Low SEO footprint suggests staleness or parking monetization"
  ]
}
```

Parked = acquirable. Low risk. Strong availability on .dev and .io.

#### namewise

Similar profile: .com parked, low risk, good secondary TLD availability.

Both names were viable. Time for a brand study using `brand-proposal`:

### The Decision

We analyzed both names against our target audience:

| Audience            | What They Want                 |
| ------------------- | ------------------------------ |
| **Developers**      | Precision tools, not advice    |
| **Architects**      | Comprehensive analysis         |
| **Entrepreneurs**   | Due diligence, risk visibility |
| **Marketing teams** | Fast, professional output      |

#### namelens

**Metaphor**: A lens focuses, clarifies, reveals what's there.

- Tool-oriented: You wield it
- Objective: It shows you what's there
- Technical: Fits developer mental model
- CLI natural: `namelens check acmecorp` flows

#### namewise

**Metaphor**: Wisdom about names. Being wise in selection.

- Advisor-oriented: It tells you things
- Prescriptive: Implies judgment
- Softer: Less technical edge

#### The Verdict

This tool **reveals** - it doesn't **prescribe**. Users look through it to see
clearly; they make their own decisions. That's a **lens**, not an **oracle**.

For technical users who want to see the data and decide for themselves,
**namelens** wins.

### The Name

**namelens** - a precision instrument for examining names.

### Lessons Learned

1. **Eat your own dogfood** - We used our own tool to name itself. It caught a
   critical conflict we might have missed.

2. **Domain availability != usability** - namescout.dev was available, but the
   .com was an active competitor. That's a dealbreaker.

3. **Parked vs active matters** - The `domain-content` prompt distinguishes
   between parked domains (acquirable) and active products (conflict).
   namelens.com was parked; namescout.com was not.

4. **Brand positioning matters** - Two viable names can have different brand
   implications. `brand-proposal` helped articulate why "lens" fit better than
   "wise" for our technical audience.

5. **The tool works** - This project's MVP goal was to use itself to choose its
   own name. Mission accomplished.

---

_This document was created using namelens v0.1.0 (under codename namescout). The
naming process used `name-availability`, `domain-content`, and `brand-proposal`
prompts._

---

**Accuracy note**: All search results and domain statuses were verified accurate
as of 2025-12-30. Domain ownership and availability may change over time.

**Disclaimer**: This project has no plans to use, register, or claim any of the
other name candidates evaluated during this process (namewise, namecraft,
nameprobe, etc.). They remain available for others to use.
