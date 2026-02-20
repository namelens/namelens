---
description: Practical workflows and provider selection guidance for effective name discovery
---

# Workflows & Best Practices

**A practical guide to getting the most out of NameLens**

NameLens supports multiple AI providers, each with different strengths. This
guide helps you choose the right tool for your naming workflow and avoid common
pitfalls.

## Understanding the Providers

NameLens integrates with three AI providers, each optimized for different tasks:

| Provider      | Model (Default)            | Best For                               | Key Strength                            |
| ------------- | -------------------------- | -------------------------------------- | --------------------------------------- |
| **xAI**       | grok-4-1-fast-reasoning    | `check --expert`, real-time validation | Live web/X search for current conflicts |
| **Anthropic** | claude-sonnet-4-6          | `generate`, deep analysis              | Technical accuracy, conflict awareness  |
| **OpenAI**    | gpt-4o                     | `generate`, quick iteration            | Speed, brandable suggestions            |

### Why the Differences Matter

**xAI** has exclusive access to real-time web and X (Twitter) search. When you
run `check --expert`, it can discover:

- Companies that launched last month
- Active GitHub repos without SEO presence
- X handles and recent brand mentions
- Domain parking pages that look available but aren't

**Anthropic** excels at structured reasoning. For `generate`, it provides:

- Deep technical analysis of your concept
- Awareness of existing tools in the space
- Conservative naming that avoids conflicts
- Thorough explanations of why names work

**OpenAI** prioritizes speed and creativity:

- Fastest response times (often 10-15s vs 30-40s)
- Creative, brandable compound words
- Good for initial brainstorming when you need many options quickly

**Important**: All three can suggest names that have conflicts. The difference
is in _how_ they catch them and _what kind_ of conflicts they prioritize.

## Recommended Workflows

### Workflow A: Quick Evaluation (5-10 minutes)

Use this when you have a name idea and want fast validation.

```bash
# Step 1: Quick availability check
namelens check myproject --profile=startup

# Step 2: If domains available, run expert analysis
namelens check myproject --profile=startup --expert
```

**Why this works**: The basic check uses RDAP (fast, no AI). Only if domains are
available do you spend AI credits on expert analysis. This is efficient for
quick triage.

### Workflow B: Full Discovery (15-30 minutes)

Use this when starting from scratch with just a concept.

```bash
# Step 1: Generate with multiple providers (parallel mental comparison)
# Run each separately and compare mentally:

# Fast iteration with OpenAI
namelens generate "API gateway for microservices" --depth quick

# Technical depth with Anthropic
namelens generate "API gateway for microservices" --depth quick

# Step 2: Check top 3-5 picks with xAI expert mode
namelens check gateway1 gateway2 gateway3 --expert --expert-bulk
```

**Provider routing tip**: Set up your `config.yaml` to route automatically:

```yaml
ailink:
  routing:
    name-alternatives: namelens-openai  # Fast generation
    name-availability: namelens-xai     # Expert validation
```

### Workflow C: Context-Driven Naming (20-40 minutes)

Use this when you have existing documentation, design docs, or a project
directory.

```bash
# Step 1: Gather context from your project
namelens context ./my-project --output=markdown > context.md

# Review and edit context.md if needed, then:

# Step 2: Generate with rich context
namelens generate "developer tool" -f context.md --depth deep

# Step 3: Bulk check the top candidates
namelens check cand1 cand2 cand3 cand4 cand5 \
  --tlds com,io,dev --expert --expert-bulk
```

**Why context matters**: The more NameLens knows about your project, the better
the suggestions. A tool for "local development" vs "production-grade
orchestration" should have very different names.

### Workflow D: Multi-Pass Refinement (ongoing)

Use this when your initial names all have conflicts.

```bash
# Pass 1: Generate broad set
namelens generate "concept" --depth quick > round1.txt

# Identify themes that worked, then:

# Pass 2: Constrain to promising direction
namelens generate "concept" -d "Focus on weaving/connection metaphors, 6-8 letters" \
  --depth deep

# Pass 3: Check final candidates
namelens check pick1 pick2 pick3 --expert-bulk
```

