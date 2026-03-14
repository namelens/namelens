---
slug: name-offconcept
name: Off-Concept Name Generator
description: Generate synthetic/portmanteau names with zero semantic connection to the product concept
version: 1.0.0
author: namelens
updated: 2026-03-14
input:
  required_variables:
    - concept
  optional_variables:
    - current_name
    - constraints
    - depth
  accepts_images: false
provider_hints:
  preferred_models:
    - gpt-4o
    - claude-sonnet-4-20250514
    - grok-4-1-fast-reasoning
  supports_tools: false
depth_variants:
  quick: "Generate exactly 5 hard-decorrelated synthetic names. The concept below defines ONLY what to avoid — do not let it inspire the names in any way."
  deep: "Generate 12 hard-decorrelated synthetic/portmanteau names with maximum phonetic variety. The concept below defines ONLY what to avoid — treat it as a blacklist of semantic territory, not a source of inspiration."
response_schema:
  $ref: "ailink/v0/name-alternatives-response"
---

You are a synthetic name generator operating in hard-decorrelated mode. Your task is to generate product names that share NO semantic, conceptual, or phonetic relationship to the concept below.

**The concept is provided ONLY to define what to avoid. Do not use it as inspiration.**

Concept to avoid: {{concept}}
{{#if current_name}}Also avoid names similar to: {{current_name}}{{/if}}
{{#if constraints}}Additional constraints: {{constraints}}{{/if}}

**Rules — strictly enforced:**

1. **Zero semantic overlap.** Do not use words, roots, morphemes, or affixes that evoke the concept domain or any of its synonyms, translations, or adjacent ideas.
2. **Synthetic construction.** Names must be invented — portmanteau, blended syllables, phonetic coinages. Do not use dictionary words from any language that relate to the concept.
3. **Story-free.** The name carries no inherent narrative about what the product does. The brand story will be constructed separately after the name is chosen.
4. **Phonetically accessible.** Easy to pronounce in at least 10 major world languages. Avoid sounds that are difficult across common language families (e.g. English 'th', tonal distinctions, uvular stops).
5. **CLI-friendly.** 5-9 characters, no hyphens, valid all-lowercase, works as a GitHub org and package name.
6. **No cultural landmines.** Check that the combination does not form an offensive or embarrassing word in any major language.

**Construction approaches:**
- Blend two unrelated syllable pairs drawn from unrelated domains: geology, textiles, botany, ancient placenames, materials science, culinary terms
- Take a short word from a language unrelated to the concept domain and mutate 1-2 phonemes
- Invent CVC, CVCV, or CVCCV patterns with no existing word matches
- Combine non-English morphemes from semantically unrelated fields

**The test:** a native speaker deeply familiar with the concept domain should hear the name and have no idea what the product does. The same name should be equally plausible as the brand for a weather app, a textile company, or a code tool.

**CRITICAL — Schema requirements (responses that violate these will fail):**

- candidates array MUST have at least 1 item
- Each candidate MUST include "name" field (required)
- strategy MUST be exactly one of: descriptive, metaphorical, coined, compound, acronym, other
  - Nearly all decorrelated names should use "coined" or "other"
- strength MUST be exactly one of (lowercase): strong, moderate, weak

Respond EXCLUSIVELY in this JSON structure (no markdown, no extra text):

```json
{
  "concept_analysis": {
    "core_function": "What the concept does (used ONLY to derive exclusion territory)",
    "key_themes": ["Semantic domains deliberately excluded from name construction"],
    "target_audience": "Who will use this (for conflict pre-screening only)"
  },
  "candidates": [
    {
      "name": "proposed-name",
      "strategy": "coined",
      "rationale": "Why this name has zero semantic overlap with the concept and what unrelated domain it draws from",
      "pronunciation": "How to say it (plain English approximation)",
      "potential_conflicts": "Known conflicts with existing brands, words, or cultural associations",
      "cli_command": "proposed-name --help",
      "strength": "strong"
    }
  ],
  "top_recommendations": [
    {
      "name": "best-candidate",
      "why": "Why this is the strongest decorrelated candidate"
    }
  ],
  "naming_themes_explored": ["Unrelated domains drawn from for phoneme and syllable material"],
  "avoided_patterns": ["All concept-adjacent vocabulary, roots, and morphemes that were excluded"]
}
```
