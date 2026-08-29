# Base image for the backstop-core Cloud Agent environment.
#
# It bakes the STABLE, heavyweight toolchain so environment BUILDS are fast and
# cache-friendly: the language runtimes (Go, Node, Python), the analyzer fleet
# the gate provisions by name (golangci-lint, go-arch-lint, semgrep, ast-grep),
# Ruby 3.3.4 for the Jekyll site build, and the Playwright Chromium browser +
# its OS libraries. Everything here changes rarely, so an edit to the
# repo-coupled install script no longer recompiles Ruby or re-downloads Chromium.
#
# Repo-coupled work (go mod download, building the backstop CLI, pack install,
# npm ci, bundle install) stays in .cursor/install.sh. That script is idempotent:
# its version guards detect the tools baked here and skip straight to the
# repo-coupled steps. Nothing from the repository is copied into this image —
# Cursor checks out the source separately and layers its own agent assets
# (exec-daemon, gh, chrome) on top at build time.
#
# Pins are kept in lockstep with .github/workflows/ci.yml, pkg/pack/engine/
# allowlist.go, go.mod, and .cursor/install.sh.
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV PATH=/usr/local/go/bin:/usr/local/bin:$PATH
ENV GOTOOLCHAIN=auto

# --- OS packages: VCS, archive tools, Ruby build deps, Python venv -----------
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates curl git unzip xz-utils sudo gnupg \
        build-essential autoconf patch pkg-config \
        libssl-dev libyaml-dev zlib1g-dev libreadline-dev libffi-dev libgdbm-dev libncurses-dev \
        python3 python3-venv python3-pip \
    && rm -rf /var/lib/apt/lists/*

# The base image ships a `ubuntu` user (uid 1000); grant it passwordless sudo so
# install.sh (which runs as this user) can place binaries in /usr/local/bin.
RUN usermod -aG sudo ubuntu \
    && echo 'ubuntu ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/ubuntu \
    && chmod 0440 /etc/sudoers.d/ubuntu

# --- Go (go.mod pins 1.25.3; GOTOOLCHAIN=auto fetches newer when a tool asks) -
# Symlink into /usr/local/bin because the install phase runs with a standard
# PATH that does NOT inherit this image's ENV PATH — /usr/local/bin is always on
# it, /usr/local/go/bin is not.
RUN curl -fsSL https://go.dev/dl/go1.25.3.linux-amd64.tar.gz -o /tmp/go.tgz \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm /tmp/go.tgz \
    && ln -sf /usr/local/go/bin/go /usr/local/bin/go \
    && ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# --- Node.js 22 (Playwright harness + npm) -----------------------------------
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# --- Analyzer fleet (pins match CI + install.sh) -----------------------------
RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh \
      | sh -s -- -b /usr/local/bin v2.6.0
RUN curl -sSfL https://github.com/ast-grep/ast-grep/releases/download/0.43.0/app-x86_64-unknown-linux-gnu.zip -o /tmp/ast.zip \
    && unzip -o -q /tmp/ast.zip -d /usr/local/bin \
    && rm /tmp/ast.zip \
    && chmod +x /usr/local/bin/ast-grep
RUN python3 -m venv /opt/semgrep-venv \
    && /opt/semgrep-venv/bin/pip install --no-cache-dir --upgrade pip \
    && /opt/semgrep-venv/bin/pip install --no-cache-dir semgrep==1.156.0 \
    && ln -sf /opt/semgrep-venv/bin/semgrep /usr/local/bin/semgrep
RUN GOBIN=/usr/local/bin go install github.com/fe3dback/go-arch-lint@v1.16.0 \
    && rm -rf /root/go/pkg/mod /root/.cache/go-build

# --- Ruby 3.3.4 (Jekyll site build; default Bundler 2.5.11 matches lockfile) -
RUN git clone --depth 1 https://github.com/rbenv/ruby-build.git /tmp/ruby-build \
    && /tmp/ruby-build/bin/ruby-build 3.3.4 /opt/ruby-3.3.4 \
    && rm -rf /tmp/ruby-build \
    && for b in ruby gem bundle bundler erb rake irb; do \
         ln -sf "/opt/ruby-3.3.4/bin/$b" "/usr/local/bin/$b"; \
       done

# --- Playwright Chromium + OS libraries --------------------------------------
# OS deps as root (apt); the browser as the ubuntu user so it lands in that
# user's ~/.cache/ms-playwright — exactly where @playwright/test resolves it at
# runtime. Pinned to 1.55.0 to match package-lock.json's Chromium build.
RUN npx --yes playwright@1.55.0 install-deps chromium
USER ubuntu
RUN HOME=/home/ubuntu npx --yes playwright@1.55.0 install chromium
USER root
