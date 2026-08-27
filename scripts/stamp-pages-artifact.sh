#!/usr/bin/env bash
set -euo pipefail

stamp_pages_artifact() {
  if [[ $# -ne 5 || $1 != "--commit" || $3 != "--run-id" ]]; then
    echo "usage: stamp-pages-artifact.sh --commit <40-hex> --run-id <integer> <site-root>" >&2
    return 2
  fi

  local commit=$2
  local run_id=$4
  local site_root=$5
  if [[ ! $commit =~ ^[0-9a-f]{40}$ ]]; then
    echo "pages-stamp: commit must be full lowercase 40-hex" >&2
    return 2
  fi
  if [[ ! $run_id =~ ^[1-9][0-9]*$ ]]; then
    echo "pages-stamp: run ID must be a positive integer" >&2
    return 2
  fi
  if [[ ! -d $site_root ]]; then
    echo "pages-stamp: site root missing: $site_root" >&2
    return 1
  fi

  python3 - "$site_root" "$commit" "$run_id" <<'PY'
import hashlib
import json
import pathlib
import re
import sys

root = pathlib.Path(sys.argv[1]).resolve()
commit = sys.argv[2]
run_id = sys.argv[3]
routes = [
    "index.html", "evaluate/index.html", "model/index.html", "adopt/index.html",
    "use-cases/index.html", "packs/index.html", "extend/index.html",
    "reference/index.html", "status/index.html", "contributing/index.html",
]
marker = f'<meta name="backstop-deployment" content="commit={commit};run={run_id}">'
pattern = re.compile(rb'<meta name="backstop-deployment"[^>]*>')

for relative in routes:
    path = root / relative
    if not path.is_file():
        raise SystemExit(f"pages-stamp: canonical output missing: {relative}")
    data = path.read_bytes()
    if pattern.search(data):
        raise SystemExit(f"pages-stamp: deployment marker already present: {relative}")
    head = data.find(b"</head>")
    if head < 0:
        raise SystemExit(f"pages-stamp: closing head missing: {relative}")
    path.write_bytes(data[:head] + b"  " + marker.encode() + b"\n" + data[head:])

marker_path = root / ".well-known" / "backstop-deployment.json"
marker_path.parent.mkdir(parents=True, exist_ok=True)
digest = hashlib.sha256()
for path in sorted((p for p in root.rglob("*") if p.is_file() and p != marker_path), key=lambda p: p.relative_to(root).as_posix().encode()):
    relative = path.relative_to(root).as_posix().encode()
    data = path.read_bytes()
    digest.update(len(relative).to_bytes(8, "big"))
    digest.update(relative)
    digest.update(len(data).to_bytes(8, "big"))
    digest.update(data)
record = {
    "schema_version": "backstop-core/pages-deployment/v1",
    "commit": commit,
    "run_id": int(run_id),
    "tree_content_sha256": digest.hexdigest(),
}
marker_path.write_text(json.dumps(record, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
print(f"pages-stamp: commit={commit} run={run_id} tree_content_sha256={record['tree_content_sha256']}")
PY
}

stamp_pages_artifact "$@"
