#!/usr/bin/env bash
# Cloud Agent install script for backstop-core.
#
# Idempotent bootstrap of the full backstop development experience:
#   - the analyzer toolchain the gate provisions BY NAME on PATH
#     (golangci-lint v2, go-arch-lint, semgrep, ast-grep) at the exact
#     pins CI uses (.github/workflows/ci.yml + pkg/pack/engine/allowlist.go),
#   - the Go module cache,
#   - the backstop CLI built from THIS source (the latest CLI), installed
#     onto PATH as `backstop`,
#   - the pack fleet reconstituted from backstop.lock into .backstop/packs/.
#
# It must terminate and be safe to re-run: every step checks for the pinned
# result and skips when already satisfied. All binaries land in /usr/local/bin
# (always on PATH) so no shell-profile mutation is needed.
set -euo pipefail

# --- Pins (keep in lockstep with .github/workflows/ci.yml) --------------------
GOLANGCI_LINT_VERSION="v2.6.0"
GO_ARCH_LINT_VERSION="v1.16.0"
SEMGREP_VERSION="1.156.0"
AST_GREP_VERSION="0.43.0"

BIN_DIR="/usr/local/bin"
SEMGREP_VENV="/opt/backstop-tools/semgrep-venv"

log() { printf '\n=== %s ===\n' "$1"; }

# Resolve repo root (the directory this script's .cursor/ lives in).
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

# --- 1. golangci-lint v2 (real binary on PATH, v2 syntax required) -----------
log "golangci-lint ${GOLANGCI_LINT_VERSION}"
if command -v golangci-lint >/dev/null 2>&1 && golangci-lint version 2>&1 | grep -q "${GOLANGCI_LINT_VERSION#v}"; then
  echo "already installed: $(golangci-lint version 2>&1 | head -1)"
else
  tmp="$(mktemp -d)"
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
    | sh -s -- -b "$tmp" "${GOLANGCI_LINT_VERSION}"
  sudo install -m 0755 "$tmp/golangci-lint" "$BIN_DIR/golangci-lint"
  rm -rf "$tmp"
  echo "installed: $(golangci-lint version 2>&1 | head -1)"
fi

# --- 2. go-arch-lint (assume-present Layer-0 analyzer) -----------------------
log "go-arch-lint ${GO_ARCH_LINT_VERSION}"
if command -v go-arch-lint >/dev/null 2>&1 && go-arch-lint version 2>&1 | grep -q "${GO_ARCH_LINT_VERSION#v}"; then
  echo "already installed: $(go-arch-lint version 2>&1 | head -1)"
else
  GOFLAGS="" go install "github.com/fe3dback/go-arch-lint@${GO_ARCH_LINT_VERSION}"
  sudo install -m 0755 "$(go env GOPATH)/bin/go-arch-lint" "$BIN_DIR/go-arch-lint"
  echo "installed: $(go-arch-lint version 2>&1 | head -1)"
fi

# --- 3. ast-grep (provisioned engine pin) ------------------------------------
log "ast-grep ${AST_GREP_VERSION}"
if command -v ast-grep >/dev/null 2>&1 && ast-grep --version 2>&1 | grep -q "${AST_GREP_VERSION}"; then
  echo "already installed: $(ast-grep --version 2>&1 | head -1)"
else
  tmp="$(mktemp -d)"
  curl -sSfL "https://github.com/ast-grep/ast-grep/releases/download/${AST_GREP_VERSION}/app-x86_64-unknown-linux-gnu.zip" -o "$tmp/ast-grep.zip"
  unzip -o -q "$tmp/ast-grep.zip" -d "$tmp"
  sudo install -m 0755 "$tmp/ast-grep" "$BIN_DIR/ast-grep"
  # ast-grep ships an `sg` alias; install it too when present.
  [ -f "$tmp/sg" ] && sudo install -m 0755 "$tmp/sg" "$BIN_DIR/sg" || true
  rm -rf "$tmp"
  echo "installed: $(ast-grep --version 2>&1 | head -1)"
fi

# --- 4. semgrep (provisioned engine pin, isolated venv) ----------------------
log "semgrep ${SEMGREP_VERSION}"
if command -v semgrep >/dev/null 2>&1 && semgrep --version 2>&1 | grep -q "${SEMGREP_VERSION}"; then
  echo "already installed: semgrep $(semgrep --version 2>&1 | head -1)"
