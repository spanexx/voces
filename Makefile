# Makefile for Voces

.PHONY: help precommit test build clean install-hooks install install-fast uninstall release release-clean engines vendor/whisper.cpp whispercpp-build whispercpp-install vendor/piper piper-build

# Default target
help:
	@echo "Voces - Makefile Commands"
	@echo ""
	@echo "  precommit              - Run all pre-commit checks"
	@echo "  test                   - Run tests with coverage"
	@echo "  build                  - Build the application"
	@echo "  clean                  - Clean build artifacts"
	@echo "  install-hooks          - Install git pre-commit hooks"
	@echo "  install                - Build and install globally (requires sudo)"
	@echo "  install-fast           - Install from existing bin/ artifacts (requires sudo)"
	@echo "  uninstall              - Remove globally installed application"
	@echo "  release [VERSION=...]  - Build the distribution tarball (Phase 8)"
	@echo "  release-clean          - Remove builds/ directory"
	@echo "  engines                - Build both whisper.cpp and piper engines"
	@echo "  check                  - precommit + build + test (all-in-one)"
	@echo ""

# Run all pre-commit checks
precommit:
	@echo "🔍 Running pre-commit checks..."
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "1/7: Checking protected directories..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@./scripts/check-protected-dirs.sh || exit 1
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "2/7: Checking for mocked test implementations..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@./scripts/check-no-test-mocks.sh || exit 1
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "3/7: Checking for mock implementations in code..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@./scripts/check-no-mocks.sh || exit 1
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "4/7: Checking file sizes..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@./scripts/check-file-size.sh || exit 1
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "5/7: Checking for proper comments..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@./scripts/check-comments.sh || exit 1
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "6/7: Checking test coverage..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@./scripts/check-coverage.sh || exit 1
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "7/7: Checking for secrets..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@./scripts/check-secrets.sh || exit 1
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo "✅ All pre-commit checks passed!"
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Run tests with coverage
test:
	@echo "🧪 Running tests with coverage..."
	@go test -mod=vendor -race -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | grep total
	@rm -f coverage.out

# Build the application
build:
	@echo "🔨 Building application..."
	@go build -mod=vendor -o bin/voces ./cmd/voces
	@go build -mod=vendor -o bin/voces-overlay ./cmd/voces-overlay
	@echo "✅ Build complete: bin/voces"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf bin/
	@rm -f coverage.out
	@echo "✅ Clean complete"

# Install git pre-commit hooks
install-hooks:
	@echo "🔧 Installing git pre-commit hooks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit install; \
		echo "✅ Pre-commit hooks installed"; \
	else \
		echo "⚠️  pre-commit not found. Install with: pip install pre-commit"; \
		echo "   Or use 'make precommit' before committing"; \
	fi

# Run Go formatting
fmt:
	@echo "📝 Formatting code..."
	@go fmt ./...
	@echo "✅ Format complete"

# Run Go vet
vet:
	@echo "🔍 Running go vet..."
	@go vet ./...
	@echo "✅ Vet complete"

# Run linter (requires golangci-lint)
lint:
	@echo "🔍 Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Global installation
PREFIX ?= /usr/local
BINDIR = $(PREFIX)/bin
DATADIR = $(PREFIX)/share
VOCESDIR = $(DATADIR)/voces
WHISPERCPPDIR = $(VOCESDIR)/whisper.cpp
ICONDIR = $(DATADIR)/icons/hicolor/64x64/apps
APPDIR = $(DATADIR)/applications

install: build
	@echo "🚚 Installing globally..."
	@sudo -v
	@$(MAKE) install-fast

install-fast:
	@echo "🚚 Installing from existing build artifacts..."
	@sudo -v
	@sudo mkdir -p $(BINDIR)
	@sudo install -m 755 bin/voces $(BINDIR)/
	@sudo install -m 755 bin/voces-overlay $(BINDIR)/
	@# Optional: install managed whisper.cpp artifacts if built
	@if [ -f vendor/whisper.cpp/build/bin/whisper-cli ]; then \
		echo "📦 Installing managed whisper.cpp..."; \
		sudo mkdir -p $(WHISPERCPPDIR)/bin $(WHISPERCPPDIR)/lib $(WHISPERCPPDIR)/models; \
		sudo install -m 755 vendor/whisper.cpp/build/bin/whisper-cli $(WHISPERCPPDIR)/bin/; \
		if [ -d vendor/whisper.cpp/build/lib ]; then sudo cp -a vendor/whisper.cpp/build/lib/. $(WHISPERCPPDIR)/lib/; fi; \
		if [ -d vendor/whisper.cpp/models ]; then sudo cp -a vendor/whisper.cpp/models/. $(WHISPERCPPDIR)/models/; fi; \
	fi
	@sudo mkdir -p $(ICONDIR)
	@sudo install -m 644 assets/icons/idle.png $(ICONDIR)/voces.png
	@sudo mkdir -p $(APPDIR)
	@sudo install -m 644 voces.desktop $(APPDIR)/
	@if command -v gtk-update-icon-cache >/dev/null 2>&1; then \
		sudo gtk-update-icon-cache -f -t $(DATADIR)/icons/hicolor || true; \
	fi
	@if command -v update-desktop-database >/dev/null 2>&1; then \
		sudo update-desktop-database $(APPDIR) || true; \
	fi
	@echo "✅ Installation complete. You can now run 'voces' from anywhere."

