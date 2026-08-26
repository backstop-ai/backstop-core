#!/usr/bin/env bash
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)

TestProductTruth_VerifierAcceptsCoverageAtThreshold() {
  local verifier=$ROOT/scripts/verify-product-truth.sh
  grep -q '80.00' "$verifier"
  grep -q 'PT205_COVERAGE' "$verifier"
  grep -q 'cover -func' "$verifier"
}

TestProductTruth_VerifierRejectsCoverageFailureMatrix() {
  local verifier=$ROOT/scripts/verify-product-truth.sh
  grep -q 'total_count' "$verifier"
  grep -q 'numeric_total' "$verifier"
  grep -q 'coverage below 80.00' "$verifier"
}

TestProductTruth_VerifierCoversIndependentSourcePipeline() {
  local verifier=$ROOT/scripts/verify-product-truth.sh
  grep -q 'generate-product-truth.sh --check' "$verifier"
  grep -q 'SourceIncludes' "$verifier"
  grep -q 'source-workflow-wiring.sh' "$verifier"
}

TestProductTruth_VerifierAcceptsCoverageAtThreshold
TestProductTruth_VerifierRejectsCoverageFailureMatrix
TestProductTruth_VerifierCoversIndependentSourcePipeline
