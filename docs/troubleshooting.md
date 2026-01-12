# Troubleshooting Guide

This guide covers common issues, symptoms, and solutions for NameLens usage.

## Table of Contents

- [Domain Availability Issues](#domain-availability-issues)
  - [RDAP Errors](#rdap-errors)
  - [DNS Resolution Failures](#dns-resolution-failures)
  - [Cache Issues](#cache-issues)
- [Output and File Issues](#output-and-file-issues)
  - [Output Sinks Not Working](#output-sinks-not-working)
  - [JSON Parsing Errors](#json-parsing-errors)
- [Expert Backend Issues](#expert-backend-issues)
  - [AILink 503 Errors](#ailink-503-errors)
  - [Missing API Keys](#missing-api-keys)
- [Configuration Issues](#configuration-issues)
  - [Profiles Not Loading](#profiles-not-loading)
  - [Environment Variables](#environment-variables)
- [Build and Installation](#build-and-installation)

---

## Domain Availability Issues

### RDAP Errors

**Symptom:** All `.app` and `.dev` domain checks return `available: 3` (error) with message "No RDAP servers responded successfully (tried 1 server(s))".

**Diagnosis:**
```bash
# Check RDAP server DNS resolution
nslookup rdap.nic.google
# Result: NXDOMAIN (server not found)
```

**Root Causes:**

1. **Network blocking** — Corporate firewalls or proxies may block outbound HTTPS to `rdap.nic.google`
2. **DNS filtering** — Some networks filter or block certain RDAP endpoints
3. **Server outage** — Google's RDAP service is temporarily unavailable

**Solutions:**

1. **Test direct access:**
   ```bash
   # Test if you can reach Google's RDAP API directly
   curl -s "https://pubapi.registry.google/rdap/domain/example.app"
   ```

2. **Check firewall rules:** Verify outbound HTTPS (port 443) to Google Registry is allowed
3. **Use fallback mechanism:** NameLens now implements multi-server fallback:
   - Primary: `https://pubapi.registry.google/rdap`
   - Fallback: `https://www.rdap.net/rdap`
   - If primary fails, NameLens automatically retries with fallback

4. **Wait and retry:** If service is temporarily unavailable, try again in 1-2 minutes

**Version Info:** Fixed in v0.1.3 — See [docs/known-issues.md](known-issues.md) for version-specific changes.

---

### DNS Resolution Failures

**Symptom:** Domain checks fail immediately or return "server not found" errors.

**Diagnosis:**
```bash
# Test DNS resolution
nslookup rdap.nic.google
nslookup rdap.google
nslookup www.rdap.net
```

**Root Causes:**

1. **DNS server not responding** — Local DNS resolver may be misconfigured
2. **Network blocking** — ISP or corporate network blocking DNS queries
3. **TTL caching** — Old DNS records cached locally

**Solutions:**

1. **Check DNS configuration:**
   ```bash
   # macOS
   scutil --dns
   # Linux
   systemd-resolve --status
   # Or check /etc/resolv.conf
   ```

2. **Flush local DNS cache:**
   ```bash
   # macOS
   sudo dscacheutil -flushcache; sudo killall -HUP mDNSResponder
   # Linux
   sudo systemd-resolve --flush-caches
   ```

3. **Use alternative DNS:** Try using Google DNS (8.8.8.8) or Cloudflare DNS (1.1.1.1)
   ```bash
   # Test with specific DNS server
   namelens check myname --verbose 2>&1 | grep "rdap"
   ```

---

### Cache Issues

#### Cache Returning Stale Data

**Symptom:** Results show `from_cache: true` but contain old information (expired domains, wrong status).

**Diagnosis:**
```bash
# Check cache entry
sqlite3 ~/.config/namelens/namelens.db "SELECT name, tld, expires_at FROM check_cache WHERE check_type='domain' ORDER BY expires_at DESC LIMIT 5"
```

**Root Cause:** Cache TTL is too long or cache was not invalidated after configuration changes.

**Solution:** Force cache refresh:
```bash
namelens check myname --no-cache
```

Or clear cache entirely:
```bash
rm ~/.config/namelens/namelens.db
```

#### Cache Not Persisting

**Symptom:** Running `namelens check` twice shows different results each time, suggesting cache isn't working.

**Diagnosis:**
```bash
# Check cache file exists and is writable
ls -la ~/.config/namelens/namelens.db

# Check database integrity
sqlite3 ~/.config/namelens/namelens.db "PRAGMA integrity_check"
```

**Root Causes:**

1. **File permissions** — Database directory or file not writable
2. **Disk full** — No space to write cache entries
3. **Corrupted database** — Previous crash or write error corrupted the file

**Solutions:**

1. **Fix permissions:**
   ```bash
   # Ensure cache directory exists with correct permissions
   mkdir -p ~/.config/namelens
   chmod 755 ~/.config/namelens
   chmod 644 ~/.config/namelens/namelens.db
   ```

2. **Recreate cache:**
   ```bash
   # Remove corrupted cache
   rm ~/.config/namelens/namelens.db
   # Namelens will recreate on next run
   ```

---

## Output and File Issues

### Output Sinks Not Working

**Symptom:** Using `--out /path` or `--out-dir /path` flags doesn't create files.

**Diagnosis:**
```bash
# Test with verbose logging
namelens check myname --out /tmp/result.json --verbose 2>&1 | grep -E "sink|writer|Output"
```

**Root Causes:**

1. **Stale binary** — You're running an old `namelens` binary that doesn't have the new output sink features
2. **Path permissions** — Target directory doesn't exist or isn't writable
3. **Permission denied** — User doesn't have write access to the specified path

**Solutions:**

1. **Rebuild and reinstall:**
   ```bash
   make build
   make install
   ```

2. **Verify new binary:**
   ```bash
   # Check for new flags in help
   namelens check --help | grep -E "out|out-dir|output-format"
   ```

3. **Check directory permissions:**
   ```bash
   # Verify directory exists and is writable
   ls -ld /tmp
   touch /tmp/test-write.txt
   ```

4. **Test with simple path:**
   ```bash
   # Test with current working directory
   namelens check myname --out result.json
   ```

---

### JSON Parsing Errors

**Symptom:** `jq` commands fail with "syntax error" or "cannot parse".

**Diagnosis:**
```bash
# Validate JSON output
namelens check myname --output-format=json | jq empty

# Check for incomplete or malformed JSON
namelens check myname --output-format=json | jq 'length'
```

**Common Issues:**

1. **Truncated output** — Check ended early or JSON was cut off
2. **Extra commas** — Malformed JSON from AI backend
3. **Unicode issues** — Special characters not properly escaped

**Solutions:**

1. **Validate with jq:**
   ```bash
   # Check JSON validity
   namelens check myname --output-format=json | jq empty
   ```

2. **Use raw output for debugging:**
   ```bash
   # Capture raw response
   namelens check myname --output-format=json > raw.json
   cat raw.json | jq '.' 2>&1 | head -50
   ```

---

## Expert Backend Issues

### AILink 503 Errors

**Symptom:** Expert analysis fails with error:

```json
{
  "ailink_error": {
    "code": "AILINK_API_ERROR",
    "message": "expert request failed",
    "details": "xai request failed: status 503: {\"code\":\"The service is currently unavailable\",\"error\":\"Service temporarily unavailable. The model is at capacity and currently cannot serve this request. Please try again later.\"}"
  }
}
```

**Root Cause:** The xAI/Grok service is experiencing high load and is temporarily rejecting new requests.

**Solutions:**

1. **Wait and retry:** The service is temporarily overloaded; try again in 5-10 minutes
2. **Disable expert mode:** For domain-only checks, you can skip expert analysis:
   ```bash
   namelens check myname --no-expert
   # Or set environment variable
   NAMELENS_EXPERT_ENABLED=false namelens check myname
   ```
3. **Check service status:** Monitor https://status.x.ai/ for service availability

**Status:** This is a provider-side issue, not a Namelens bug. Monitor the service status page for resolution.

---

### Missing API Keys

**Symptom:** Expert analysis fails with error:

```json
{
  "ailink_error": {
    "code": "AILINK_NO_API_KEY",
    "message": "provider api key not configured"
  }
}
```

**Root Cause:** AI provider API key is not configured in NameLens configuration.

**Solutions:**

1. **Check configuration:**
   ```bash
   # Check if expert is enabled
   namelens config show 2>&1 | grep -A 5 "expert"
   ```

2. **Add API key:**
   ```bash
   # Set via environment variable (for xAI/Grok)
   NAMELENS_AILINK_PROVIDERS_NAMELENS_XAI_CREDENTIALS_0_API_KEY=your-key-here
   
   # Or configure in config file
   namelens config edit
   # Follow prompts to add provider credentials
   ```

3. **Verify key is set:**
   ```bash
   # Check if key is loaded
   namelens doctor 2>&1 | grep -i "api.*key"
   ```

---

## Configuration Issues

### Profiles Not Loading

**Symptom:** Using `--profile startup` or other profiles returns error or uses default TLDs instead.

**Diagnosis:**
```bash
# List available profiles
namelens profile list

# Check profile configuration
namelens profile show startup
```

**Root Causes:**

1. **Custom config file** — Using a config file in non-standard location
2. **Profile file corrupted** — Custom profile has syntax errors
3. **Profile not found** — Referenced profile doesn't exist

**Solutions:**

1. **Check config location:**
   ```bash
   # Find config file
   namelens config show | grep "config file"
   
   # Check expected location
   ls -la ~/.config/namelens/config.yaml
   ```

2. **Reset to defaults:**
   ```bash
   # Remove custom config to use defaults
   rm ~/.config/namelens/config.yaml
   ```

3. **Create or repair profile:**
   ```bash
   # Edit startup profile
   namelens profile edit startup
   
   # Or use built-in defaults without profile
   namelens check myname --tlds com,io,dev
   ```

---

### Environment Variables

**Symptom:** Configuration settings aren't being applied, or NameLens behavior differs from documentation.

**Diagnosis:**
```bash
# Check environment variables
env | grep NAMELENS

# Check if a variable is set
echo $NAMELENS_EXPERT_ENABLED
```

**Common Variables:**

| Variable | Purpose | Default |
| -------- | -------- | -------- |
| `NAMELENS_CONFIG_PATH` | Custom config file location | `$XDG_CONFIG_HOME/namelens/config.yaml` |
| `NAMELENS_DB_PATH` | Custom database location | `$XDG_CONFIG_HOME/namelens/namelens.db` |
| `NAMELENS_EXPERT_ENABLED` | Enable/disable expert mode | `true` |
| `NAMELENS_AILINK_PROMPTS_DIR` | Custom prompts directory | Embedded prompts |
| `NAMELENS_LOG_LEVEL` | Logging verbosity | `info` |

**Solutions:**

1. **Set environment variable:**
   ```bash
   export NAMELENS_LOG_LEVEL=debug
   namelens check myname
   ```

2. **Unset conflicting variables:**
   ```bash
   unset NAMELENS_CONFIG_PATH
   namelens check myname
   ```

3. **Use config file instead:** For complex configuration, use a config file rather than environment variables.

---

## Build and Installation

### Build Failures

**Symptom:** `make build` fails with errors during compilation.

**Diagnosis:**
```bash
# Check Go version
go version

# Check dependencies
go mod tidy

# Try verbose build
make build 2>&1
```

**Common Issues:**

1. **Go version too old** — NameLens requires Go 1.22+
2. **Dependency conflicts** — `go.mod` has conflicting module versions
3. **Missing dependencies** — `go mod download` failed to fetch required modules

**Solutions:**

1. **Update Go:**
   ```bash
   # macOS
   brew upgrade go
   
   # Linux
   sudo apt-get update && sudo apt-get install golang
   ```

2. **Clean dependencies:**
   ```bash
   go mod tidy
   go mod verify
   ```

3. **Rebuild from clean:**
   ```bash
   make clean
   make build
   ```

---

### Installation Issues

**Symptom:** `namelens` command not found after `make install`.

**Diagnosis:**
```bash
# Check if binary was installed
which namelens

# Check installation location
ls -la ~/.local/bin/namelens

# Check PATH
echo $PATH | tr ':' '\n' | grep local
```

**Root Causes:**

1. **PATH not updated** — Shell needs to reload PATH or you need to open new terminal
2. **Installation failed** — `make install` had errors but didn't report them
3. **Binary not created** — Build step failed silently

**Solutions:**

1. **Reload shell:**
   ```bash
   # Bash/Zsh
   source ~/.zshrc
   
   # Fish
   source ~/.config/fish/config.fish
   ```

2. **Manual installation:**
   ```bash
   # Install binary directly
   cp bin/namelens ~/.local/bin/namelens
   
   # Verify it works
   ~/.local/bin/namelens --version
   ```

3. **Check build output:**
   ```bash
   # Look for errors in build
   make build 2>&1 | tee build.log
   
   # Check if binary was created
   ls -lh bin/namelens
   ```

---

## Getting Help

### Command Not Found

**Symptom:** Running `namelens <command>` returns "command not found" error.

**Diagnosis:**
```bash
# List all available commands
namelens --help

# Check command syntax
namelens check myname --help
```

**Common Issues:**

1. **Typo in command name** — Command names are case-sensitive (e.g., `Check` vs `check`)
2. **Flag format** — Using wrong flag format (e.g., `-output json` instead of `--output-format json`)
3. **Missing dependencies** — Some commands require additional setup

**Solutions:**

1. **Use --help:** Every command has help available:
   ```bash
   namelens check --help
   namelens review --help
   namelens generate --help
   ```

2. **Check documentation:**
   ```bash
   # Read relevant docs
   cat README.md
   cat docs/user-guide/README.md
   ```

---

## Advanced Debugging

### Enable Verbose Logging

For detailed debugging information, use the `-v` or `--verbose` flag:

```bash
namelens check myname --verbose
```

This will output:
- HTTP request/response details
- Cache hit/miss information
- RDAP server selection and fallback attempts
- Error stack traces (if any)

---

### Check Database Directly

To inspect the cache database directly:

```bash
# Connect to database
sqlite3 ~/.config/namelens/namelens.db

# List tables
.tables

# Query domain cache
SELECT name, tld, available, expires_at, extra_data FROM check_cache WHERE check_type='domain' LIMIT 10;

# Exit
.quit
```

---

### Verify Configuration

Run diagnostics to check your setup:

```bash
namelens doctor
```

This will display:
- Configuration file location
- Database status
- AI provider configuration
- Rate limit state
- Profile definitions

---

## Still Having Issues?

If you're still experiencing problems after following this guide:

1. **Check for known issues:** See [docs/known-issues.md](known-issues.md) for version-specific problems
2. **Search existing issues:** Check GitHub issues at https://github.com/3leaps/namelens/issues
3. **File a bug report:** Create an issue with:
   - Your NameLens version (`namelens --version`)
   - OS and architecture (`uname -a` on macOS/Linux, `uname -m` on Linux)
   - Steps to reproduce
   - Expected vs actual behavior
   - Full error messages (use `--verbose` flag)
4. **Share output:** Include relevant JSON output or command results
