#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
ci="$root/.github/workflows/ci.yml"
pages="$root/.github/workflows/pages.yml"

fail() { echo "workflow-wiring: $1" >&2; exit 1; }

grep -Fq './scripts/verify-website-capabilities.sh' "$ci" || fail "CI missing built Seed 5 entrypoint"
grep -Fq 'needs: gate' "$ci" || fail "CI website job must block after the gate"
if grep -E 'verify-website-capabilities.sh.*--deployed-origin' "$ci" >/dev/null; then
  fail "CI must not run deployed mode"
fi
grep -Fq 'website-capabilities' "$ci" || fail "CI missing website-capabilities job"

grep -Fq './scripts/verify-pages-deployment.sh' "$pages" || fail "Pages missing SPEC-075 proof"
grep -Fq './scripts/verify-website-capabilities.sh' "$pages" || fail "Pages missing Seed 5 deployed entrypoint"
grep -Fq '--deployed-origin https://backstop.sh' "$pages" || fail "Pages must use canonical HTTPS origin"
grep -Fq 'BACKSTOP_DEPLOY_COMMIT' "$pages" || fail "Pages missing authoritative commit"
grep -Fq 'BACKSTOP_DEPLOY_RUN_ID' "$pages" || fail "Pages missing authoritative run id"

python3 - "$pages" <<'PY'
from pathlib import Path
import sys
text = Path(sys.argv[1]).read_text()
proof = text.find("./scripts/verify-pages-deployment.sh")
accept = text.find("./scripts/verify-website-capabilities.sh")
if proof < 0 or accept < 0 or accept < proof:
    raise SystemExit("Pages deployed acceptance must follow SPEC-075 proof")
if "rollback" in text.lower():
    raise SystemExit("Pages must not claim rollback")
PY

! grep -Eiq 'rollback' "$ci" "$pages" || fail "workflows must not claim rollback"
echo "workflow-wiring: ok"
