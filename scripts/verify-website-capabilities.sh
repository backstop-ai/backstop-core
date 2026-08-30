#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

SELF_TEST="${BACKSTOP_WEBSITE_SELF_TEST:-0}"
PREREQ_FAIL="${BACKSTOP_WEBSITE_PREREQ_FAIL:-}"
STATE=""
BEFORE_STATUS=""

fail() {
  echo "website-capabilities: $1" >&2
  exit 1
}

source_status() {
  git status --porcelain=v1 --untracked-files=all
}

cleanup_website_capabilities() {
  local status=0
  if [[ -n ${STATE:-} && -d $STATE ]]; then
    find "$STATE" -depth -delete || status=1
  fi
  if [[ -n ${BEFORE_STATUS:-} ]]; then
    local after
    after="$(source_status)"
    if [[ $after != "$BEFORE_STATUS" ]]; then
      echo "website-capabilities[cleanup]: residual temporary state remains" >&2
      status=1
    fi
  fi
  return "$status"
}

parse_coverage_total() {
  local text="$1"
  local matches
  matches="$(printf '%s\n' "$text" | grep -E 'coverage: [0-9]+\.[0-9]+% of statements' || true)"
  local count
  count="$(printf '%s\n' "$matches" | grep -c . || true)"
  if [[ $count -eq 0 ]]; then
    echo "absent"
    return
  fi
  if [[ $count -gt 1 ]]; then
    echo "duplicate"
    return
  fi
  local total
  total="$(printf '%s\n' "$matches" | sed -E 's/.*coverage: ([0-9]+\.[0-9]+)%.*/\1/')"
  if [[ ! $total =~ ^[0-9]+\.[0-9]+$ ]]; then
    echo "nonnumeric"
    return
  fi
  echo "$total"
}

accept_coverage_total() {
  local total="$1"
  case "$total" in
    absent|duplicate|nonnumeric)
      fail "coverage total is $total"
      ;;
  esac
  if ! python3 - "$total" <<'PY'
import sys
total=float(sys.argv[1])
raise SystemExit(0 if total >= 80.00 else 1)
PY
  then
    fail "coverage total $total is below 80.00"
  fi
}

assert_integration_consumer() {
  if grep -R --include='*.go' --exclude='*_test.go' -E 'scripts/sitecheck|scripts/producttruth|"jekyll"' scripts/websitejourney >/dev/null; then
    fail "Seed 5 must remain an integration consumer"
  fi
  if find scripts/websitejourney -name '*.js' | grep -q .; then
    fail "published runtime is prohibited"
  fi
}

run_prerequisites() {
  local commands=(
    ./scripts/verify-public-product-model.sh
    ./scripts/verify-documentation-semantics-integration.sh
    ./scripts/verify-product-truth.sh
    ./scripts/verify-public-site.sh
  )
  local command
  for command in "${commands[@]}"; do
    if [[ -n $PREREQ_FAIL && $command == *"$PREREQ_FAIL"* ]]; then
      fail "seed dependency $PREREQ_FAIL failed before traversal"
    fi
    if [[ $SELF_TEST == 1 ]]; then
      continue
    fi
    "$command" || fail "governed dependency $command failed before traversal"
  done
}

if [[ ${1:-} == --parse-coverage ]]; then
  parse_coverage_total "$(cat)"
  exit 0
fi
if [[ ${1:-} == --accept-coverage ]]; then
  accept_coverage_total "${2:-}"
  echo "website-capabilities: coverage ${2:-} accepted"
  exit 0
fi
if [[ ${1:-} == --assert-consumer ]]; then
  assert_integration_consumer
  echo "website-capabilities: integration consumer"
  exit 0
fi

deployed_origin=""
deploy_commit=""
deploy_run_id=""
capability=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --deployed-origin)
      deployed_origin="${2:-}"
      shift 2
      ;;
    --commit)
      deploy_commit="${2:-}"
      shift 2
      ;;
    --run-id)
      deploy_run_id="${2:-}"
      shift 2
      ;;
    --capability)
      capability="${2:-}"
      shift 2
      ;;
    *)
      fail "unknown argument: $1"
      ;;
  esac
done

if [[ -n $deployed_origin ]]; then
  if [[ -z $deploy_commit || -z $deploy_run_id ]]; then
    fail "--deployed-origin requires --commit and --run-id"
  fi
  assert_integration_consumer
  deployed_args=(--deployed-origin "$deployed_origin" --commit "$deploy_commit" --run-id "$deploy_run_id")
  if [[ -n $capability ]]; then
    deployed_args+=(--capability "$capability")
  fi
  go run ./scripts/websitejourney "${deployed_args[@]}"
  echo "website-capabilities: deployed journeys verified"
  exit 0
fi

trap 'cleanup_website_capabilities' EXIT

BEFORE_STATUS="$(source_status)"
STATE="$(mktemp -d "$root/.backstop-website-capabilities.XXXXXX")"

assert_integration_consumer
run_prerequisites

coverprofile="$STATE/cover.out"
test_output="$STATE/go-test.txt"
if [[ $SELF_TEST == 1 ]]; then
  printf '%s\n' "ok 	github.com/backstop-ai/backstop-core/scripts/websitejourney	0.1s	coverage: 80.00% of statements" >"$test_output"
else
  go test ./scripts/websitejourney/ -count=1 -coverprofile="$coverprofile" | tee "$test_output"
fi

total="$(parse_coverage_total "$(cat "$test_output")")"
accept_coverage_total "$total"
echo "website-capabilities: verified (coverage $total)"
