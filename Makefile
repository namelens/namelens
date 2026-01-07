# NameLens Makefile
# Follows 3leaps/crucible makefile-minimum standard

.PHONY: all help bootstrap check fmt fmt-check lint check-prompts test test-cov build build-all clean run version install
.PHONY: precommit prepush dependencies licenses
.PHONY: release-clean release-download release-checksums release-verify-checksums
.PHONY: release-sign release-export-keys release-verify-keys release-notes
.PHONY: release-upload release-upload-provenance release-upload-all release-build

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
# All env vars use NAMELENS_ prefix for hygiene
# Set NAMELENS_RELEASE_TAG=vX.Y.Z before running release targets
NAMELENS_RELEASE_TAG ?= v$(VERSION)
DIST_RELEASE ?= dist/release
SIGNING_ENV_PREFIX ?= $(shell echo "$(BINARY_NAME)" | tr '[:lower:]-' '[:upper:]_')

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

check: fmt-check lint test ## Run all quality checks

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
	@$(GOTEST) ./... -v
	@echo "Tests complete"

build: ## Build binary
	@echo "Building $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p bin
	@CGO_ENABLED=1 $(GOCMD) build -ldflags="$(LDFLAGS)" -o bin/$(BINARY_NAME)$(EXT) ./cmd/$(BINARY_NAME)
	@echo "Built: bin/$(BINARY_NAME)$(EXT)"

build-all: ## Build for current OS (CGO required, no cross-compile)
	@echo "Building for current OS (CGO required)..."
	@echo "Note: Cross-compilation not supported due to CGO dependency (go-libsql)"
	@mkdir -p bin
	@CGO_ENABLED=1 $(GOCMD) build -ldflags="$(LDFLAGS) -s -w" -trimpath -o bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
	@echo "Build complete: bin/$(BINARY_NAME)"

run: ## Run server in development mode
	@$(GOCMD) run ./cmd/$(BINARY_NAME) serve

version: ## Print current version
	@echo "$(VERSION)"

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
	@$(GOTEST) ./... -coverprofile=coverage.out
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

prepush: ## Pre-push checks (format, lint, security - fail on HIGH)
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
# Env vars (use NAMELENS_ prefix):
#   NAMELENS_RELEASE_TAG   - release tag (e.g., v0.1.0)
#   NAMELENS_MINISIGN_KEY  - path to minisign secret key
#   NAMELENS_MINISIGN_PUB  - path to minisign public key (optional, derived from key)
#   NAMELENS_PGP_KEY_ID    - GPG key ID for PGP signing (optional)
#   NAMELENS_GPG_HOME      - GPG homedir containing signing key (required if PGP_KEY_ID set)
# ─────────────────────────────────────────────────────────────────────────────

release-clean: ## Clean dist/release staging
	@echo "🧹 Cleaning $(DIST_RELEASE)..."
	@rm -rf "$(DIST_RELEASE)"
	@mkdir -p "$(DIST_RELEASE)"
	@echo "✅ Cleaned"

release-download: ## Download GitHub release assets (NAMELENS_RELEASE_TAG=vX.Y.Z)
	@./scripts/release-download.sh "$(NAMELENS_RELEASE_TAG)" "$(DIST_RELEASE)"

release-checksums: ## Generate SHA256SUMS and SHA512SUMS in dist/release
	@echo "→ Generating checksum manifests in $(DIST_RELEASE)..."
	@./scripts/generate-checksums.sh "$(DIST_RELEASE)" "$(BINARY_NAME)"

release-verify-checksums: ## Verify SHA256SUMS and SHA512SUMS against artifacts
	@./scripts/verify-checksums.sh "$(DIST_RELEASE)"

release-sign: ## Sign checksum manifests (minisign required; PGP optional)
	@SIGNING_ENV_PREFIX="$(SIGNING_ENV_PREFIX)" SIGNING_APP_NAME="$(BINARY_NAME)" \
		./scripts/sign-release-manifests.sh "$(NAMELENS_RELEASE_TAG)" "$(DIST_RELEASE)"

release-export-keys: ## Export public signing keys into dist/release
	@SIGNING_ENV_PREFIX="$(SIGNING_ENV_PREFIX)" SIGNING_APP_NAME="$(BINARY_NAME)" \
		./scripts/export-release-keys.sh "$(DIST_RELEASE)"

release-verify-keys: ## Verify exported public keys are public-only
	@if [ -f "$(DIST_RELEASE)/$(BINARY_NAME)-minisign.pub" ]; then \
		./scripts/verify-minisign-public-key.sh "$(DIST_RELEASE)/$(BINARY_NAME)-minisign.pub"; \
	else \
		echo "ℹ️  No minisign public key found (skipping)"; \
	fi
	@if [ -f "$(DIST_RELEASE)/3leaps-release-signing-key.asc" ]; then \
		./scripts/verify-public-key.sh "$(DIST_RELEASE)/3leaps-release-signing-key.asc"; \
	else \
		echo "ℹ️  No PGP public key found (skipping)"; \
	fi

release-notes: ## Copy docs/releases/vX.Y.Z.md into dist/release
	@notes_src="docs/releases/$(NAMELENS_RELEASE_TAG).md"; \
	notes_dst="$(DIST_RELEASE)/release-notes-$(NAMELENS_RELEASE_TAG).md"; \
	if [ ! -f "$$notes_src" ]; then \
		echo "ℹ️  No release notes found at $$notes_src (skipping)"; \
	else \
		cp "$$notes_src" "$$notes_dst"; \
		echo "✅ Copied $$notes_src → $$notes_dst"; \
	fi

release-upload: release-upload-provenance ## Upload provenance assets to GitHub (NAMELENS_RELEASE_TAG=vX.Y.Z)
	@:

release-upload-provenance: release-verify-checksums release-verify-keys ## Upload manifests, signatures, keys, notes
	@./scripts/release-upload-provenance.sh "$(NAMELENS_RELEASE_TAG)" "$(DIST_RELEASE)"

release-upload-all: release-verify-checksums release-verify-keys ## Upload binaries + provenance (manual-only)
	@./scripts/release-upload.sh "$(NAMELENS_RELEASE_TAG)" "$(DIST_RELEASE)"

release-build: release-clean ## Build release artifacts locally (for manual release)
	@echo "→ Building release artifacts for $(BINARY_NAME) v$(VERSION)..."
	@mkdir -p "$(DIST_RELEASE)"
	@# Note: CGO required - only current platform supported
	@CGO_ENABLED=1 $(GOCMD) build -ldflags="$(LDFLAGS) -s -w" -trimpath \
		-o "$(DIST_RELEASE)/$(BINARY_NAME)-$(shell go env GOOS)-$(shell go env GOARCH)" \
		./cmd/$(BINARY_NAME)
	@$(MAKE) release-checksums
	@echo "✅ Release build complete (current platform only)"
	@echo "   For multi-platform builds, use CI (push a tag)"
