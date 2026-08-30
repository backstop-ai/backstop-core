#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
wrapper="$root/scripts/verify-website-capabilities.sh"

TestWebsiteJourney_VerificationCleanupPasses() {
  local before after
  before="$(cd "$root" && git status --porcelain=v1 --untracked-files=all)"
  BACKSTOP_WEBSITE_SELF_TEST=1 "$wrapper"
  after="$(cd "$root" && git status --porcelain=v1 --untracked-files=all)"
  [[ $before == "$after" ]]
  ! find "$root" -maxdepth 1 -type d -name '.backstop-website-capabilities.*' | grep -q .
}

TestWebsiteJourney_VerificationCleanupOnFailure() {
  local before after
  before="$(cd "$root" && git status --porcelain=v1 --untracked-files=all)"
  if BACKSTOP_WEBSITE_SELF_TEST=1 BACKSTOP_WEBSITE_PREREQ_FAIL=verify-public-product-model "$wrapper"; then
    echo "expected governed dependency failure" >&2
    return 1
  fi
  after="$(cd "$root" && git status --porcelain=v1 --untracked-files=all)"
  [[ $before == "$after" ]]
}

TestWebsiteJourney_RemainsIntegrationConsumer() {
  "$wrapper" --assert-consumer
}

TestWebsiteJourney_WrapperPropagatesGovernedDependencyFailure() {
  local output
  if output="$(BACKSTOP_WEBSITE_SELF_TEST=1 BACKSTOP_WEBSITE_PREREQ_FAIL=verify-public-product-model "$wrapper" 2>&1)"; then
    echo "wrapper accepted a failed governed dependency" >&2
    return 1
  fi
  printf '%s\n' "$output" | grep -Fq 'verify-public-product-model'
  printf '%s\n' "$output" | grep -Fq 'before traversal'
}

TestWebsiteJourney_VerificationCleanupPasses
TestWebsiteJourney_VerificationCleanupOnFailure
TestWebsiteJourney_RemainsIntegrationConsumer
TestWebsiteJourney_WrapperPropagatesGovernedDependencyFailure
echo "verify-wrapper: ok"