uninstall:
	@echo "🗑️  Uninstalling globally..."
	@sudo -v
	@sudo rm -f $(BINDIR)/voces
	@sudo rm -f $(BINDIR)/voces-overlay
	@sudo rm -rf $(WHISPERCPPDIR)
	@sudo rm -f $(ICONDIR)/voces.png
	@sudo rm -f $(APPDIR)/voces.desktop
	@if command -v gtk-update-icon-cache >/dev/null 2>&1; then \
		sudo gtk-update-icon-cache -f -t $(DATADIR)/icons/hicolor || true; \
	fi
	@echo "✅ Uninstallation complete."

vendor/whisper.cpp:
	@git clone --depth 1 https://github.com/ggerganov/whisper.cpp vendor/whisper.cpp

whispercpp-build: vendor/whisper.cpp
	@echo "🔨 Building whisper.cpp (managed)..."
	@mkdir -p vendor/whisper.cpp/build
	@cmake -S vendor/whisper.cpp -B vendor/whisper.cpp/build -DWHISPER_BUILD_EXAMPLES=ON -DBUILD_SHARED_LIBS=ON
	@cmake --build vendor/whisper.cpp/build -j

whispercpp-install: whispercpp-build
	@echo "📦 Installing whisper.cpp artifacts into $(WHISPERCPPDIR)..."
	@sudo mkdir -p $(WHISPERCPPDIR)/bin $(WHISPERCPPDIR)/lib $(WHISPERCPPDIR)/models
	@sudo install -m 755 vendor/whisper.cpp/build/bin/whisper-cli $(WHISPERCPPDIR)/bin/
	@if [ -d vendor/whisper.cpp/build/lib ]; then sudo cp -a vendor/whisper.cpp/build/lib/. $(WHISPERCPPDIR)/lib/; fi
	@if [ -d vendor/whisper.cpp/models ]; then sudo cp -a vendor/whisper.cpp/models/. $(WHISPERCPPDIR)/models/; fi

# Run all checks (precommit + build + test)
check: precommit build test
	@echo ""
	@echo "✅ All checks passed!"

# =============================================================================
# Phase 8: release pipeline (IMPL §8 / ADR-0001 / ADR-0002)
# =============================================================================

# release [VERSION=vX.Y.Z] — produce the distributable tarball.
# Defaults to a `dev` version if VERSION is unset. Builds both Go binaries
# (with -ldflags for Version injection), compiles both engines, and bundles
# them together with docs + install-deps.sh + config.yaml.example.
release:
	@VERSION="${VERSION:-v0.0.0-dev}"; \
	if [ "$$VERSION" = "v0.0.0-dev" ]; then \
		echo "ℹ️  VERSION not set, defaulting to $$VERSION"; \
		echo "   Override with: make release VERSION=v0.2.0"; \
	fi; \
	bash scripts/build-release.sh "$$VERSION"

# release-clean — wipe the builds/ directory (the release artifacts).
release-clean:
	@echo "🧹 Cleaning builds/ directory..."
	@rm -rf builds/
	@echo "✅ Release artifacts removed."

# engines — convenience: build both engine binaries without the full tarball.
# Useful for local smoke testing or `make install` after manual edits.
engines: whispercpp-build piper-build

vendor/piper:
	@echo "📥 Cloning piper (rhasspy/piper) into vendor/piper..."
	@git clone --depth 1 https://github.com/rhasspy/piper vendor/piper
	@echo "✅ piper vendored."
	@echo ""
	@echo "⚠️  NOTE: piper has heavy build deps (ONNX runtime, espeak-ng, etc.)."
	@echo "   If 'make piper-build' fails, you can either:"
	@echo "   (a) install piper system-wide (apt: piper); or"
	@echo "   (b) download a prebuilt binary from piper's release page."

# piper-build — compile the vendored piper source. Heavy build.
# Currently a documented stub: prints instructions. Full cmake build is
# gated on a working ONNX runtime + espeak-ng + build toolchain.
piper-build: vendor/piper
	@if [ ! -f vendor/piper/CMakeLists.txt ]; then \
		echo "❌ vendor/piper/CMakeLists.txt not found."; \
		echo "   The vendored piper source seems incomplete."; \
		echo "   Try: rm -rf vendor/piper && make vendor/piper"; \
		exit 1; \
	fi
	@echo "🔨 Building piper (managed)..."
	@echo ""
	@echo "   Full piper build requires ONNX runtime + espeak-ng. Skipping the"
	@echo "   heavy build by default. To enable:"
	@echo "     1. apt install libonnxruntime-dev libespeak-ng-dev"
	@echo "     2. cd vendor/piper && cmake -B build -S . && cmake --build build -j"
	@echo "     3. re-run 'make release' — piper will be picked up automatically"
	@echo ""
	@echo "   For now, this target is a no-op so 'make release' still works."
	@echo "   TTS users will need a system-wide piper install until Phase 8.1 lands."
	@exit 0
