#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: install-design-assets.sh <installed-pack-root> <built-site-root>" >&2
  exit 2
fi

pack_root=$1
built_root=$2
export_path="$pack_root/contracts/public-site-acceptance.yml"

if [[ ! -f "$export_path" ]]; then
  echo "design-assets: owner export missing: $export_path" >&2
  exit 1
fi

token_block=$(awk '
  /^token_asset:/ { active=1; next }
  active && /^[^[:space:]]/ { exit }
  active { print }
' "$export_path")
installed_relative_path=$(sed -n 's/^  installed_relative_path: //p' <<<"$token_block")
expected_sha=$(sed -n 's/^  sha256: //p' <<<"$token_block")
public_output=$(sed -n 's/^  public_output: //p' <<<"$token_block")

if [[ -z "$installed_relative_path" || -z "$expected_sha" || -z "$public_output" ]]; then
  echo "design-assets: incomplete token_asset contract" >&2
  exit 1
fi

source_path="$pack_root/$installed_relative_path"
output_path="$built_root/$public_output"
if [[ ! -f "$source_path" ]]; then
  echo "design-assets: owner token asset missing: $installed_relative_path" >&2
  exit 1
fi

observed_sha=$(sha256sum "$source_path" | awk '{print $1}')
if [[ "$observed_sha" != "$expected_sha" ]]; then
  echo "design-assets: owner token digest mismatch: expected=$expected_sha observed=$observed_sha" >&2
  exit 1
fi

mkdir -p "$(dirname "$output_path")"
install -m 0644 "$source_path" "$output_path"
copied_sha=$(sha256sum "$output_path" | awk '{print $1}')
if [[ "$copied_sha" != "$expected_sha" ]]; then
  echo "design-assets: copied token digest mismatch: expected=$expected_sha observed=$copied_sha" >&2
  exit 1
fi

echo "design-assets: installed $public_output sha256:$copied_sha"
