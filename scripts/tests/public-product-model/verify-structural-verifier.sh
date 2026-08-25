#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
verifier="${root}/scripts/verify-public-product-model.sh"
run_verifier() { BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 BACKSTOP_PUBLIC_MODEL_ROOT="$1" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$root" "$verifier" "${2:-full}"; }
copy_complete_fixture() {
  local target="$1"
  cp -R "$root/docs" "$target/docs"
  cp -R "$root/artifacts" "$target/artifacts"
  cp -R "$root/pkg" "$target/pkg"
  cp -R "$root/cmd" "$target/cmd"
  cp -R "$root/bundles" "$target/bundles"
  cp -R "$root/issues" "$target/issues"
  cp "$root/README.md" "$root/backstop.yml" "$root/backstop.lock" "$target/"
}
make_claim_fixture() {
  local target="$1"
  copy_complete_fixture "$target"
  cat >"$target/docs/status.md" <<'EOF'
## Adjacent guidance {#adjacent-guidance}

<!-- backstop-claim: CLAIM-005 -->
Backstop stops at an inspectable verdict because external orchestration and organizational enforcement have different owners.

<!-- backstop-journey-link: JLINK-024 -->
[Continue outside Backstop](/contributing/#external-ownership)

That continuation is guidance, not a guarantee provided by Backstop.
<!-- /backstop-claim -->
EOF
}
verify_final_copy_structural_completeness(){ run_verifier "$root"; }
verify_declared_bash_function_contract(){
  bash -c 'source "$1"; declare -F verify_public_product_model >/dev/null; BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 verify_public_product_model --registries-only' _ "$verifier"
}
verify_seed_one_has_no_generalized_prose_system(){
  ! find "$root" -type f \( -iname '*prose*lsp*' -o -iname '*writing-style*' -o -iname '*prose-quality*' \) -not -path '*/.git/*' | grep -q .
}
verify_markdown_claim_region_bijection(){ run_verifier "$root"; }
verify_markdown_claim_region_rejects_invalid_linkage(){
  expect_full_mutation_failure missing-claim-region docs/status.md
  expect_full_mutation_failure wrong-claim-bytes CLAIM-001
  expect_full_mutation_failure duplicate-claim CLAIM-001
  expect_full_mutation_failure nested-claim CLAIM-001
  expect_full_mutation_failure overlapping-claim-regions CLAIM-001
  expect_full_mutation_failure unknown-claim-region CLAIM-999
  expect_full_mutation_failure orphaned-inventory-claim CLAIM-001
  expect_full_mutation_failure noncanonical-claim-placement CLAIM-001
}
expect_full_mutation_failure() {
  local mutation="$1" expected="$2" tmp output
  tmp="$(mktemp -d)"; copy_complete_fixture "$tmp"
  python3 - "$tmp" "$mutation" <<'PY'
import os,sys,yaml
r,m=sys.argv[1:]
def y(rel):
 p=os.path.join(r,rel); d=yaml.safe_load(open(p)); return p,d
def w(p,d): yaml.safe_dump(d,open(p,'w'),sort_keys=False)
if m=='delete-claim-evidence':
 p,d=y('docs/_data/evidence-inventory.yml'); d['claims'][0]['evidence_refs']=[]; w(p,d)
elif m=='blank-unit-summary':
 p,d=y('docs/_data/content-inventory.yml'); d['sources'][0]['useful_units'][0]['summary']=''; w(p,d)
elif m=='blank-unit-topic':
 p,d=y('docs/_data/content-inventory.yml'); d['sources'][0]['useful_units'][1]['topic']=''; w(p,d)
elif m=='missing-legacy-source':
 p,d=y('docs/_data/content-inventory.yml'); d['sources']=d['sources'][1:]; w(p,d)
elif m=='missing-useful-unit':
 p,d=y('docs/_data/content-inventory.yml'); d['sources'][0]['useful_units']=d['sources'][0]['useful_units'][1:]; w(p,d)
elif m=='configure-argv':
 p,d=y('docs/_data/content-topology.yml'); d['adoption_instructions'][1]['argv']=['gate']; w(p,d)
elif m=='missing-adoption-record':
 p,d=y('docs/_data/content-topology.yml'); d['adoption_instructions']=d['adoption_instructions'][1:]; w(p,d)
elif m=='adoption-command-missing':
 p=os.path.join(r,'docs/adopt.md'); s=open(p).read().replace('GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0','GOBIN omitted',1); open(p,'w').write(s)
elif m=='consistent-jlink-label':
 p,d=y('docs/_data/content-topology.yml'); old=d['journey_links'][0]['label']; d['journey_links'][0]['label']='Mutated label'; w(p,d)
 page=os.path.join(r,'docs/index.md'); s=open(page).read().replace('['+old+']','[Mutated label]'); open(page,'w').write(s)
elif m=='empty-arch': open(os.path.join(r,'docs/_diagrams/ARCH-001-delivery-lifecycle.mmd'),'w').write('')
elif m=='bad-arch-owner':
 p,d=y('docs/_data/product-model.yml'); d['architecture_views'][0]['owner']['anchor']='wrong'; w(p,d)
elif m=='unassigned-neighborhood':
 p,d=y('docs/_data/content-topology.yml'); d['pages'][0]['neighborhood_ids']=[]; w(p,d)
elif m=='multiply-assigned-neighborhood':
 p,d=y('docs/_data/content-topology.yml'); d['pages'][1]['neighborhood_ids'].append('NBR-001'); w(p,d)
elif m=='wrong-hero':
 p,d=y('docs/_data/content-topology.yml'); d['pages'][0]['hero_question']='Wrong hero'; w(p,d)
elif m=='bad-boundary-state':
 p,d=y('docs/_data/product-model.yml'); d['boundaries'][0]['state']='planned'; w(p,d)
elif m=='bad-boundary-fields':
 p,d=y('docs/_data/product-model.yml'); d['boundaries'][0]['continuation']={'journey_link_id':'JLINK-001'}; w(p,d)
elif m=='bad-boundary-claim-link':
 p,d=y('docs/_data/product-model.yml'); d['boundaries'][0]['claim_id']='CLAIM-999'; w(p,d)
elif m=='bad-claim-type':
 p,d=y('docs/_data/evidence-inventory.yml'); d['claims'][0]['claim_type']='unknown'; w(p,d)
elif m=='bad-evidence-kind':
 p,d=y('docs/_data/evidence-inventory.yml'); d['claims'][0]['evidence_refs'][0]['kind']='unknown'; w(p,d)
elif m=='nonexistent-commit':
 p,d=y('docs/_data/evidence-inventory.yml'); d['claims'][7]['evidence_refs'][0]['locator']['commit']='ffffffffffffffffffffffffffffffffffffffff'; w(p,d)
elif m=='wrong-role-kind':
 p,d=y('docs/_data/evidence-inventory.yml'); d['corpus_roles']['architecture_view']='EVIDENCE-009'; w(p,d)
elif m=='missing-page': os.remove(os.path.join(r,'docs/index.md'))
elif m=='duplicate-claim':
 p=os.path.join(r,'docs/status.md'); s=open(p).read(); block='<!-- backstop-claim: CLAIM-001 -->\nBackstop currently validates intent artifacts and runs installed pack engines as a blocking gate.\n<!-- /backstop-claim -->\n'; open(p,'w').write(s+block)
elif m=='missing-claim-region':
 p=os.path.join(r,'docs/status.md'); s=open(p).read(); s=s.replace('<!-- backstop-claim: CLAIM-001 -->\n','',1); open(p,'w').write(s)
elif m=='nested-claim':
 p=os.path.join(r,'docs/status.md'); s=open(p).read().replace('Backstop currently validates','<!-- backstop-claim: CLAIM-099 -->\nnested\n<!-- /backstop-claim -->\nBackstop currently validates',1); open(p,'w').write(s)
elif m=='overlapping-claim-regions':
 p=os.path.join(r,'docs/status.md'); s=open(p).read(); start='<!-- backstop-claim: CLAIM-001 -->\n'; s=s.replace(start,start+'<!-- backstop-claim: CLAIM-002 -->\n',1); s=s.replace('<!-- /backstop-claim -->','<!-- /backstop-claim -->\n<!-- /backstop-claim -->',1); open(p,'w').write(s)
elif m=='unknown-claim-region':
 p=os.path.join(r,'docs/status.md'); s=open(p).read().replace('<!-- backstop-claim: CLAIM-001 -->','<!-- backstop-claim: CLAIM-999 -->',1); open(p,'w').write(s)
elif m=='orphaned-inventory-claim':
 p=os.path.join(r,'docs/status.md'); s=open(p).read(); block='<!-- backstop-claim: CLAIM-001 -->\nBackstop currently validates intent artifacts and runs installed pack engines as a blocking gate.\n<!-- /backstop-claim -->'; assert block in s; open(p,'w').write(s.replace(block,'',1))
elif m=='noncanonical-claim-placement':
 p=os.path.join(r,'docs/status.md'); s=open(p).read(); block='<!-- backstop-claim: CLAIM-001 -->\nBackstop currently validates intent artifacts and runs installed pack engines as a blocking gate.\n<!-- /backstop-claim -->'; assert block in s; s=s.replace(block,'',1); s=s.replace('## Boundary states {#boundary-states}','## Boundary states {#boundary-states}\n\n'+block,1); open(p,'w').write(s)
elif m=='wrong-claim-bytes':
 p=os.path.join(r,'docs/status.md'); s=open(p).read().replace('Backstop currently validates','Backstop supposedly validates',1); open(p,'w').write(s)
elif m=='wrong-link-edge':
 p=os.path.join(r,'docs/evaluate.md'); s=open(p).read().replace('[See the operating model](/model/#operating-model)','[See the operating model](/model/#wrong)'); open(p,'w').write(s)
elif m=='jlink-global-nav-substitution':
 p=os.path.join(r,'docs/index.md'); s=open(p).read(); block='<!-- backstop-journey-link: JLINK-001 -->\n[Evaluate the failure fit](/evaluate/#failure-fit)'; s=s.replace(block,'').replace('# Backstop\n','# Backstop\n\n'+block+'\n',1); open(p,'w').write(s)
elif m=='missing-heading':
 p=os.path.join(r,'docs/model.md'); s=open(p).read().replace('{#operating-model}','{#wrong-operating-model}',1); open(p,'w').write(s)
elif m=='stale-cayman':
 p=os.path.join(r,'docs/index.md'); open(p,'a').write('\nCayman theme\n')
elif m=='placeholder':
 p=os.path.join(r,'docs/index.md'); open(p,'a').write('\ndraft placeholder\n')
elif m=='unmarked-consequential':
 p=os.path.join(r,'docs/index.md'); open(p,'a').write('\nInstalled packs execute their declared engines and the lock guarantees those exact bytes.\n')
elif m=='extra-claim-blank-line':
 p=os.path.join(r,'docs/status.md'); s=open(p).read().replace('Backstop currently validates intent artifacts and runs installed pack engines as a blocking gate.\n<!-- /backstop-claim -->','Backstop currently validates intent artifacts and runs installed pack engines as a blocking gate.\n\n<!-- /backstop-claim -->',1); open(p,'w').write(s)
elif m.startswith('claim-005-') or m=='arbitrary-comment-byte-significant':
 p=os.path.join(r,'docs/status.md'); s=open(p).read(); marker='<!-- backstop-journey-link: JLINK-024 -->\n'; link='[Continue outside Backstop](/contributing/#external-ownership)'
 if m.endswith('before-explanation'): s=s.replace(marker,'').replace('Backstop stops',marker+'Backstop stops',1)
 elif m.endswith('outside-claim'): s=s.replace(marker,''); s=marker+s
 elif m.endswith('after-link'): s=s.replace(marker+link,link+'\n'+marker.rstrip())
 elif m.endswith('intervening-line'): s=s.replace(marker+link,marker+'intervening metadata\n'+link)
 else: s=s.replace('Backstop stops','<!-- arbitrary-comment -->\nBackstop stops',1)
 open(p,'w').write(s)
PY
  if output="$(run_verifier "$tmp" 2>&1)"; then rm -rf "$tmp"; echo "$mutation unexpectedly passed" >&2; return 1; fi
  if ! grep -Fq "$expected" <<<"$output"; then rm -rf "$tmp"; echo "$mutation missing diagnostic $expected: $output" >&2; return 1; fi
  rm -rf "$tmp"
}
verify_site_mutation_manifest(){
  local manifest="${root}/scripts/tests/public-product-model/fixtures/site-mutation-manifest.yml" mutation expected
  while IFS=$'\t' read -r mutation expected; do
    expect_full_mutation_failure "$mutation" "$expected"
  done < <(python3 - "$manifest" <<'PY'
import sys,yaml
d=yaml.safe_load(open(sys.argv[1])); assert d.get('mutations') and len(d['mutations'])>=19
for m in d['mutations']: print(m['id']+'\t'+m['expected_diagnostic'])
PY
  )
}
verify_adjacent_guidance_embedded_jlink_claim_layout(){
  local tmp; tmp="$(mktemp -d)"; make_claim_fixture "$tmp"
  BACKSTOP_PUBLIC_MODEL_ROOT="$tmp" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$root" "$verifier" --claim-005-layout-only
  rm -rf "$tmp"
}
expect_claim_layout_failure() {
  local mutation="$1" tmp; tmp="$(mktemp -d)"; make_claim_fixture "$tmp"
  python3 - "$tmp/docs/status.md" "$mutation" <<'PY'
import sys
p,m=sys.argv[1:]; s=open(p).read(); marker='<!-- backstop-journey-link: JLINK-024 -->\n'; link='[Continue outside Backstop](/contributing/#external-ownership)'
if m=='before': s=s.replace(marker,'').replace('Backstop stops',marker+'Backstop stops',1)
elif m=='outside': s=s.replace(marker,''); s=marker+s
elif m=='after': s=s.replace(marker+link,link+'\n'+marker.rstrip())
elif m=='intervening': s=s.replace(marker+link,marker+'intervening metadata\n'+link)
elif m=='comment': s=s.replace('Backstop stops','<!-- arbitrary-comment -->\nBackstop stops',1)
open(p,'w').write(s)
PY
  if BACKSTOP_PUBLIC_MODEL_ROOT="$tmp" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$root" "$verifier" --claim-005-layout-only >/dev/null 2>&1; then
    rm -rf "$tmp"; echo "$mutation invalid CLAIM-005 layout passed" >&2; return 1
  fi
  rm -rf "$tmp"
}
verify_markdown_claim_region_rejects_invalid_embedded_jlink(){
  expect_claim_layout_failure before; expect_claim_layout_failure outside
  expect_claim_layout_failure after; expect_claim_layout_failure intervening
  expect_claim_layout_failure comment
}
verify_journey_link_matrix_rejects_invalid_claim_embedding(){
  expect_claim_layout_failure after; expect_claim_layout_failure intervening
}
verify_final_copy_structural_completeness
verify_declared_bash_function_contract
verify_seed_one_has_no_generalized_prose_system
verify_markdown_claim_region_bijection
verify_markdown_claim_region_rejects_invalid_linkage
verify_adjacent_guidance_embedded_jlink_claim_layout
verify_markdown_claim_region_rejects_invalid_embedded_jlink
verify_journey_link_matrix_rejects_invalid_claim_embedding
verify_site_mutation_manifest
