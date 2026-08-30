#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
wrapper="$root/scripts/verify-website-capabilities.sh"

accept() {
  "$wrapper" --accept-coverage "$1"
}

reject() {
  local total="$1" token="$2" output
  if output="$("$wrapper" --accept-coverage "$total" 2>&1)"; then
    echo "accepted coverage $total" >&2
    return 1
  fi
  printf '%s\n' "$output" | grep -Fq "$token"
}

parse_eq() {
  local got
  got="$("$wrapper" --parse-coverage <<<"$1")"
  [[ $got == "$2" ]]
}

TestWebsiteJourney_VerifierAcceptsCoverageAtThreshold() {
  accept 80.00
  accept 80.01
  accept 100.00
  parse_eq $'ok\tpkg\t0.1s\tcoverage: 80.00% of statements' 80.00
}

TestWebsiteJourney_VerifierRejectsCoverageFailureMatrix() {
  reject 79.99 79.99
  reject absent absent
  reject duplicate duplicate
  reject nonnumeric nonnumeric
  parse_eq $'no coverage here' absent
  parse_eq $'coverage: 80.00% of statements\ncoverage: 81.00% of statements' duplicate
}

TestWebsiteJourney_VerifierAcceptsCoverageAtThreshold
TestWebsiteJourney_VerifierRejectsCoverageFailureMatrix
echo "coverage-cleanup-mutations: ok"
