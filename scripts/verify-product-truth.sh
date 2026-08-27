#!/usr/bin/env bash
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"

verify_product_truth_pipeline() {
  local scratch profile summary total_count numeric_total
  scratch=$(mktemp -d "$ROOT/tmp/product-truth.XXXXXX")
  profile=$scratch/cover.out
  trap "rm -rf '$scratch'" EXIT HUP INT TERM

  go test ./scripts/producttruth/... -race -covermode=atomic -coverprofile="$profile"
  summary=$(go tool cover -func="$profile" | awk '$1 == "total:" { print $3 }')
  total_count=$(printf '%s\n' "$summary" | awk 'NF { count++ } END { print count+0 }')
  numeric_total=${summary%%%}
  if [ "$total_count" -ne 1 ] || ! printf '%s\n' "$numeric_total" | grep -Eq '^[0-9]+([.][0-9]+)?$'; then
    echo "product-truth[PT205_COVERAGE] job=pipeline output=- inputs=scripts/producttruth: expected exactly one numeric total" >&2
    return 1
  fi
  if ! awk -v total="$numeric_total" 'BEGIN { exit !(total >= 80.00) }'; then
    echo "product-truth[PT205_COVERAGE] job=pipeline output=- inputs=scripts/producttruth: coverage below 80.00 (${numeric_total})" >&2
    return 1
  fi

  ./scripts/generate-product-truth.sh --check
  go test ./scripts/producttruth/... -run 'SourceIncludes|SourceRejectsParallel'
  ./scripts/tests/product-truth/real-repository.sh
  printf 'product truth pipeline: pass (%s%% coverage)\n' "$numeric_total"
}

mkdir -p "$ROOT/tmp"
verify_product_truth_pipeline "$@"