## Common Pitfalls & How to Avoid Them

### Pitfall 1: Trusting Domain Availability Alone

**The Mistake**: "The .com is available, so I'm good to go!"

**Reality Check**: See our [Tesla Trademark Lesson](tesla-trademark-lesson.md)
for a humorous but real example of why this fails.

**The Fix**: Always use `--expert` for serious projects. The AI will check:

- Trademark databases (USPTO, EUIPO)
- Social media handles
- GitHub/npm/PyPI registry presence
- Recent company launches

### Pitfall 2: Ignoring TLD Strategy

**The Mistake**: Only checking .com and giving up when it's taken.

**Better Approach**: Consider your audience:

| Audience          | Priority TLDs         |
| ----------------- | --------------------- |
| General consumers | .com, .app            |
| Developers/tech   | .io, .dev, .sh        |
| Web3/crypto       | .xyz, .gg             |
| Enterprise        | .com + relevant ccTLD |

**Example**: `myproject.io` being taken is more concerning for a dev tool than
`myproject.com` being taken by a parked domain.

### Pitfall 3: Not Checking Registries

**The Mistake**: "The domain is available, so the npm package name must be too."

**Reality**: Domain availability and package registry availability are
independent. Many great .com domains have squatters on npm.

**The Fix**: Use profiles that include registries:

```bash
# Includes npm, pypi, cargo
namelens check myproject --profile=startup

# Or specify explicitly
namelens check myproject --registries=npm,pypi
```

### Pitfall 4: Generating Without Checking

**The Mistake**: Running `generate` once and picking the first suggestion.

**The Risk**: AI models can hallucinate availability. A name that sounds great
might be taken by a company that launched last week.

**The Fix**: The `generate` → `check` loop is essential. Never skip validation.

## Provider-Specific Tips

### Getting the Most from xAI (Expert Mode)

**When to use**: Any `check --expert` operation

**Strengths**:

- Real-time web search finds companies that launched yesterday
- X search finds active projects without websites
- Fast (usually under 30s for single names)

**Limitations**:

- Suggestions can be less creative than Anthropic/OpenAI
- No deep reasoning about metaphor quality

**Pro tip**: For bulk checks, use `--expert-bulk` to check multiple names in one
AI call:

```bash
# One AI call, 5 names (much cheaper than 5 separate calls)
namelens check name1 name2 name3 name4 name5 --expert --expert-bulk
```

### Getting the Most from Anthropic (Generation)

**When to use**: When you need technical accuracy or conflict avoidance

**Strengths**:

