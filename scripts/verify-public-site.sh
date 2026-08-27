#!/usr/bin/env bash
set -euo pipefail

verify_public_site_delivery() {
  if [[ $# -ne 0 ]]; then
    echo "public-site[arguments]: expected no arguments" >&2
    return 2
  fi

  local root commit profile summary total_count numeric_total exit_code=0
  root=$(git rev-parse --show-toplevel)
  cd "$root"
  PUBLIC_SITE_STATE=$(mktemp -d "$root/.backstop-public-site-state.XXXXXX")
  PUBLIC_SITE_RETAIN=${BACKSTOP_SITE_RETAIN:-0}
  commit=${BACKSTOP_SITE_COMMIT:-$(git rev-parse HEAD)}
  if [[ ! $commit =~ ^[0-9a-f]{40}$ ]] || [[ $commit != "$(git rev-parse HEAD)" ]]; then
    echo "public-site[commit]: expected the tested full HEAD commit, observed $commit" >&2
    find "$PUBLIC_SITE_STATE" -depth -delete
    return 2
  fi
  if [[ $PUBLIC_SITE_RETAIN != 0 && $PUBLIC_SITE_RETAIN != 1 ]]; then
    echo "public-site[retain]: expected 0 or 1, observed $PUBLIC_SITE_RETAIN" >&2
    find "$PUBLIC_SITE_STATE" -depth -delete
    return 2
  fi
  if [[ -n ${BACKSTOP_SITE_OUTPUT:-} ]]; then
    PUBLIC_SITE_OUTPUT=$(realpath -m "$BACKSTOP_SITE_OUTPUT")
    if [[ $PUBLIC_SITE_OUTPUT != "$root/_site" ]]; then
      echo "public-site[output]: retained output must be $root/_site" >&2
      find "$PUBLIC_SITE_STATE" -depth -delete
      return 2
    fi
    if [[ -e $PUBLIC_SITE_OUTPUT ]]; then
      echo "public-site[output]: refusing root output collision at $PUBLIC_SITE_OUTPUT" >&2
      find "$PUBLIC_SITE_STATE" -depth -delete
      return 1
    fi
  else
    PUBLIC_SITE_OUTPUT=$(mktemp -d "$root/.backstop-public-site-build.XXXXXX")
  fi
  source_status() {
    git status --porcelain=v1 --untracked-files=all | awk -v retain="$PUBLIC_SITE_RETAIN" '
      retain == "1" && $0 ~ /^\?\? _site\// { next }
      { print }
    '
  }
  PUBLIC_SITE_BEFORE_STATUS=$(source_status)

  cleanup_public_site() {
    local cleanup_status
    if [[ -d $PUBLIC_SITE_STATE ]]; then
      find "$PUBLIC_SITE_STATE" -depth -delete
    fi
    if [[ $PUBLIC_SITE_RETAIN == 0 && -d $PUBLIC_SITE_OUTPUT ]]; then
      find "$PUBLIC_SITE_OUTPUT" -depth -delete
    fi
    cleanup_status=$(source_status)
    if [[ $cleanup_status != "$PUBLIC_SITE_BEFORE_STATUS" ]]; then
      echo "public-site[cleanup]: checked-in source or residual state changed" >&2
      diff <(printf '%s\n' "$PUBLIC_SITE_BEFORE_STATUS") <(printf '%s\n' "$cleanup_status") >&2 || true
      return 1
    fi
  }
  trap 'exit_code=$?; cleanup_public_site || exit_code=1; exit "$exit_code"' EXIT HUP INT TERM

  profile="$PUBLIC_SITE_STATE/sitecheck.cover"
  echo "public-site[coverage]: race-enabled atomic sitecheck suite"
  go test ./scripts/sitecheck/... -race -covermode=atomic -coverprofile="$profile"
  summary=$(go tool cover -func="$profile" | awk '$1 == "total:" { print $3 }')
  total_count=$(printf '%s\n' "$summary" | awk 'NF { count++ } END { print count+0 }')
  numeric_total=${summary%%%}
  if [[ $total_count -ne 1 ]] || ! grep -Eq '^[0-9]+([.][0-9]+)?$' <<<"$numeric_total"; then
    echo "public-site[coverage]: expected exactly one numeric total, observed ${summary:-absent}" >&2
    return 1
  fi
  if ! awk -v total="$numeric_total" 'BEGIN { exit !(total >= 80.00) }'; then
    echo "public-site[coverage]: expected >=80.00, observed $numeric_total" >&2
    return 1
  fi

  echo "public-site[documentation-semantics]: released integration"
  ./scripts/verify-documentation-semantics-integration.sh
  echo "public-site[product-truth]: deterministic check mode"
  ./scripts/generate-product-truth.sh --check

  echo "public-site[jekyll]: locked production build"
  if command -v bundle >/dev/null 2>&1; then
    JEKYLL_ENV=production bundle exec jekyll build --source docs --destination "$PUBLIC_SITE_OUTPUT" --trace
  elif [[ -n ${BACKSTOP_RUBY:-} && -x ${BACKSTOP_RUBY} && -n ${BACKSTOP_BUNDLE_SCRIPT:-} && -f ${BACKSTOP_BUNDLE_SCRIPT} ]]; then
    JEKYLL_ENV=production "$BACKSTOP_RUBY" "$BACKSTOP_BUNDLE_SCRIPT" exec jekyll build --source docs --destination "$PUBLIC_SITE_OUTPUT" --trace
  else
    echo "public-site[jekyll]: bundle is unavailable" >&2
    return 2
  fi

  echo "public-site[owner-assets]: immutable design-system token bytes"
  ./scripts/install-design-assets.sh .backstop/packs/backstop-ai/backstop-design-system "$PUBLIC_SITE_OUTPUT"
  echo "public-site[annotation]: structured owner contracts at $commit"
  go run ./scripts/render-public-site-contracts --root "$root" --built-root "$PUBLIC_SITE_OUTPUT" --site-commit "$commit"
  echo "public-site[structure]: routes, links, ownership, Pages, and eight-corpus design matrix"
  go run ./scripts/sitecheck --root "$root" --check-diff --built-root "$PUBLIC_SITE_OUTPUT" --site-commit "$commit" --design-system-matrix

  echo "public-site[browser]: Chromium no-JavaScript viewport and 200-percent matrix"
  PUBLIC_SITE_ROOT="$PUBLIC_SITE_OUTPUT" PLAYWRIGHT_OUTPUT_DIR="$PUBLIC_SITE_STATE/playwright" npx playwright test
  echo "public-site[pass]: commit=$commit coverage=${numeric_total}% output=$PUBLIC_SITE_OUTPUT"
}

verify_public_site_delivery "$@"
