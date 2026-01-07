# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project adheres to Semantic
Versioning.

## [Unreleased]

## [0.1.0] - 2026-01-04

### Added

- Domain availability checks via RDAP bootstrap with caching and rate limits
- Optional WHOIS/DNS fallback for TLDs without RDAP
- Registry checks for npm, PyPI, and GitHub
- Expert analysis via AILink prompt library (xAI/Grok driver)
- AILink provider instances with multi-credential selection and role routing
- Generate command for AI-powered naming ideation
- Phonetics and suitability analysis prompts
- Built-in profiles: startup, developer, website, minimal, web3
- Tri-state availability reporting (available/taken/unknown)
- Batch checks with table/json/markdown output formats
- CLI diagnostics: envinfo, doctor, and AILink connectivity checks
