# NameLens Makefile
# Follows 3leaps/crucible makefile-minimum standard

.PHONY: all help bootstrap check fmt fmt-check lint check-prompts test test-cov test-standalone-binary build build-all clean run version install
.PHONY: precommit prepush dependencies licenses
.PHONY: version-set version-bump version-bump-major version-bump-minor version-bump-patch
.PHONY: release-clean release-download release-checksums release-verify-checksums
.PHONY: release-sign release-export-keys release-verify-keys release-verify-signatures release-notes
.PHONY: release-upload release-upload-provenance release-upload-all release-build
.PHONY: release-guard-tag-version
.PHONY: api-lint api-generate check-api
.PHONY: sync-embedded-config verify-embedded-config

# Binary and version information
BINARY_NAME := namelens
VERSION := $(shell cat VERSION 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE)

# Go commands
GOCMD := go
GOTEST := $(GOCMD) test
GOFMT := gofmt

# Pinned tool versions
SFETCH_VERSION := latest
GONEAT_VERSION := v0.3.24

# Release signing configuration
# Env vars use NAMELENS_NAMELENS_ prefix (ORG_REPO pattern).
# Why doubled? The namelens org will have multiple repos (namelens, namelens-web, etc.)
# and using just NAMELENS_ would conflict. This follows the fulmenhq pattern where
# FULMENHQ_FULMINAR_, FULMENHQ_GONEAT_, etc. are used for disambiguation.
# Set NAMELENS_NAMELENS_RELEASE_TAG=vX.Y.Z before running release targets
NAMELENS_NAMELENS_RELEASE_TAG ?= v$(VERSION)
DIST_RELEASE ?= dist/release
SIGNING_ENV_PREFIX ?= NAMELENS_NAMELENS

# Install configuration
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
EXT :=
ifeq ($(GOOS),windows)
EXT := .exe
INSTALL_PREFIX ?= $(USERPROFILE)
INSTALL_BINDIR ?= $(INSTALL_PREFIX)/bin
else
INSTALL_PREFIX ?= $(HOME)
INSTALL_BINDIR ?= $(INSTALL_PREFIX)/.local/bin
endif
INSTALL_TARGET ?= $(INSTALL_BINDIR)/$(BINARY_NAME)$(EXT)

