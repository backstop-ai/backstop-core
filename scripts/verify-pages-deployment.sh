#!/usr/bin/env bash
set -euo pipefail

verify_pages_deployment() {
  local repository="" run_id="" commit="" artifact_id="" page_url=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --repository) repository=${2-}; shift 2 ;;
      --run-id) run_id=${2-}; shift 2 ;;
      --commit) commit=${2-}; shift 2 ;;
      --artifact-id) artifact_id=${2-}; shift 2 ;;
      --page-url) page_url=${2-}; shift 2 ;;
      *) echo "pages-verify: unknown argument: $1" >&2; return 2 ;;
    esac
  done

  [[ $repository =~ ^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$ ]] || { echo "pages-verify: invalid repository" >&2; return 2; }
  [[ $run_id =~ ^[1-9][0-9]*$ ]] || { echo "pages-verify: invalid run ID" >&2; return 2; }
  [[ $commit =~ ^[0-9a-f]{40}$ ]] || { echo "pages-verify: invalid commit" >&2; return 2; }
  [[ $artifact_id =~ ^[1-9][0-9]*$ ]] || { echo "pages-verify: invalid artifact ID" >&2; return 2; }
  [[ $page_url == "https://backstop.sh/" ]] || { echo "pages-verify: page URL must be https://backstop.sh/" >&2; return 1; }
  command -v gh >/dev/null || { echo "pages-verify: gh is required" >&2; return 2; }
  command -v curl >/dev/null || { echo "pages-verify: curl is required" >&2; return 2; }

  local scratch
  scratch=$(mktemp -d "${TMPDIR:-/tmp}/backstop-pages-proof.XXXXXX")
  trap 'chmod -R u+w "$scratch" 2>/dev/null || true; find "$scratch" -depth -delete 2>/dev/null || true' RETURN

  gh api "repos/$repository/pages" >"$scratch/pages.json"
  gh api "repos/$repository/actions/runs/$run_id" >"$scratch/run.json"
  gh api "repos/$repository/actions/artifacts/$artifact_id" >"$scratch/artifact.json"
  gh api "repos/$repository/deployments?sha=$commit&environment=github-pages&per_page=100" >"$scratch/deployments.json"

  local deployment_id
  deployment_id=$(python3 - "$scratch" "$run_id" "$commit" "$artifact_id" <<'PY'
import json, pathlib, re, sys
root = pathlib.Path(sys.argv[1])
run_id, commit, artifact_id = int(sys.argv[2]), sys.argv[3], int(sys.argv[4])
pages = json.loads((root / "pages.json").read_text())
run = json.loads((root / "run.json").read_text())
artifact = json.loads((root / "artifact.json").read_text())
deployments = json.loads((root / "deployments.json").read_text())
errors = []
if pages.get("build_type") != "workflow": errors.append(f"Pages build_type={pages.get('build_type')!r}")
if pages.get("cname") != "backstop.sh": errors.append(f"Pages cname={pages.get('cname')!r}")
if pages.get("https_enforced") is not True: errors.append("Pages HTTPS is not enforced")
if run.get("id") != run_id or run.get("head_sha") != commit: errors.append("workflow run identity mismatch")
if run.get("path") != ".github/workflows/pages.yml": errors.append(f"workflow path={run.get('path')!r}")
if artifact.get("id") != artifact_id: errors.append("artifact ID mismatch")
if artifact.get("name") != "github-pages": errors.append(f"artifact name={artifact.get('name')!r}")
if artifact.get("expired") is not False: errors.append("artifact is expired")
if artifact.get("workflow_run", {}).get("id") != run_id: errors.append("artifact workflow-run mismatch")
if not re.fullmatch(r"sha256:[0-9a-f]{64}", artifact.get("digest", "")): errors.append("artifact archive digest missing")
matching = [d for d in deployments if d.get("sha") == commit and d.get("environment") == "github-pages"]
if len(matching) != 1: errors.append(f"expected one github-pages deployment, observed {len(matching)}")
if errors:
    raise SystemExit("pages-verify: authoritative API: " + "; ".join(errors))
print(matching[0]["id"])
PY
  )
  gh api "repos/$repository/deployments/$deployment_id/statuses?per_page=100" >"$scratch/statuses.json"
  python3 - "$scratch/statuses.json" "$page_url" <<'PY'
import json, sys
statuses = json.load(open(sys.argv[1]))
page_url = sys.argv[2]
successes = [s for s in statuses if s.get("state") == "success" and s.get("environment_url") == page_url]
if not successes:
    raise SystemExit("pages-verify: no successful deployment status with the expected environment URL")
