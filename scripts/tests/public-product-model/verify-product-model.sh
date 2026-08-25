#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"; model="${root}/docs/_data/product-model.yml"
validate(){ python3 - "$1" "$root" "${2:-$root/docs/_data/evidence-inventory.yml}" <<'PY'
import os,sys,yaml
d=yaml.safe_load(open(sys.argv[1])); root=sys.argv[2]
evidence=yaml.safe_load(open(sys.argv[3])); claimmap={x['claim_id']:x for x in evidence['claims']}
territories={'product-category','intent-artifacts','work-tracks','plans','standards-packs','recipes','gates','baselines-ratchets','waivers','capabilities-journeys','provenance-verification','harness-integration','ownership-trust-boundaries'}
cs=d.get('concepts',[]); assert {x.get('territory') for x in cs}==territories,'concept territories'
assert len({x['concept_id'] for x in cs})==len(cs)
for x in cs:
 assert x.get('name') and x.get('definition') and x.get('source_refs') and x.get('related_concept_ids') is not None
 assert isinstance(x.get('owner'),dict) and set(x['owner'])=={'route','anchor'} and x['owner']['route'] and x['owner']['anchor'],x['concept_id']+' owner fields'
 for ref in x['source_refs']: assert os.path.exists(os.path.join(root,ref)),f"{x['concept_id']} missing {ref}"
views=d.get('architecture_views',[]); viewids=[x.get('architecture_id') for x in views]; expected_views=['ARCH-001','ARCH-002','ARCH-003']
if viewids!=expected_views:
 duplicate=next((x for x in viewids if viewids.count(x)>1),None); missing=next((x for x in expected_views if x not in viewids),None)
 raise AssertionError(str(duplicate or missing or viewids[0])+': architecture_id inventory')
paths=['docs/_diagrams/ARCH-001-delivery-lifecycle.mmd','docs/_diagrams/ARCH-002-enforcement-loop.mmd','docs/_diagrams/ARCH-003-ownership-boundaries.mmd']
anchors=['delivery-lifecycle','enforcement-loop','ownership-boundaries']
for x,p,a in zip(views,paths,anchors):
 assert x['diagram_source']==p and x['owner']=={'route':'/model/','anchor':a} and os.path.isfile(os.path.join(root,p)),x['architecture_id']+' source/owner fields'
 assert open(os.path.join(root,p),encoding='utf-8').read().strip(),x['architecture_id']+' empty Mermaid'
 for ref in x['source_refs']: assert os.path.exists(os.path.join(root,ref)),f"{x['architecture_id']} missing {ref}"
bs=d.get('boundaries',[]); assert [x.get('boundary_id') for x in bs]==[f'BOUNDARY-{i:03}' for i in range(1,6)]
for b,state in zip(bs,['supported','limitation','planned','non-goal','adjacent-guidance']): assert b.get('state')==state,b.get('boundary_id','boundary')+' state'
assert len({x['claim_id'] for x in bs})==len(bs),'BOUNDARY-001: duplicate claim_id linkage'
for b in bs:
 assert set(b)=={'boundary_id','statement','state','owner','source_refs','visitor_implication','claim_id','explanation_markdown','continuation','guarantee_denial_markdown'},b['boundary_id']+' field cardinality'
 assert b.get('explanation_markdown') and b.get('source_refs') and b.get('visitor_implication'),b['boundary_id']+' explanation/source_refs/visitor_implication fields'
 pair=b['boundary_id']+'/'+str(b.get('claim_id'))
 assert b['claim_id'] in claimmap and claimmap[b['claim_id']].get('boundary_id')==b['boundary_id'],pair+' claim linkage'
 assert claimmap[b['claim_id']].get('claim_type') in {'mechanism','runtime-behavior','compatibility','observed-failure','observed-outcome'},pair+' claim_type'
 assert claimmap[b['claim_id']].get('evidence_refs') and all(x.get('evidence_id') and x.get('kind') and x.get('locator') for x in claimmap[b['claim_id']]['evidence_refs']),pair+' incomplete evidence'
 for ref in b['source_refs']: assert os.path.exists(os.path.join(root,ref)),f"{b['boundary_id']} missing {ref}"
 if b['state']=='adjacent-guidance':
  assert b.get('continuation')=={'journey_link_id':'JLINK-024','route':'/contributing/','anchor':'external-ownership','label':'Continue outside Backstop'} and b.get('guarantee_denial_markdown'),pair+' adjacent fields route/anchor/label/composition'
 else: assert b.get('continuation') is None and b.get('guarantee_denial_markdown') is None,b['boundary_id']+' null structured fields'
PY
}
reject(){ local output; if output="$(validate "$1" "${3:-$root/docs/_data/evidence-inventory.yml}" 2>&1)"; then echo invalid-model-passed >&2; return 1; fi; if [[ -n "${2:-}" ]] && ! grep -Fq "$2" <<<"$output"; then echo "invalid model missing diagnostic $2: $output" >&2; return 1; fi; }
validate_full(){ BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 BACKSTOP_PUBLIC_MODEL_ROOT="$root" "$root/scripts/verify-public-product-model.sh"; }
mutate_reject(){
 local mutation="$1" tmp evtmp; tmp="$(mktemp)"; evtmp="$(mktemp)"; cp "$model" "$tmp"; cp "$root/docs/_data/evidence-inventory.yml" "$evtmp"
 python3 - "$tmp" "$evtmp" "$mutation" <<'PY'
import sys,yaml
p,ep,m=sys.argv[1:]; d=yaml.safe_load(open(p)); e=yaml.safe_load(open(ep))
if m=='duplicate-owner': d['concepts'][0]['owner']['secondary_route']='/evaluate/'
elif m=='unknown-state': d['boundaries'][0]['state']='maybe'
elif m=='missing-state': d['boundaries'][0].pop('state')
elif m=='multiple-state': d['boundaries'][0]['states']=['supported','planned']
elif m=='contradict-state': d['boundaries'][0]['state']='planned'
elif m=='missing-arch': d['architecture_views']=d['architecture_views'][1:]
elif m=='duplicate-arch': d['architecture_views'][1]['architecture_id']='ARCH-001'
elif m=='bad-arch-owner': d['architecture_views'][0]['owner']['anchor']='wrong'
elif m=='bad-arch-source': d['architecture_views'][0]['diagram_source']='docs/_diagrams/missing.mmd'
elif m=='duplicate-claim': d['boundaries'][1]['claim_id']=d['boundaries'][0]['claim_id']
elif m=='missing-claim': d['boundaries'][0]['claim_id']=''
elif m=='unknown-claim': d['boundaries'][0]['claim_id']='CLAIM-999'
elif m=='invalid-fields': d['boundaries'][0]['continuation']={'journey_link_id':'JLINK-001'}
elif m=='missing-explanation': d['boundaries'][0]['explanation_markdown']=''
elif m=='missing-continuation': d['boundaries'][-1]['continuation']=None
elif m=='missing-denial': d['boundaries'][-1]['guarantee_denial_markdown']=''
elif m=='bad-adjacent': d['boundaries'][-1]['continuation']['journey_link_id']='JLINK-001'
elif m=='bad-route': d['boundaries'][-1]['continuation']['route']='/status/'
elif m=='bad-anchor': d['boundaries'][-1]['continuation']['anchor']='wrong'
elif m=='bad-label': d['boundaries'][-1]['continuation']['label']='Wrong'
elif m=='bad-composition': d['boundaries'][-1]['continuation']['journey_link_id']='JLINK-023'
elif m=='invalid-boundary-claim-type': next(x for x in e['claims'] if x['claim_id']=='CLAIM-005')['claim_type']='unknown'
elif m=='incomplete-boundary-evidence': next(x for x in e['claims'] if x['claim_id']=='CLAIM-005')['evidence_refs']=[]
yaml.safe_dump(d,open(p,'w'),sort_keys=False)
yaml.safe_dump(e,open(ep,'w'),sort_keys=False)
PY
	 reject "$tmp" "$2" "$evtmp"; rm -f "$tmp" "$evtmp"
}
verify_canonical_product_model_ownership(){ validate "$model"; }
verify_canonical_product_model_has_no_parallel_truth(){ ! find "$root/docs" -type f \( -iname '*agent*ia*' -o -iname '*mcp*' -o -iname '*product-model*' ! -path "$model" \) | grep -q .; }
verify_canonical_product_model_rejects_duplicate_owner(){ reject "${root}/scripts/tests/public-product-model/fixtures/product-model-invalid.yml" concept; mutate_reject duplicate-owner CONCEPT-001; }
verify_product_boundary_state_matrix_positive(){ validate "$model"; }; verify_product_boundary_state_matrix_negative(){ mutate_reject missing-state BOUNDARY-001; mutate_reject multiple-state BOUNDARY-001; mutate_reject unknown-state BOUNDARY-001; mutate_reject contradict-state BOUNDARY-001; }
verify_adjacent_guidance_contract(){ validate "$model"; validate_full; mutate_reject bad-adjacent BOUNDARY-005; }; verify_required_architecture_view_inventory(){ validate "$model"; validate_full; }; verify_required_architecture_view_rejects_missing_or_invalid_view(){ mutate_reject missing-arch ARCH-001; mutate_reject duplicate-arch ARCH-001; mutate_reject bad-arch-owner ARCH-001; mutate_reject bad-arch-source ARCH-001; }
verify_product_boundary_claim_bijection(){ validate "$model"; validate_full; }; verify_product_boundary_rejects_invalid_claim_linkage(){ mutate_reject missing-claim BOUNDARY-001; mutate_reject duplicate-claim BOUNDARY-001; mutate_reject unknown-claim BOUNDARY-001; mutate_reject invalid-boundary-claim-type BOUNDARY-005/CLAIM-005; mutate_reject incomplete-boundary-evidence BOUNDARY-005/CLAIM-005; }
verify_product_boundary_structured_field_matrix(){ validate "$model"; validate_full; }; verify_product_boundary_rejects_invalid_structured_fields(){ mutate_reject invalid-fields BOUNDARY-001; mutate_reject missing-explanation BOUNDARY-001; mutate_reject missing-continuation BOUNDARY-005; mutate_reject missing-denial BOUNDARY-005; mutate_reject bad-adjacent BOUNDARY-005/CLAIM-005; mutate_reject bad-route BOUNDARY-005/CLAIM-005; mutate_reject bad-anchor BOUNDARY-005/CLAIM-005; mutate_reject bad-label BOUNDARY-005/CLAIM-005; mutate_reject bad-composition BOUNDARY-005/CLAIM-005; }
verify_canonical_product_model_ownership; verify_canonical_product_model_rejects_duplicate_owner; verify_canonical_product_model_has_no_parallel_truth
verify_product_boundary_state_matrix_positive; verify_product_boundary_state_matrix_negative; verify_adjacent_guidance_contract
verify_required_architecture_view_inventory; verify_required_architecture_view_rejects_missing_or_invalid_view
verify_product_boundary_claim_bijection; verify_product_boundary_rejects_invalid_claim_linkage
verify_product_boundary_structured_field_matrix; verify_product_boundary_rejects_invalid_structured_fields
