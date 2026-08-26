#!/usr/bin/env bash
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)

TestProductTruth_ReleaseBlocksStaleLatestMain() {
  local release=$ROOT/.github/workflows/release.yml
  grep -q 'release-history-current:' "$release"
  grep -q 'generate-product-truth.sh --check' "$release"
  grep -q 'origin/main' "$release"
}

TestProductTruth_ReleaseHandshakePassesAfterMainRegeneration() {
  local release=$ROOT/.github/workflows/release.yml
  grep -q 'fetch-depth: 0' "$release"
  grep -q 'github.ref_name' "$release"
  grep -Eq 'needs:.*release-history-current' "$release"
}

TestProductTruth_ReleaseWorkflowRejectsTagCheckoutSubstitution() {
  local release=$ROOT/.github/workflows/release.yml
  local block
  block=$(sed -n '/release-history-current:/,/^  [a-zA-Z0-9_-]*:/p' "$release")
  printf '%s' "$block" | grep -q 'ref: main'
  printf '%s' "$block" | grep -q 'git fetch.*refs/tags'
  ! printf '%s' "$block" | grep -q 'ref:.*github.ref'
}

TestProductTruth_ReleaseBlocksStaleLatestMain
TestProductTruth_ReleaseHandshakePassesAfterMainRegeneration
TestProductTruth_ReleaseWorkflowRejectsTagCheckoutSubstitution