PY

  gh api "repos/$repository/actions/artifacts/$artifact_id/zip" >"$scratch/artifact.zip"
  python3 - "$scratch/artifact.json" "$scratch/artifact.zip" <<'PY'
import hashlib, json, pathlib, sys
record = json.load(open(sys.argv[1]))
observed = "sha256:" + hashlib.sha256(pathlib.Path(sys.argv[2]).read_bytes()).hexdigest()
if observed != record["digest"]:
    raise SystemExit(f"pages-verify: archive digest mismatch expected={record['digest']} observed={observed}")
PY
  mkdir -p "$scratch/archive" "$scratch/tree"
  unzip -q "$scratch/artifact.zip" -d "$scratch/archive"
  local archive_tar
  archive_tar=$(find "$scratch/archive" -type f -name '*.tar' -print -quit)
  [[ -n $archive_tar ]] || { echo "pages-verify: retained Pages artifact contains no tar archive" >&2; return 1; }
  tar -xf "$archive_tar" -C "$scratch/tree"

  curl --fail --silent --show-error --proto '=https' --tlsv1.2 \
    "https://backstop.sh/.well-known/backstop-deployment.json" >"$scratch/remote-marker.json"
  python3 - "$scratch/tree" "$scratch/remote-marker.json" "$commit" "$run_id" <<'PY'
import hashlib, json, pathlib, sys
root = pathlib.Path(sys.argv[1])
remote = json.load(open(sys.argv[2]))
commit, run_id = sys.argv[3], int(sys.argv[4])
local_marker = json.loads((root / ".well-known" / "backstop-deployment.json").read_text())
digest = hashlib.sha256()
marker_path = root / ".well-known" / "backstop-deployment.json"
for path in sorted((p for p in root.rglob("*") if p.is_file() and p != marker_path), key=lambda p: p.relative_to(root).as_posix().encode()):
    relative = path.relative_to(root).as_posix().encode(); data = path.read_bytes()
    digest.update(len(relative).to_bytes(8, "big")); digest.update(relative)
    digest.update(len(data).to_bytes(8, "big")); digest.update(data)
expected = {"schema_version": "backstop-core/pages-deployment/v1", "commit": commit, "run_id": run_id, "tree_content_sha256": digest.hexdigest()}
if local_marker != expected: raise SystemExit(f"pages-verify: retained marker mismatch expected={expected} observed={local_marker}")
if remote != expected: raise SystemExit(f"pages-verify: remote marker mismatch expected={expected} observed={remote}")
PY

  local marker='<meta name="backstop-deployment" content="commit='"$commit"';run='"$run_id"'">'
  local route body
  for route in / /evaluate/ /model/ /adopt/ /use-cases/ /packs/ /extend/ /reference/ /status/ /contributing/; do
    body=$(curl --fail --silent --show-error --proto '=https' --tlsv1.2 --max-redirs 0 "https://backstop.sh$route")
    [[ $(grep -Foc "$marker" <<<"$body") -eq 1 ]] || { echo "pages-verify: $route deployment marker mismatch" >&2; return 1; }
  done
  local alias destination alias_body
  while IFS=' ' read -r alias destination; do
    alias_body=$(curl --fail --silent --show-error --proto '=https' --tlsv1.2 --max-redirs 0 "https://backstop.sh$alias")
    grep -Fq 'href="https://backstop.sh'"${destination}"'"' <<<"$alias_body" || { echo "pages-verify: $alias canonical target mismatch" >&2; return 1; }
    grep -Fq 'content="0; url='"${destination}"'"' <<<"$alias_body" || { echo "pages-verify: $alias immediate refresh mismatch" >&2; return 1; }
    grep -Fq 'href="'"${destination}"'"' <<<"$alias_body" || { echo "pages-verify: $alias fallback target mismatch" >&2; return 1; }
    ! grep -Eiq '<script([[:space:]>])' <<<"$alias_body" || { echo "pages-verify: $alias contains a client-scripted redirect" >&2; return 1; }
  done <<'ALIASES'
/getting-started.html /adopt/
/concepts.html /model/
/artifact-workflow.html /model/
/pack-authoring.html /extend/
/cli-reference.html /reference/
ALIASES
  echo "pages-verify: authoritative API, retained artifact, and HTTPS publication agree for $commit run $run_id"
}

verify_pages_deployment "$@"
