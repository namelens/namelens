# Release Checklist

Standard checklist for NameLens releases to ensure consistency and quality.

## Pre-Release Phase

### Version Planning

- [ ] Feature briefs completed in `.plans/active/<version>/`
- [ ] All planned features implemented and tested
- [ ] Breaking changes documented
- [ ] Version number decided (semantic versioning: MAJOR.MINOR.PATCH)

### Code Quality

- [ ] All tests passing: `make test`
- [ ] Code formatted: `make fmt`
- [ ] Lint checks clean: `make lint`
- [ ] Application builds: `make build`
- [ ] Manual smoke tests completed:
  - [ ] `./bin/namelens version`
  - [ ] `./bin/namelens check example`
  - [ ] `./bin/namelens serve` (starts without errors)

### Documentation

- [ ] `README.md` reviewed and updated
- [ ] Feature documentation added to `docs/` (if applicable)

### Dependencies

- [ ] `go.mod` dependencies reviewed
- [ ] `go mod tidy` executed
- [ ] No security vulnerabilities in dependencies

## Release Preparation

### Version Updates

- [ ] Update VERSION file: `echo "X.Y.Z" > VERSION`
- [ ] Search for hardcoded version references

### Git Hygiene

- [ ] All changes committed
- [ ] Commit messages follow attribution standard
- [ ] No uncommitted changes: `git status` clean

### Final Validation

- [ ] Fresh clone test: Clone repo fresh, run `make build && make test`
- [ ] CGO builds successfully on current platform

## Release Execution

### Tagging

- [ ] Create annotated git tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
- [ ] Push commits: `git push origin main`
- [ ] Push tag: `git push origin vX.Y.Z`

### CI Build (Automatic)

When you push a tag, CI automatically:

1. Validates VERSION matches tag
2. Builds CGO-enabled binaries on native runners:
   - `ubuntu-latest` for linux-amd64
   - `ubuntu-24.04-arm` for linux-arm64
   - `macos-13` for darwin-amd64
   - `macos-latest` for darwin-arm64
3. Creates macOS universal binary via `lipo`
4. Uploads binaries to GitHub Release

### Release Artifacts & Signing (Manual)

Follow the Fulmen "manifest-only" provenance pattern:

- Download CI-built artifacts
- Generate SHA256 + SHA512 manifests
- Sign manifests with minisign (primary) and optionally PGP
- Upload provenance assets (never re-upload binaries)

#### Environment Variables

All env vars use the `NAMELENS_` prefix for hygiene:

```bash
# Required for all signing operations
export NAMELENS_RELEASE_TAG=v<version>                    # e.g., v0.1.0 (include 'v' prefix)

# Minisign (required)
export NAMELENS_MINISIGN_KEY=/path/to/namelens.key       # secret key for signing
export NAMELENS_MINISIGN_PUB=/path/to/namelens.pub       # public key for export

# PGP (optional)
export NAMELENS_PGP_KEY_ID="security@3leaps.net"          # may be email, fingerprint, or subkey
export NAMELENS_GPG_HOME=/path/to/gnupg-3leaps            # required if PGP_KEY_ID is set
```

#### Signing Workflow

- [ ] Download CI-built artifacts:

  ```bash
  make release-clean
  make release-download
  ```

- [ ] Generate checksum manifests:

  ```bash
  make release-checksums
  make release-verify-checksums
  ```

- [ ] Sign manifests (minisign required; PGP optional):

  ```bash
  make release-sign
  ```

- [ ] Export public keys into `dist/release/`: `make release-export-keys`
- [ ] Verify exported keys are public-only: `make release-verify-keys`
- [ ] Copy release notes into `dist/release/`: `make release-notes`
- [ ] Upload provenance assets (manifests + signatures + public keys + notes):
      `make release-upload`
  - If you are doing a fully manual release build (no CI artifacts), use:
    `make release-upload-all`

## Supported Platforms

NameLens uses `go-libsql` which requires CGO and only supports:

| OS    | Architecture | Binary Name                 | Notes                   |
| ----- | ------------ | --------------------------- | ----------------------- |
| Linux | amd64        | `namelens-linux-amd64`      | Requires glibc          |
| Linux | arm64        | `namelens-linux-arm64`      | Requires glibc          |
| macOS | amd64        | `namelens-darwin-amd64`     | Intel Macs              |
| macOS | arm64        | `namelens-darwin-arm64`     | Apple Silicon           |
| macOS | universal    | `namelens-darwin-universal` | Fat binary (Intel + AS) |

**Not supported**: Windows, musl/Alpine Linux.

See `docs/operations/builds.md` for build/runtime requirements and recommended
base images.

## Post-Release

### Verification

- [ ] GitHub Release page shows all expected artifacts
- [ ] Download and verify at least one binary works
- [ ] Verify checksums: `sha256sum -c SHA256SUMS --ignore-missing`
- [ ] Verify signature: `minisign -Vm SHA256SUMS -p namelens-minisign.pub`

### Communication

- [ ] Announce release (if applicable)
- [ ] Update any downstream references

### Housekeeping

- [ ] Archive old planning docs (move to `.plans/archive/` if needed)
- [ ] Plan next version features

## Troubleshooting

### CGO Build Failures

If builds fail with CGO errors:

1. Ensure `CGO_ENABLED=1` is set
2. Verify C compiler is available (gcc/clang)
3. Check go-libsql version compatibility

### Version Mismatch

If release fails with version mismatch:

1. Update VERSION file: `echo "X.Y.Z" > VERSION`
2. Commit and push
3. Delete the tag: `git tag -d vX.Y.Z && git push origin :refs/tags/vX.Y.Z`
4. Re-tag and push

### Missing Platform Artifacts

If a platform build fails:

1. Check the specific job logs in GitHub Actions
2. The workflow uses `fail-fast: false` so other platforms continue
3. Fix the issue and re-run the failed job, or delete tag and re-release

## Signing Key Setup

### Minisign (Required)

```bash
# Generate key pair (one-time)
minisign -G -p namelens.pub -s namelens.key

# Store securely and set env vars in your shell profile
export NAMELENS_MINISIGN_KEY=/secure/path/namelens.key
export NAMELENS_MINISIGN_PUB=/secure/path/namelens.pub
```

### PGP (Optional)

```bash
# Use isolated GPG homedir for signing keys
export NAMELENS_GPG_HOME=/secure/path/gnupg-3leaps
export NAMELENS_PGP_KEY_ID="security@3leaps.net"
```

### Quick Reference

All env vars for release signing:

| Variable                | Required | Description                                               |
| ----------------------- | -------- | --------------------------------------------------------- |
| `NAMELENS_RELEASE_TAG`  | Yes      | Release tag (e.g., `v0.1.0`)                              |
| `NAMELENS_MINISIGN_KEY` | Yes      | Path to minisign secret key                               |
| `NAMELENS_MINISIGN_PUB` | No       | Path to minisign public key (derived from key if not set) |
| `NAMELENS_PGP_KEY_ID`   | No       | GPG key ID for PGP signing                                |
| `NAMELENS_GPG_HOME`     | If PGP   | GPG homedir containing signing key                        |