# Default target
all: check build

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-25s\033[0m %s\n", $$1, $$2}'

bootstrap: ## Install dependencies and tools
	@echo "Bootstrapping namelens development environment..."
	@echo ""
	@# Step 0: Verify curl is available (required trust anchor)
	@if ! command -v curl >/dev/null 2>&1; then \
		echo "[!!] curl not found (required for bootstrap)"; \
		echo ""; \
		echo "Install curl for your platform:"; \
		echo "  macOS:  brew install curl"; \
		echo "  Ubuntu: sudo apt install curl"; \
		echo "  Fedora: sudo dnf install curl"; \
		exit 1; \
	fi
	@echo "[ok] curl found"
	@echo ""
	@# Step 1: Install sfetch (trust anchor)
	@if ! command -v sfetch >/dev/null 2>&1; then \
		echo "[..] Installing sfetch (trust anchor)..."; \
		curl -fsSL https://github.com/3leaps/sfetch/releases/download/$(SFETCH_VERSION)/install-sfetch.sh | bash -s -- --install; \
	else \
		echo "[ok] sfetch already installed"; \
	fi
	@# Verify sfetch
	@if ! command -v sfetch >/dev/null 2>&1; then echo "[!!] sfetch installation failed"; exit 1; fi; \
	echo "[ok] sfetch: $$(command -v sfetch)"
	@echo ""
	@# Step 2: Install goneat via sfetch
	@if ! command -v goneat >/dev/null 2>&1; then \
		echo "[..] Installing goneat $(GONEAT_VERSION) via sfetch..."; \
		sfetch --repo fulmenhq/goneat --tag $(GONEAT_VERSION) --install; \
	else \
		echo "[ok] goneat already installed"; \
	fi
	@# Verify goneat
	@if ! command -v goneat >/dev/null 2>&1; then echo "[!!] goneat installation failed"; exit 1; fi; \
	echo "[ok] goneat: $$(goneat version 2>&1 | head -n1)"
	@echo ""
	@# Step 3: Install foundation and lint tools via goneat
	@echo "[..] Installing foundation tools via goneat..."
	@goneat doctor tools --scope foundation --install --yes 2>/dev/null || \
		echo "[!!] goneat doctor tools (foundation) failed, some tools may need manual installation"
	@echo "[..] Installing lint tools via goneat..."
	@goneat doctor tools --scope lint --install --yes 2>/dev/null || \
		echo "[!!] goneat doctor tools (lint) failed, some tools may need manual installation"
	@echo ""
	@echo "Installing Go dependencies..."
	@$(GOCMD) mod download
	@$(GOCMD) mod tidy
	@echo ""
	@echo "[ok] Bootstrap complete"

check: fmt-check lint check-api test ## Run all quality checks

fmt: ## Format code
	@echo "Formatting code with goneat..."
	@if ! command -v goneat >/dev/null 2>&1; then \
		echo "[!!] goneat not found (run 'make bootstrap')"; \
		exit 1; \
	fi
	@goneat assess --categories format --fix --fail-on high --format concise
	@echo "Formatting complete"

fmt-check: ## Check code formatting
	@echo "Checking formatting with goneat..."
	@if ! command -v goneat >/dev/null 2>&1; then \
		echo "[!!] goneat not found (run 'make bootstrap')"; \
		exit 1; \
	fi
	@goneat assess --categories format --check --fail-on high --format concise
	@echo "Formatting check complete"

lint: ## Run linters (goneat: golangci-lint, yamllint, actionlint, checkmake)
	@echo "Running goneat lint..."
	@if ! command -v goneat >/dev/null 2>&1; then \
		echo "[!!] goneat not found (run 'make bootstrap')"; \
		exit 1; \
	fi
	@goneat assess --categories lint --check --fail-on high --format concise
	@echo "Lint complete"

check-prompts: ## Validate AILink prompt formatting and structure
	@echo "Checking prompt formatting..."
	@goneat format --check internal/ailink/prompt/prompts/ || \
		(echo "[!!] Prompt formatting diverged - run 'goneat format internal/ailink/prompt/prompts/'" && exit 1)
	@echo "Checking for fenced JSON blocks..."
	@for f in internal/ailink/prompt/prompts/*.md; do \
		grep -q '```json' "$$f" || (echo "[!!] Missing fenced JSON block: $$f" && exit 1); \
	done
	@echo "Prompt checks complete"

test: ## Run tests
	@echo "Running tests..."
	@$(GOTEST) -tags sysprims_shared ./... -v
	@echo "Tests complete"

test-standalone-binary: ## Run standalone binary integration test
	@echo "Running standalone binary integration test..."
	@$(GOTEST) -tags sysprims_shared ./test/integration -run TestStandaloneBinaryVersionAndCommandsWorkOutsideRepo -v
	@echo "Standalone binary integration test complete"

build: ## Build binary
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p bin
	@CGO_ENABLED=1 $(GOCMD) build -tags sysprims_shared -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)$(EXT) ./cmd/$(BINARY_NAME)
	@echo "Built: bin/$(BINARY_NAME)$(EXT)"

build-all: ## Build for current OS (CGO required, no cross-compile)
	@echo "Building for current OS (CGO required)..."
	@echo "Note: Cross-compilation not supported due to CGO dependency (go-libsql)"
	@mkdir -p bin
	@CGO_ENABLED=1 $(GOCMD) build -tags sysprims_shared -ldflags="$(LDFLAGS) -s -w" -trimpath -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Build complete: bin/$(BINARY_NAME)"

run: ## Run server in development mode
	@$(GOCMD) run -tags sysprims_shared ./cmd/$(BINARY_NAME) serve

version: ## Print current version
	@echo "$(VERSION)"

version-bump: ## Bump version (usage: make version-bump TYPE=patch|minor|major)
	@if [ -z "$(TYPE)" ]; then \
		echo "❌ TYPE not specified. Usage: make version-bump TYPE=patch|minor|major"; \
		exit 1; \
	fi
	@echo "Bumping version ($(TYPE))..."
	@if ! command -v goneat >/dev/null 2>&1; then \
		echo "[!!] goneat not found (run 'make bootstrap')"; \
		exit 1; \
	fi
	@goneat version bump $(TYPE)
	@echo "✅ Version bumped to $$(cat VERSION)"

version-set: ## Set version (usage: make version-set V=x.y.z)
	@if [ -z "$(V)" ]; then \
		echo "❌ V not specified. Usage: make version-set V=x.y.z"; \
		exit 1; \
	fi
	@echo "$(V)" > VERSION
	@echo "✅ Version set to $(V)"

version-bump-major: ## Bump major version (x.0.0)
	@$(MAKE) version-bump TYPE=major

version-bump-minor: ## Bump minor version (0.x.0)
	@$(MAKE) version-bump TYPE=minor

version-bump-patch: ## Bump patch version (0.0.x)
	@$(MAKE) version-bump TYPE=patch

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -rf bin/ dist/ coverage.out coverage.html
	@echo "Clean complete"

install: build ## Install binary to ~/.local/bin (or %USERPROFILE%/bin on Windows)
	@mkdir -p "$(INSTALL_BINDIR)"
	@cp "bin/$(BINARY_NAME)$(EXT)" "$(INSTALL_TARGET)"
ifeq ($(GOOS),windows)
	@echo "Installed $(BINARY_NAME)$(EXT) to $(INSTALL_TARGET)"
	@echo "Ensure $(INSTALL_BINDIR) is on your PATH"
else
	@chmod 755 "$(INSTALL_TARGET)"
	@echo "Installed $(BINARY_NAME) to $(INSTALL_TARGET)"
	@if ! echo "$$PATH" | tr ':' '\n' | grep -q "^$(INSTALL_BINDIR)$$"; then \
		echo "Note: Add $(INSTALL_BINDIR) to your PATH if not already present"; \
	fi
endif

# Development helpers
test-cov: ## Run tests with coverage
	@$(GOTEST) -tags sysprims_shared ./... -coverprofile=coverage.out
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# ─────────────────────────────────────────────────────────────────────────────
# Git hooks (precommit/prepush)
# ─────────────────────────────────────────────────────────────────────────────

precommit: ## Pre-commit checks (format, lint, security - fail on CRITICAL)
	@echo "Running pre-commit checks..."
	@if ! command -v goneat >/dev/null 2>&1; then \
		echo "[!!] goneat not found (run 'make bootstrap')"; \
		exit 1; \
	fi
	@goneat format; goneat assess --check --categories format,lint,security --fail-on critical --format concise
	@echo "Pre-commit checks passed"

sync-embedded-config: ## Sync embedded config/schema assets from source of truth
	@mkdir -p internal/config/embedded/namelens/v0
	@mkdir -p internal/config/embedded/schemas/namelens/v0
	@cp config/namelens/v0/namelens-defaults.yaml internal/config/embedded/namelens/v0/namelens-defaults.yaml
	@cp schemas/namelens/v0/config.schema.json internal/config/embedded/schemas/namelens/v0/config.schema.json
	@mkdir -p internal/ailink/prompt/embedded/schemas/ailink/v0
	@cp schemas/ailink/v0/prompt.schema.json internal/ailink/prompt/embedded/schemas/ailink/v0/prompt.schema.json
	@mkdir -p internal/ailink/embedded/schemas/ailink/v0
	@for f in schemas/ailink/v0/*.json; do cp "$$f" internal/ailink/embedded/schemas/ailink/v0/; done
	@echo "✅ Embedded config assets synced"

verify-embedded-config: ## Verify embedded config/schema assets are in sync
	@cmp -s config/namelens/v0/namelens-defaults.yaml internal/config/embedded/namelens/v0/namelens-defaults.yaml || \
		(echo "❌ Embedded defaults drifted; run 'make sync-embedded-config'" && exit 1)
	@cmp -s schemas/namelens/v0/config.schema.json internal/config/embedded/schemas/namelens/v0/config.schema.json || \
		(echo "❌ Embedded schema drifted; run 'make sync-embedded-config'" && exit 1)
	@cmp -s schemas/ailink/v0/prompt.schema.json internal/ailink/prompt/embedded/schemas/ailink/v0/prompt.schema.json || \
		(echo "❌ Embedded prompt schema drifted; run 'make sync-embedded-config'" && exit 1)
	@for f in schemas/ailink/v0/*.json; do \
		base=$$(basename "$$f"); \
		cmp -s "$$f" "internal/ailink/embedded/schemas/ailink/v0/$$base" || \
			(echo "❌ Embedded ailink schema $$base drifted; run 'make sync-embedded-config'" && exit 1); \
	done
	@echo "✅ Embedded config assets verified"

prepush: verify-embedded-config test-standalone-binary ## Pre-push checks (format, lint, security - fail on HIGH)
	@echo "Running pre-push checks..."
	@if ! command -v goneat >/dev/null 2>&1; then \
		echo "[!!] goneat not found (run 'make bootstrap')"; \
		exit 1; \
	fi
	@goneat format; goneat assess --check --categories format,lint,security --fail-on high --format concise
	@echo "Pre-push checks passed"

# ─────────────────────────────────────────────────────────────────────────────
# Dependency and license management
# ─────────────────────────────────────────────────────────────────────────────

dependencies: ## Generate SBOM and check dependency licenses
	@echo "Generating SBOM and checking dependencies..."
	@if ! command -v goneat >/dev/null 2>&1; then \
		echo "[!!] goneat not found (run 'make bootstrap')"; \
		exit 1; \
	fi
	@mkdir -p sbom
	@goneat dependencies --sbom --sbom-output sbom/$(BINARY_NAME).cdx.json
	@echo "SBOM generated at sbom/$(BINARY_NAME).cdx.json"

licenses: ## Show dependency license summary
	@echo "Dependency licenses:"
	@if ! command -v goneat >/dev/null 2>&1; then \
		echo "[!!] goneat not found (run 'make bootstrap')"; \
		exit 1; \
	fi
	@goneat dependencies --licenses

# ─────────────────────────────────────────────────────────────────────────────
# Release signing workflow (manual, after CI builds artifacts)
#
# Pattern: CI uploads binaries → maintainer downloads, signs, uploads provenance
#
# Env vars (use NAMELENS_NAMELENS_ prefix - org_repo pattern):
#   NAMELENS_NAMELENS_RELEASE_TAG   - release tag (e.g., v0.1.0)
#   NAMELENS_NAMELENS_MINISIGN_KEY  - path to minisign secret key
#   NAMELENS_NAMELENS_MINISIGN_PUB  - path to minisign public key (optional)
#   NAMELENS_NAMELENS_PGP_KEY_ID    - GPG key ID for PGP signing (optional)
#   NAMELENS_NAMELENS_GPG_HOMEDIR   - GPG homedir containing signing key (required if PGP_KEY_ID set)
# ─────────────────────────────────────────────────────────────────────────────

release-guard-tag-version: ## Guard: verify RELEASE_TAG matches VERSION file
	@tag="$(NAMELENS_NAMELENS_RELEASE_TAG)"; \
	ver="v$(VERSION)"; \
	if [ "$$tag" != "$$ver" ]; then \
		echo "❌ Version mismatch: RELEASE_TAG=$$tag but VERSION file says $$ver" >&2; \
		echo "   Either update VERSION file or set correct NAMELENS_NAMELENS_RELEASE_TAG" >&2; \
		exit 1; \
	fi; \
	echo "✅ Version guard passed: $$tag"

release-clean: ## Clean dist/release staging
	@echo "🧹 Cleaning $(DIST_RELEASE)..."
	@rm -rf "$(DIST_RELEASE)"
	@mkdir -p "$(DIST_RELEASE)"
	@echo "✅ Cleaned"

release-download: ## Download GitHub release assets (NAMELENS_NAMELENS_RELEASE_TAG=vX.Y.Z)
	@./scripts/release-download.sh "$(NAMELENS_NAMELENS_RELEASE_TAG)" "$(DIST_RELEASE)"

release-checksums: ## Generate SHA256SUMS and SHA512SUMS in dist/release
	@echo "→ Generating checksum manifests in $(DIST_RELEASE)..."
	@./scripts/generate-checksums.sh "$(DIST_RELEASE)" "$(BINARY_NAME)"

release-verify-checksums: ## Verify SHA256SUMS and SHA512SUMS against artifacts
	@./scripts/verify-checksums.sh "$(DIST_RELEASE)"

release-sign: ## Sign checksum manifests (minisign required; PGP optional)
	@SIGNING_ENV_PREFIX="$(SIGNING_ENV_PREFIX)" SIGNING_APP_NAME="$(BINARY_NAME)" \
		./scripts/sign-release-manifests.sh "$(NAMELENS_NAMELENS_RELEASE_TAG)" "$(DIST_RELEASE)"

release-export-keys: ## Export public signing keys into dist/release
	@SIGNING_ENV_PREFIX="$(SIGNING_ENV_PREFIX)" SIGNING_APP_NAME="$(BINARY_NAME)" \
		./scripts/export-release-keys.sh "$(DIST_RELEASE)"

release-verify-keys: ## Verify exported public keys are public-only
	@if [ -f "$(DIST_RELEASE)/$(BINARY_NAME)-minisign.pub" ]; then \
		./scripts/verify-minisign-public-key.sh "$(DIST_RELEASE)/$(BINARY_NAME)-minisign.pub"; \
	else \
		echo "ℹ️  No minisign public key found (skipping)"; \
	fi
	@if [ -f "$(DIST_RELEASE)/$(BINARY_NAME)-pgp.asc" ]; then \
		./scripts/verify-public-key.sh "$(DIST_RELEASE)/$(BINARY_NAME)-pgp.asc"; \
	else \
		echo "ℹ️  No PGP public key found (skipping)"; \
	fi

release-verify-signatures: ## Verify signatures on checksum manifests
	@echo "🔍 Verifying signatures for $(NAMELENS_NAMELENS_RELEASE_TAG)..."
	@if [ -z "$(NAMELENS_NAMELENS_RELEASE_TAG)" ]; then \
		echo "❌ NAMELENS_NAMELENS_RELEASE_TAG not set. Use: make release-verify-signatures NAMELENS_NAMELENS_RELEASE_TAG=vX.Y.Z"; \
		exit 1; \
	fi
	@if [ ! -d "$(DIST_RELEASE)" ]; then \
		echo "❌ $(DIST_RELEASE) directory not found."; \
		exit 1; \
	fi
	@cd "$(DIST_RELEASE)" && \
		GPG_HOMEDIR_EFF="$${$(SIGNING_ENV_PREFIX)_GPG_HOMEDIR}"; \
		if [ -z "$$GPG_HOMEDIR_EFF" ]; then GPG_HOMEDIR_EFF="$$GPG_HOMEDIR"; fi; \
		echo "🔐 Verifying GPG signatures..."; \
		for asc in SHA256SUMS.asc SHA512SUMS.asc; do \
			if [ -f "$$asc" ]; then \
				if [ -n "$$GPG_HOMEDIR_EFF" ]; then \
					gpg --homedir "$$GPG_HOMEDIR_EFF" --verify "$$asc" "$${asc%.asc}" && \
					echo "  ✅ $$asc - Good signature"; \
				else \
					echo "  ⚠️  $$asc - GPG_HOMEDIR not set; skipping verification"; \
				fi; \
			else \
				echo "  ⚠️  $$asc - Signature file not found"; \
			fi; \
		done; \
		echo "🔏 Verifying minisign signatures..."; \
		for sig in SHA256SUMS.minisig SHA512SUMS.minisig; do \
			if [ -f "$$sig" ] && [ -f "$(BINARY_NAME)-minisign.pub" ]; then \
				minisign -Vm "$${sig%.minisig}" -p $(BINARY_NAME)-minisign.pub && \
				echo "  ✅ $$sig - Good signature"; \
			else \
				echo "  ⚠️  $$sig - Signature or public key file not found"; \
			fi; \
		done
	@echo "✅ Signature verification completed for $(NAMELENS_NAMELENS_RELEASE_TAG)"

release-notes: ## Copy docs/releases/vX.Y.Z.md into dist/release
	@notes_src="docs/releases/$(NAMELENS_NAMELENS_RELEASE_TAG).md"; \
	notes_dst="$(DIST_RELEASE)/release-notes-$(NAMELENS_NAMELENS_RELEASE_TAG).md"; \
	if [ ! -f "$$notes_src" ]; then \
		echo "ℹ️  No release notes found at $$notes_src (skipping)"; \
	else \
		cp "$$notes_src" "$$notes_dst"; \
		echo "✅ Copied $$notes_src → $$notes_dst"; \
	fi

release-upload: release-upload-provenance ## Upload provenance assets to GitHub (NAMELENS_NAMELENS_RELEASE_TAG=vX.Y.Z)
	@:

release-upload-provenance: release-verify-checksums release-verify-keys ## Upload manifests, signatures, keys, notes
	@./scripts/release-upload-provenance.sh "$(NAMELENS_NAMELENS_RELEASE_TAG)" "$(DIST_RELEASE)"

release-upload-all: release-verify-checksums release-verify-keys ## Upload binaries + provenance (manual-only)
	@./scripts/release-upload.sh "$(NAMELENS_NAMELENS_RELEASE_TAG)" "$(DIST_RELEASE)"

release-build: release-clean ## Build release artifacts locally (for manual release)
	@echo "→ Building release artifacts for $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p "$(DIST_RELEASE)"
	@# Note: CGO required - only current platform supported
	@CGO_ENABLED=1 $(GOCMD) build -tags sysprims_shared -ldflags="$(LDFLAGS) -s -w" -trimpath \
		-o "$(DIST_RELEASE)/$(BINARY_NAME)-$(shell go env GOOS)-$(shell go env GOARCH)" \
		./cmd/$(BINARY_NAME)
	@$(MAKE) release-checksums
	@echo "✅ Release build complete (current platform only)"
	@echo "   For multi-platform builds, use CI (push a tag)"

# ─────────────────────────────────────────────────────────────────────────────
# API code generation (OpenAPI spec → Go code)
# ─────────────────────────────────────────────────────────────────────────────

api-lint: ## Lint OpenAPI spec
	@echo "Linting OpenAPI spec..."
	@if ! command -v vacuum >/dev/null 2>&1; then \
		echo "[..] Installing vacuum..."; \
		$(GOCMD) install github.com/daveshanley/vacuum@latest; \
	fi
	@vacuum lint openapi.yaml --no-style
	@echo "OpenAPI lint complete"

api-generate: ## Generate Go code from OpenAPI spec
	@echo "Generating API code from OpenAPI spec..."
	@if ! command -v oapi-codegen >/dev/null 2>&1; then \
		echo "[..] Installing oapi-codegen..."; \
		$(GOCMD) install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest; \
	fi
	@mkdir -p internal/api
	@oapi-codegen -generate types,chi-server -package api \
		-o internal/api/openapi.gen.go openapi.yaml
	@echo "Generated internal/api/openapi.gen.go"

check-api: api-lint api-generate ## Check API spec and generated code are in sync
	@echo "Checking API spec and generated code are in sync..."
	@if ! git diff --exit-code internal/api/openapi.gen.go >/dev/null 2>&1; then \
		echo "❌ API spec and generated code are out of sync"; \
		echo "   Run 'make api-generate' and commit the changes"; \
		exit 1; \
	fi
	@echo "✅ API spec and generated code are in sync"
