#!/usr/bin/env bash
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
cd "$ROOT"

before=$(git diff -- docs/_includes/generated)
./scripts/generate-product-truth.sh --write
./scripts/generate-product-truth.sh
./scripts/generate-product-truth.sh --check
after=$(git diff -- docs/_includes/generated)

if [ "$before" != "$after" ]; then
  echo "product-truth[PT202_DRIFT] job=pipeline output=docs/_includes/generated inputs=repository: repeated write changed the generated cohort" >&2
  exit 1
fi

test "$(find docs/_includes/generated -maxdepth 1 -type f -name '*.md' | wc -l)" -eq 4
test ! -e docs/_includes/.product-truth-transaction
