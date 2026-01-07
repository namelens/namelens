---
last_updated: 2025-12-30
tool_version: 0.1.0
prompt_version: name-availability@1.0.0, domain-content@1.0.0
---

# Tesla Trademark Lesson

**Why domain availability doesn't equal usability**

This example demonstrates a common naming pitfall: assuming that if you can
register a domain, you can use the name.

## The Setup

```bash
$ namelens version --extended
namelens 0.1.0
Commit: 9c27abf
Built: 2025-12-29T16:24:01Z
Go: go1.25.1

Gofulmen: 0.1.8
Crucible: 0.2.27
```

## Scenario 1: "Is tesla available?"

A naive user might wonder: "Hey, is 'tesla' available for my new project?"

```bash
$ namelens check tesla --tlds=com,io,dev --expert
╭────────┬───────────┬────────────────┬──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ TYPE   │ NAME      │ STATUS         │ NOTES                                                                                                                                                            │
├────────┼───────────┼────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ domain │ tesla.com │ taken          │ exp: 2026-11-03T05:00:00Z; registrar: MarkMonitor Inc.                                                                                                           │
│ domain │ tesla.dev │ taken          │ exp: 2026-02-20T16:51:49.603Z; registrar: MarkMonitor Inc.                                                                                                       │
│ domain │ tesla.io  │ taken          │ source: whois                                                                                                                                                    │
│ expert │ ailink    │ risk: critical │ The name 'tesla' is unavailable with critical risks due to Tesla Inc.'s extensive global trademarks, active enforcement against squatters, and ongoing disputes. │
├────────┼───────────┼────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│        │           │ 0/3 AVAILABLE  │                                                                                                                                                                  │
╰────────┴───────────┴────────────────┴──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

**Result**: All domains taken. Expert analysis flags **critical risk** due to
Tesla Inc.'s trademarks.

No surprise there. But what about a creative variant?

## Scenario 2: "How about mycooltesla?"

"Okay, tesla is taken. But what if I add some words around it?"

```bash
$ namelens check mycooltesla --tlds=com,io,dev --expert
╭────────┬─────────────────┬───────────────┬───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ TYPE   │ NAME            │ STATUS        │ NOTES                                                                                                                                                             │
├────────┼─────────────────┼───────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ domain │ mycooltesla.com │ available     │                                                                                                                                                                   │
│ domain │ mycooltesla.dev │ available     │                                                                                                                                                                   │
│ domain │ mycooltesla.io  │ available     │ source: whois                                                                                                                                                     │
│ expert │ ailink          │ risk: high    │ No direct evidence of existing use for 'mycooltesla' across X handles, domains, or trademarks, but high risk due to incorporation of protected 'Tesla' trademark. │
├────────┼─────────────────┼───────────────┼───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│        │                 │ 3/3 AVAILABLE │                                                                                                                                                                   │
╰────────┴─────────────────┴───────────────┴───────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

**Result**: All domains available! But wait...

The expert analysis still flags **high risk**: "incorporation of protected
'Tesla' trademark."

This is the key insight: **domain availability does not equal trademark
safety**.

## Bonus: What's Actually on tesla.com?

When a domain is "taken," the next question is: what's there? Is it a parked
page (potentially acquirable) or an active product (no chance)?

The `domain-content` prompt investigates:

```bash
$ namelens check tesla --tlds=com --expert --expert-prompt=domain-content
╭────────┬───────────┬────────────────┬─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╮
│ TYPE   │ NAME      │ STATUS         │ NOTES                                                                                                                                                                                           │
├────────┼───────────┼────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│ domain │ tesla.com │ taken          │ exp: 2026-11-03T05:00:00Z; registrar: MarkMonitor Inc.                                                                                                                                          │
│ expert │ ailink    │ risk: critical │ tesla.com is the active official website of Tesla, Inc., featuring electric vehicles, solar energy, and clean energy products/services. Severe naming conflict due to established global brand. │
├────────┼───────────┼────────────────┼─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┤
│        │           │ 0/1 AVAILABLE  │                                                                                                                                                                                                 │
╰────────┴───────────┴────────────────┴─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────╯
```

The JSON output reveals more detail:

```json
{
  "ailink": {
    "summary": "tesla.com is the active official website of Tesla, Inc...",
    "likely_available": false,
    "risk_level": "critical",
    "insights": [
      "Fully active e-commerce and marketing site for Tesla's core products",
      "No parking or placeholder elements; high traffic and authority",
      "Historical note: Acquired by Elon Musk/Tesla in 2016 for significant sum"
    ],
    "mentions": [
      {
        "source": "web",
        "description": "Elon Musk spent $11M on tesla.com domain in 2016",
        "url": "https://www.namecheap.com/blog/musk-spent-11-million-on-tesla-domain/",
        "relevance": "high"
      }
    ]
  }
}
```

**Key insight**: This isn't a parked domain waiting to be bought. It's the
flagship web presence of a $500B+ company. The `domain-content` prompt correctly
identifies it as an active product site with zero acquisition potential.

Compare this to domains that _are_ parked (showing GoDaddy/Sedo pages) - those
might be negotiable. tesla.com is not.

## Lessons Learned

1. **Domain availability is just the first check** - You can register a domain,
   but that doesn't mean you can legally use it.

2. **Trademark law extends to confusingly similar names** - Adding
   prefixes/suffixes to a trademarked term doesn't make it safe. Companies like
   Tesla actively enforce against variations.

3. **Expert search catches what RDAP misses** - The `--expert` flag uses
   AI-powered search to find trademark conflicts, social media presence, and
   existing brand usage that domain checks alone would miss.

4. **MarkMonitor = Big Company Alert** - When you see "MarkMonitor Inc." as the
   registrar, that's a brand protection service used by major corporations.
   They're watching.

## Key Takeaway

Always use `--expert` when evaluating names for serious projects. The few
seconds of additional checking can save you from:

- Cease and desist letters
- Forced rebranding costs
- Legal fees
- Lost brand equity

---

_Example created with namelens v0.1.0. Expert analysis powered by xAI Grok with
live web/X search._