else
  # `python3 -m venv` needs ensurepip, shipped by the python3-venv apt package
  # (absent on the base image). Probe by actually creating a throwaway venv,
  # because `venv --help` succeeds even when ensurepip is missing.
  probe="$(mktemp -d)"
  if ! python3 -m venv "$probe/v" >/dev/null 2>&1; then
    sudo apt-get update -y
    sudo apt-get install -y python3-venv
  fi
  rm -rf "$probe"
  sudo mkdir -p "$(dirname "$SEMGREP_VENV")"
  sudo chown "$(id -u):$(id -g)" "$(dirname "$SEMGREP_VENV")"
  # Recreate cleanly if a previous run left a broken (ensurepip-less) venv.
  [ -x "$SEMGREP_VENV/bin/pip" ] || { rm -rf "$SEMGREP_VENV"; python3 -m venv "$SEMGREP_VENV"; }
  "$SEMGREP_VENV/bin/pip" install --quiet --upgrade pip
  "$SEMGREP_VENV/bin/pip" install --quiet "semgrep==${SEMGREP_VERSION}"
  sudo ln -sf "$SEMGREP_VENV/bin/semgrep" "$BIN_DIR/semgrep"
  echo "installed: semgrep $(semgrep --version 2>&1 | head -1)"
fi

# --- 5. Go module dependencies -----------------------------------------------
log "go mod download"
go mod download

# --- 6. Build the backstop CLI from THIS source and put it on PATH -----------
log "build backstop CLI (latest, from source)"
go build -o ./bin/backstop ./cmd/backstop
sudo install -m 0755 ./bin/backstop "$BIN_DIR/backstop"
echo "installed: $(backstop --version 2>&1 | head -1 || echo 'backstop (version flag unavailable)')"

# --- 7. Reconstitute the pack fleet from backstop.lock -----------------------
# Packs live outside core in gitignored .backstop/packs/ (like node_modules);
# the lock file is the durable record. Without this the gate reports
# capability_absent for every dimension and passes having checked nothing.
log "backstop pack install"
./bin/backstop pack install

# --- 8. Playwright + Chromium (public-site verification) ---------------------
# tests/public-site.spec.ts drives a headless Chromium via @playwright/test
# (pinned in package-lock.json). Mirrors .github/workflows/site-verification.yml:
# `npm ci` then `playwright install --with-deps chromium`. The browser downloads
# into ~/.cache/ms-playwright (captured by the environment snapshot); the OS libs
# go in via apt (--with-deps auto-uses sudo for the apt step). Idempotent: npm ci
# is deterministic and playwright skips an already-downloaded browser.
log "npm ci (playwright test harness)"
npm ci
log "playwright install --with-deps chromium"
npx --yes playwright install --with-deps chromium

# --- 9. Ruby + Jekyll (public-site build) ------------------------------------
# scripts/verify-public-site.sh runs `bundle exec jekyll build --source docs`.
# CI pins Ruby 3.3.4 (whose default Bundler 2.5.11 matches Gemfile.lock's
# BUNDLED WITH). No Ruby ships on the base image and there is no version
# manager, so build 3.3.4 with ruby-build into a fixed prefix and expose it
# (plus the bundle-provided `jekyll` executable) on PATH via /usr/local/bin.
RUBY_VERSION_PIN="3.3.4"
RUBY_PREFIX="/opt/ruby-${RUBY_VERSION_PIN}"
log "ruby ${RUBY_VERSION_PIN}"
if [ -x "$RUBY_PREFIX/bin/ruby" ] && "$RUBY_PREFIX/bin/ruby" --version 2>/dev/null | grep -q "$RUBY_VERSION_PIN"; then
  echo "already built: $("$RUBY_PREFIX/bin/ruby" --version)"
else
  sudo apt-get update -y
  sudo apt-get install -y autoconf patch build-essential libssl-dev libyaml-dev \
    zlib1g-dev libreadline-dev libffi-dev libgdbm-dev libncurses-dev
  tmp="$(mktemp -d)"
  git clone --depth 1 https://github.com/rbenv/ruby-build.git "$tmp/ruby-build"
  sudo mkdir -p "$RUBY_PREFIX"
  sudo chown "$(id -u):$(id -g)" "$RUBY_PREFIX"
  "$tmp/ruby-build/bin/ruby-build" "$RUBY_VERSION_PIN" "$RUBY_PREFIX"
  rm -rf "$tmp"
  echo "built: $("$RUBY_PREFIX/bin/ruby" --version)"
fi
for b in ruby gem bundle bundler erb rake irb; do
  [ -x "$RUBY_PREFIX/bin/$b" ] && sudo ln -sf "$RUBY_PREFIX/bin/$b" "$BIN_DIR/$b"
done

log "bundle install (jekyll / github-pages)"
bundle install
# The `jekyll` executable is a bundle-provided gem wrapper in the ruby prefix's
# bin; `bundle exec jekyll` resolves it via PATH, so expose it in /usr/local/bin.
[ -x "$RUBY_PREFIX/bin/jekyll" ] && sudo ln -sf "$RUBY_PREFIX/bin/jekyll" "$BIN_DIR/jekyll"
echo "installed: $(bundle exec jekyll --version 2>/dev/null | tail -1)"

log "install complete"
