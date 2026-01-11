# Brand Guide

Using Namelens for competitive research, brand gap analysis, and strategic naming.

---

## Competitive Intelligence

### Research Competitor Naming Patterns

Understand how competitors in your space name their products:

```bash
# Check multiple competitors
namelens check competitor-a --tlds=com,io --expert
namelens check competitor-b --tlds=com,io --expert
namelens check competitor-c --tlds=com,io --expert
```

**What to analyze:**

| Aspect | What to Look For | How Namelens Helps |
|--------|-----------------|-------------------|
| TLD preference | .com vs .io vs .dev | Domain availability by TLD |
| Naming patterns | Descriptive vs metaphorical | Expert prompt: brand-proposal |
| Trademark filings | Protected brands | Trademark conflict detection |
| Social presence | Active communities | Handle availability |
| Web sentiment | Customer opinions | Expert sentiment analysis |

### Find Brand Gaps

Identify naming opportunities your competitors have missed:

1. **Analyze competitor names**
   ```bash
   namelens batch competitors.txt --expert
   ```

2. **Generate alternatives in similar space**
   ```bash
   namelens generate "AI productivity tool" --expert
   ```

3. **Check for untapped patterns**
   - Are they all descriptive? Try metaphorical names.
   - Do they all use .com? Consider .dev/.app positioning.
   - Are names long and complex? Try shorter, punchier alternatives.

---

## Brand Alignment Audits

### Audit Current Brand Assets

Check consistency across your brand portfolio:

```bash
# Check your product
namelens check myproduct --profile=startup --expert

# Check related tools/sidecars
namelens check myproduct-cli
namelens check myproduct-pro
namelens check myproduct-cloud

# Check partner/integration names
namelens check myproduct-for-x
```

**What you're auditing:**
- Domain ownership consistency
- Handle availability
- Trademark risk alignment
- Naming convention adherence

### Brand Portfolio Management

Track multiple related products:

```bash
# Create portfolio file
cat > portfolio.txt <<EOF
core-product
core-product-cli
core-product-pro
core-product-cloud
core-product-integration
EOF

# Batch audit
namelens batch portfolio.txt --output=json > portfolio-audit.json
```

Use the JSON output to:
- Identify gaps (missing domains or handles)
- Spot inconsistencies (some .com, some .io)
- Prioritize registration tasks
- Generate portfolio reports for stakeholders

---

## Product Launch Preparation

### Pre-Launch Risk Assessment

Before going public, verify:

```bash
namelens check yourproduct \
  --profile=startup \
  --expert \
  --expert-prompt=domain-content \
  --phonetics \
  --suitability \
  --locales=en-US,es-ES,fr-FR,de-DE
```

**Launch readiness checklist:**

- [ ] Low trademark risk (expert analysis)
- [ ] .com secured or acquirable
- [ ] Key social handles available
- [ ] Strong phonetics score (85+)
- [ ] Cultural suitability verified (90+ in target markets)
- [ ] No negative web sentiment
- [ ] Clear proceed recommendation

### Competitive Positioning

Use expert analysis to position against competitors:

```bash
# Get brand proposal for positioning insights
namelens check yourproduct \
  --expert \
  --expert-prompt=brand-proposal \
  --expert-depth=deep
```

**What you get:**
- Target audience analysis
- Differentiation opportunities
- Messaging recommendations
- Competitive landscape assessment

---

## Rebranding & Mergers

### Pre-Rebrand Due Diligence

Before rebranding, verify new candidate:

```bash
namelens check new-brand-name \
  --profile=startup \
  --expert \
  --phonetics \
  --suitability \
  --expert-prompt=brand-proposal
```

**Critical checks for rebranding:**
- Trademark clearance (higher risk tolerance for new brand)
- Domain availability (.com non-negotiable for most companies)
- Social handle portability (can you migrate?)
- URL redirects (can you secure old domains for redirects?)
- Cultural fit in all existing markets

### M&A Brand Integration

When acquiring or merging companies:

```bash
# Check surviving brand
namelens check surviving-brand --profile=startup --expert

# Check acquired brands
namelens check acquired-brand-a --profile=startup --expert
namelens check acquired-brand-b --profile=startup --expert

# Check potential combined names
namelens check combined-brand-name --profile=startup --expert
```

**M&A considerations:**
- Which brand has better domain/ handle portfolio?
- Which has lower trademark risk?
- Can domains be consolidated?
- Which name scales better post-merger?

---

## International Markets

### Multi-Market Suitability

Before expanding to new regions:

```bash
namelens check yourproduct \
  --suitability \
  --locales=en-US,ja-JP,ko-KR,zh-CN,de-DE,fr-FR,es-ES,pt-BR
```

**What to evaluate:**
- Pronunciation difficulty
- Cultural appropriateness
- Negative associations (slang, offensive meanings)
- Legal restrictions (country-specific naming laws)

### Local Domain Strategy

Research ccTLD availability:

```bash
# Check local domains for target markets
namelens check yourproduct --tlds=co.uk,de,fr,jp,kr,cn,br
```

**Strategy considerations:**
- Priority: Secure .com + major market ccTLDs
- Redirect strategy: ccTLD → .com or local landing pages
- Local naming: Consider localized variants if global name has issues

---

## Brand Monitoring (Coming Soon)

Future releases will include:

- **Domain expiration alerts** — Monitor competitor domains
- **Trademark filing notifications** — Know when new trademarks are filed
- **Social handle tracking** — Spot squatting early
- **Sentiment monitoring** — Track brand mentions over time
- **Competitor name launches** — Get alerts when competitors rebrand

---

## Case Studies

### [The Tesla Trademark Lesson](../examples/tesla-trademark-lesson.md)

Shows why domain availability ≠ brand safety, and how expert analysis catches conflicts simple checks miss.

### [Namelens Origin Story](../examples/namelens-origin-story.md)

Full branding journey from codename to final brand, including competitor analysis and brand proposal.

---

## Tips for Marketing Professionals

1. **Start early** — Don't wait until creative brief phase. Use Namelens during strategy development.

2. **Batch your research** — Check multiple candidates simultaneously for efficiency.

3. **Deep dive finalists** — Reserve `--expert` for 3-5 finalists, not every candidate.

4. **Document your process** — Save outputs for stakeholder presentations and audit trails.

5. **Iterate quickly** — Namelens is fast; run multiple rounds of checks as naming evolves.

6. **Involve legal early** — Use Namelens to narrow candidates, then have trademark counsel do final clearance.

7. **Think globally** — Even if launching locally, consider international expansion potential.

---

## Need Help?

```bash
namelens check --help
namelens batch --help
namelens doctor  # Troubleshooting
```

See also:
- [Startup Naming Guide](startup-guide.md) — End-to-end venture naming
- [Expert Analysis](expert-search.md) — Deep research capabilities
- [Batch Processing](batch.md) — Compare multiple names
