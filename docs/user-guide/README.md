# NameLens User Guide

This guide covers installation, configuration, and usage of NameLens.

> **Disclaimer**: This tool is for informational purposes only and does not
> constitute legal or professional advice. See
> [USAGE-NOTICE.md](../USAGE-NOTICE.md) before making business decisions based
> on results.

## Contents

| Document                              | Description                                              |
| ------------------------------------- | -------------------------------------------------------- |
| [Configuration](configuration.md)     | Layered configuration, environment variables, validation |
| [Domain Fallback](domain-fallback.md) | Whois and DNS fallback for TLDs without RDAP             |
| [Expert Search](expert-search.md)     | AI-powered brand availability analysis                   |
| [Name Generation](generate.md)        | Generate naming candidates from a product concept        |

## Quick Start

```bash
# Generate name candidates from a concept
namelens generate "static analyzer for shell scripts"

# Check a name across default TLDs
namelens check myproject

# Check with specific TLDs
namelens check myproject --tlds=com,io,sh

# Check with expert AI analysis
namelens check myproject --expert

# Check with phonetics and suitability analysis
namelens check myproject --phonetics --suitability

# View current configuration
namelens envinfo

# Run diagnostics
namelens doctor

# Initialize a default config
namelens doctor init
```

## Getting Help

```bash
namelens --help
namelens generate --help
namelens check --help
namelens doctor
```

## Related Documentation

- [AGENTS.md](../../AGENTS.md) - AI agent guidelines
- [Architecture](../architecture/) - System design decisions
- [Operations](../operations/) - Build and deployment