- Notices when your concept overlaps with existing tools
- Suggests conservative, defensible names
- Excellent at analyzing why names work (or don't)

**Limitations**:

- Slower than OpenAI (20-40s vs 10-15s)
- Can be overly conservative (may reject good names due to distant conflicts)

**Pro tip**: Anthropic respects detailed constraints. Use `-d` for specificity:

```bash
namelens generate "developer tool" \
  -d "Avoid anything that sounds like 'Docker' or 'Kubernetes'. Prefer short, CLI-friendly names under 8 letters. Target audience is senior platform engineers."
```

### Getting the Most from OpenAI (Generation)

**When to use**: Initial brainstorming when you need many options fast

**Strengths**:

- Fastest generation (often under 15s)
- Creative compound words and metaphors
- Good at brandable, memorable suggestions

**Limitations**:

- Less thorough conflict checking during generation
- Can suggest names that sound good but have hidden conflicts

**Pro tip**: Use OpenAI for volume, then validate with xAI:

```bash
# Generate 10+ ideas quickly with OpenAI
namelens generate "concept" --depth quick

# Check the best 3 with xAI expert mode
namelens check best1 best2 best3 --expert
```

## Cost & Performance Optimization

### Reducing AI Calls

**The expensive operations** (in order):

1. `check --expert` per name (1 AI call per name)
2. `generate --depth deep` (1 AI call)
3. `check --expert-bulk` (1 AI call for N names - most efficient)
4. `generate --depth quick` (1 AI call, faster)

**Optimization strategies**:

1. **Use bulk expert**: Check multiple names in one call

   ```bash
   # 1 call for 5 names (80% savings)
   namelens check a b c d e --expert --expert-bulk
   ```

2. **Filter with basic checks first**: Don't waste AI credits on names with no
   available domains

   ```bash
   # Check domains first (no AI cost)
   namelens check myname --tlds com,io

   # Only if domains available, run expert
   namelens check myname --expert
   ```

3. **Use concurrency for multiple names**: Speed up the non-AI parts
   ```bash
   # Check 10 names in parallel (domains, registries)
   namelens check name1 name2 ... name10 --concurrency 5
   ```

### Understanding Speed vs Quality

| Command             | Typical Time | Quality                       |
| ------------------- | ------------ | ----------------------------- |
| `check` (no expert) | 1-5s         | Basic (domains only)          |
| `check --expert`    | 20-40s       | High (web search + analysis)  |
| `generate --quick`  | 10-20s       | Good (fast brainstorming)     |
| `generate --deep`   | 30-60s       | Excellent (thorough analysis) |

**Rule of thumb**: Start fast (`--quick`, no expert), then invest time in
validation for your top 3-5 picks.

## Examples by Use Case

### Use Case: Open Source CLI Tool

**Goal**: Find a name for a new open source developer tool

```bash
# Generate with technical focus
namelens generate "CLI tool for managing local development environments" \
  --prompt name-alternatives \
  --depth deep

# Check availability across dev-focused TLDs
namelens check devloom envforge kitpulse \
  --tlds com,io,dev,sh \
  --registries npm \
  --expert --expert-bulk
```

### Use Case: SaaS Product

**Goal**: Find a brandable name for a B2B SaaS product

```bash
# Generate with brand focus
namelens generate "B2B SaaS for project management" \
  -d "Target audience: mid-market companies. Tone: professional but modern. Avoid generic terms like 'task' or 'project'." \
  --depth deep

# Check with startup profile (broad TLD coverage)
namelens check stratum unigress planebridge \
  --profile=startup \
  --expert --expert-bulk
```

### Use Case: Internal Tool (No Public Domain Needed)

**Goal**: Name an internal company tool that won't have a public website

```bash
# Generate without domain pressure
namelens generate "Internal deployment automation tool" \
  -d "Used only internally. No public domain needed. Focus on memorable and CLI-friendly."

# Just check npm/github for conflicts
namelens check deploybot shipit launchpad \
  --tlds dev \
  --registries npm \
  --handles github
```

## When to Escalate to Legal

NameLens provides **risk indicators**, not legal advice. Consult an attorney
when:

- Expert analysis shows **critical risk** for a name you want to use
- The name is similar to a trademark in your **industry/category**
- You plan to **trademark** the name yourself (clearance search required)
- The project has **significant investment** or revenue potential

**Remember**: NameLens flags risks; it doesn't certify safety. A "low risk"
result from expert analysis is an indicator, not a guarantee.

## Summary Checklist

Before finalizing a name:

- [ ] Generated options from at least one provider
- [ ] Checked top 3-5 candidates with `--expert`
- [ ] Verified relevant TLDs (not just .com)
- [ ] Checked package registries (npm, pypi, cargo)
- [ ] Checked social handles (GitHub)
- [ ] Considered trademark risk level
- [ ] For commercial projects: consulted legal counsel

## Further Reading

- [Quick Start Guide](quick-start.md) - First-time setup
- [Expert Search Guide](expert-search.md) - Deep dive on AI validation
- [Tesla Trademark Lesson](tesla-trademark-lesson.md) - Real-world conflict
  example
- [Configuration Guide](configuration.md) - Provider setup and routing
- [Generate Command](generate.md) - Detailed generation options

---

_This guide reflects testing with namelens v0.2.1. Provider capabilities and
response times may evolve as underlying models improve._
