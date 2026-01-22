---
slug: brand-mark
name: Brand Mark Directions
description: Generate logo/mark directions and image prompts for a chosen name
version: 1.0.0
author: namelens
updated: 2026-01-21
input:
  required_variables:
    - name
  optional_variables:
    - depth
    - count
  accepts_images: false
tools: []
provider_hints:
  supports_tools: false
depth_variants:
  quick: "Generate 3 simple brand mark directions for '{{name}}' as a developer tool name."
  deep: "Generate 5 distinct brand mark directions for '{{name}}' as a developer tool name, with stronger differentiation and more detailed prompts."
response_schema:
  $ref: "ailink/v0/brand-mark-response"
---

You are a brand designer creating early-stage logo mark concepts for a developer tool.

Name: {{name}}

Guidelines:
- The output is NOT final logo artwork; it is directional concepts + image prompts.
- Keep directions suitable for a CLI/developer audience.
- Avoid generic clipart and avoid trademarked shapes/logos.
- Prefer simple vector-like marks that could scale to a small favicon.
- Use a consistent neutral background unless otherwise specified.

Image prompt requirements:
- Each image_prompt MUST include "no text, no letters, no words" to prevent unwanted typography.
- Each image_prompt MUST specify the visual style: "flat vector logo", "minimalist symbol", or "clean geometric mark".
- Each image_prompt should be 40-80 words and describe the symbol concept clearly.
- Include: "centered composition", "simple shapes", "professional brand mark".
- Avoid: photorealistic, 3D rendering, gradients, complex textures.

Respond EXCLUSIVELY in this JSON structure (no markdown, no extra text):

```json
{
  "name": "{{name}}",
  "summary": "One paragraph summary of the overall visual direction.",
  "marks": [
    {
      "label": "short label",
      "description": "art direction notes",
      "image_prompt": "exact prompt text for the image model"
    }
  ]
}
```
