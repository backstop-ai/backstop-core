#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
inventory="${repo_root}/docs/_data/content-inventory.yml"

validate_inventory() {
  python3 - "$1" <<'PY'
import sys, yaml
path=sys.argv[1]
d=yaml.safe_load(open(path, encoding='utf-8'))
expected_sources=['docs/index.html','docs/getting-started.md','docs/concepts.md','docs/artifact-workflow.md','docs/pack-authoring.md','docs/cli-reference.md']
expected_units={
'docs/index.html':[f'HOME-{i:03d}' for i in range(1,5)],
'docs/getting-started.md':[f'GET-{i:03d}' for i in range(1,6)],
'docs/concepts.md':[f'CON-{i:03d}' for i in range(1,7)],
'docs/artifact-workflow.md':[f'ART-{i:03d}' for i in range(1,6)],
'docs/pack-authoring.md':[f'PACK-{i:03d}' for i in range(1,7)],
'docs/cli-reference.md':[f'CLI-{i:03d}' for i in range(1,6)]}
sources=d.get('sources',[])
got=[x.get('source') for x in sources]
if got!=expected_sources:
  duplicate=next((x for x in got if got.count(x)>1),None); missing=next((x for x in expected_sources if x not in got),None); extra=next((x for x in got if x not in expected_sources),None)
  raise AssertionError(f'{duplicate or missing or extra}: source field inventory')
seen=[]
for src in sources:
  units=src.get('useful_units',[])
  ids=[u.get('unit_id') for u in units]
  if ids!=expected_units[src['source']]:
    expected=expected_units[src['source']]; duplicate=next((x for x in ids if ids.count(x)>1),None); missing=next((x for x in expected if x not in ids),None); extra=next((x for x in ids if x not in expected),None)
    raise AssertionError(f'{duplicate or extra or missing}: unit_id inventory for {src["source"]}')
  for u in units:
    uid=u.get('unit_id','<missing>'); seen.append(uid)
    for field in ('source_locator','topic','summary','disposition','rationale','target_routes'):
      assert field in u and u[field] not in ('',None), f'{uid}: missing {field}'
    disp=u['disposition']; targets=u['target_routes']
    assert disp in ('rewrite','merge','decompose','retain','retire'), f'{uid}: invalid disposition'
    required={'rewrite':1,'merge':1,'retain':1}.get(disp)
    if required is not None: assert len(targets)==required, f'{uid}: invalid target cardinality'
    if disp=='decompose': assert len(set(targets))>=2, f'{uid}: invalid target cardinality'
    if disp=='retire': assert not targets, f'{uid}: invalid target cardinality'
assert len(seen)==31 and len(set(seen))==31, 'useful unit IDs must be exactly 31 and unique'
PY
}

verify_legacy_content_disposition_inventory() { validate_inventory "$inventory"; }
verify_legacy_content_disposition_rejects_invalid_entry() {
  local mutation tmp
  local output expected
  if output="$(validate_inventory "${repo_root}/scripts/tests/public-product-model/fixtures/content-inventory-invalid.yml" 2>&1)"; then return 1; fi
  grep -Fq docs/index.html <<<"$output" || { echo "invalid fixture missing docs/index.html diagnostic: $output" >&2; return 1; }
  for mutation in duplicate missing extra malformed; do
  tmp="$(mktemp)"; cp "$inventory" "$tmp"
  python3 - "$tmp" "$mutation" <<'PY'
import sys,yaml
p,m=sys.argv[1:]; d=yaml.safe_load(open(p))
if m=='duplicate': d['sources'][1]['source']=d['sources'][0]['source']
elif m=='missing': d['sources']=d['sources'][1:]
elif m=='extra': d['sources'].append({'source':'docs/unknown.md','useful_units':[]})
elif m=='malformed': d['sources'][0].pop('source')
yaml.safe_dump(d,open(p,'w'),sort_keys=False)
PY
  expected=docs/index.html; [[ "$mutation" == extra ]] && expected=docs/unknown.md
  if output="$(validate_inventory "$tmp" 2>&1)"; then rm -f "$tmp"; echo "$mutation legacy source unexpectedly passed" >&2; return 1; fi
  grep -Fq "$expected" <<<"$output" || { rm -f "$tmp"; echo "$mutation missing diagnostic $expected: $output" >&2; return 1; }
  rm -f "$tmp"
  done
}
verify_legacy_useful_unit_inventory() { validate_inventory "$inventory"; }
verify_legacy_useful_unit_rejects_invalid_record() {
  local mutation tmp
  for mutation in missing-unit extra-unit unknown-id missing-locator missing-rationale duplicate-id rewrite-no-target decompose-one-target retire-with-target; do
    tmp="$(mktemp)"; cp "$inventory" "$tmp"
    python3 - "$tmp" "$mutation" <<'PY'
import sys,yaml
p,m=sys.argv[1:]; d=yaml.safe_load(open(p)); u=d['sources'][0]['useful_units'][0]
if m=='missing-unit': d['sources'][0]['useful_units']=d['sources'][0]['useful_units'][1:]
elif m=='extra-unit': d['sources'][0]['useful_units'].append(dict(u,unit_id='HOME-005'))
elif m=='unknown-id': u['unit_id']='UNKNOWN-001'
elif m=='missing-locator': u['source_locator']=''
elif m=='missing-rationale': u['rationale']=''
elif m=='duplicate-id': d['sources'][0]['useful_units'][1]['unit_id']=u['unit_id']
elif m=='rewrite-no-target': u['target_routes']=[]
elif m=='decompose-one-target': u['disposition']='decompose'; u['target_routes']=['/']
elif m=='retire-with-target': u['disposition']='retire'; u['target_routes']=['/']
yaml.safe_dump(d,open(p,'w'),sort_keys=False)
PY
    if output="$(validate_inventory "$tmp" 2>&1)"; then rm -f "$tmp"; echo "$mutation invalid unit unexpectedly passed" >&2; return 1; fi
    expected=HOME-001; [[ "$mutation" == extra-unit ]] && expected=HOME-005; [[ "$mutation" == unknown-id ]] && expected=UNKNOWN-001
    grep -Fq "$expected" <<<"$output" || { rm -f "$tmp"; echo "$mutation missing $expected diagnostic: $output" >&2; return 1; }
    rm -f "$tmp"
  done
}

verify_legacy_content_disposition_inventory
verify_legacy_content_disposition_rejects_invalid_entry
verify_legacy_useful_unit_inventory
verify_legacy_useful_unit_rejects_invalid_record
