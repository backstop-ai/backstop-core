#!/usr/bin/env bash
# Fixes vendored ripgrep permissions in Claude Code.
# npm install strips execute bits from the rg binary, which breaks
# Grep/Glob tools and agent file discovery.
# Run after: npm install -g @anthropic-ai/claude-code
# See: https://github.com/anthropics/claude-code/issues/11205

set -euo pipefail

RG_DIR="$(npm root -g)/@anthropic-ai/claude-code/vendor/ripgrep"

if [ ! -d "$RG_DIR" ]; then
  echo "Claude Code ripgrep directory not found at $RG_DIR"
  exit 1
fi

find "$RG_DIR" -name "rg" -exec chmod +x {} \;
echo "Fixed ripgrep permissions in $RG_DIR"
