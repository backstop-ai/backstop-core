#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"; evidence="${root}/docs/_data/evidence-inventory.yml"
validate(){ python3 - "$1" "$root" <<'PY'
import os,re,subprocess,sys,yaml
d=yaml.safe_load(open(sys.argv[1])); root=sys.argv[2]
types={'mechanism','runtime-behavior','compatibility','observed-failure','observed-outcome'}; mechanism={'source','schema','test','implementation-commit'}
claims=d.get('claims',[]); assert claims,'claims inventory empty'; assert len({x.get('claim_id') for x in claims})==len(claims),'claim_id duplicate'
refs={}
for c in claims:
 for f in ('claim_id','statement','claim_type','owner','mechanism_summary','guarantee_boundary','known_limitations','adoption_implications','direction','statement_markdown','evidence_refs'): assert c.get(f) not in (None,'',[]),f"{c.get('claim_id')} missing {f}"
 assert c['claim_type'] in types,c['claim_id']+' claim_type'
 assert any(e.get('kind') in mechanism for e in c['evidence_refs']),c['claim_id']+' mechanism evidence'
 kinds={e.get('kind') for e in c['evidence_refs']}
 if c['claim_type'] in ('runtime-behavior','compatibility'): assert kinds & {'captured-execution','reproducible-execution'},c['claim_id']+' execution evidence'
 if c['claim_type']=='observed-failure': assert kinds & {'incident','example'},c['claim_id']+' incident/example evidence'
 if c['claim_type']=='observed-outcome': assert kinds & {'example','measurement'},c['claim_id']+' example/measurement evidence'
 if c['claim_type']=='compatibility':
  assert c.get('operability') and c.get('preserved_guarantees') and c.get('unpreserved_guarantees')
  assert c['operability'] != c['preserved_guarantees'], c['claim_id']+' guarantee equivalence'
 for e in c['evidence_refs']:
  eid=e.get('evidence_id'); assert eid and eid not in refs; refs[eid]=e
  assert e.get('kind') in {'source','schema','test','implementation-commit','captured-execution','reproducible-execution','incident','example','measurement'},f"{c['claim_id']} {eid} unknown kind"
  assert e.get('locator') and ('path' in e['locator'] or 'version' in e['locator'] or 'commit' in e['locator'])
  assert e.get('relevance'),f"{c['claim_id']} evidence relevance"
  if 'path' in e['locator']: assert os.path.exists(os.path.join(root,e['locator']['path'])),f"{c['claim_id']} missing {e['locator']['path']}"
  if 'version' in e['locator']: assert re.fullmatch(r'v\d+\.\d+\.\d+',str(e['locator']['version'])),f"{c['claim_id']} mutable version"
  if 'commit' in e['locator']:
   commit=str(e['locator']['commit']); assert re.fullmatch(r'[0-9a-f]{40}',commit),f"{c['claim_id']} {eid} mutable commit"
   assert subprocess.run(['git','-C',root,'cat-file','-e',commit+'^{commit}'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode==0,f"{c['claim_id']} {eid} nonexistent commit {commit}"
   if 'version' in e['locator']:
    tagged=subprocess.run(['git','-C',root,'rev-list','-n','1',str(e['locator']['version'])],text=True,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL)
    assert tagged.returncode==0 and tagged.stdout.strip()==commit,f"{c['claim_id']} {eid} version/commit provenance mismatch"
  if e['kind']=='reproducible-execution': assert e.get('command') and e.get('prerequisites')
  if e['kind'] in ('captured-execution','incident','example','measurement'): assert e.get('observation_name'),f"{c['claim_id']} {eid} unnamed observation"
  if e['kind'] in ('captured-execution','example'): assert e.get('artifact')
  if e['kind']=='example': assert e.get('provenance')
  if e['kind']=='incident': assert re.search(r'(?i)(issue|incident|report)',os.path.basename(e['locator'].get('path','')))
  if e['kind']=='measurement': assert all(e.get(x) for x in ('provenance','population','period','method')),f"{c['claim_id']} {eid} measurement fields"
assert 'currently interpreted' in next(c for c in claims if c['claim_id']=='CLAIM-007')['statement'],'CLAIM-007 evidence truth contract'
assert next(e for c in claims for e in c['evidence_refs'] if e['evidence_id']=='EVIDENCE-009')['locator']['path']=='pkg/recipe/substitute.go','EVIDENCE-009 evidence truth/provenance path'
assert next(e for c in claims for e in c['evidence_refs'] if e['evidence_id']=='EVIDENCE-012')['locator']['path']=='issues/ISSUE-166-contracts-pack-phase3-fixtures-fail-on-linux-ci.issue.md','EVIDENCE-012 evidence truth/provenance path'
assert next(e for c in claims for e in c['evidence_refs'] if e['evidence_id']=='EVIDENCE-014')['locator']['path']=='issues/ISSUE-097-unbound-selfpack-waivers-fail-open.issue.md','EVIDENCE-014 evidence truth/provenance path'
roles=d.get('corpus_roles',{}); expected={'failure_incident','failure_to_enforcement_example','captured_gate_result','source_or_commit_trace','architecture_view'}
missing_roles=expected-set(roles); assert not missing_roles,next(iter(missing_roles))+': missing corpus role'
assert set(roles)==expected,'unexpected corpus role'
assert len(set(roles.values()))==5,'corpus_roles: evidence IDs must be distinct'
rk={'failure_incident':{'incident'},'failure_to_enforcement_example':{'example'},'captured_gate_result':{'captured-execution'},'source_or_commit_trace':{'source','implementation-commit'},'architecture_view':{'source'}}
for role,eid in roles.items():
 assert eid in refs,f'{role} unknown evidence {eid}'; e=refs[eid]; assert e['kind'] in rk[role],f'{role} wrong kind {eid}'
 if role=='failure_to_enforcement_example': assert all(e.get(x) for x in ('artifact','provenance','before','after')),role+' qualification'
 if role=='architecture_view': assert e['locator'].get('path','').endswith('.mmd'),role+' qualification'
PY
}
reject(){ local output; if output="$(validate "$1" 2>&1)"; then echo invalid-evidence-passed >&2; return 1; fi; if [[ -n "${2:-}" ]] && ! grep -Fq "$2" <<<"$output"; then echo "invalid evidence missing diagnostic $2: $output" >&2; return 1; fi; }
validate_full(){ BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 BACKSTOP_PUBLIC_MODEL_ROOT="$root" "$root/scripts/verify-public-product-model.sh"; }
mutate_reject(){
 local mutation="$1" tmp; tmp="$(mktemp)"; cp "$evidence" "$tmp"
 python3 - "$tmp" "$mutation" <<'PY'
import sys,yaml
p,m=sys.argv[1:]; d=yaml.safe_load(open(p)); claims=d['claims']
if m=='incomplete': claims[0]['mechanism_summary']=''
elif m=='missing-claim-id': claims[0]['claim_id']=''
elif m=='unknown-claim-type': claims[0]['claim_type']='unknown'
elif m=='equivalence':
 c=next(x for x in claims if x['claim_type']=='compatibility'); c['preserved_guarantees']=c['operability']
elif m.startswith('remove-'):
 raw=m[7:]; mechanism_only=raw.startswith('mechanism-'); typ=raw[10:] if mechanism_only else raw; c=next(x for x in claims if x['claim_type']==typ)
 required={'source','schema','test','implementation-commit'} if mechanism_only else {'runtime-behavior':{'captured-execution','reproducible-execution'},'compatibility':{'captured-execution','reproducible-execution'},'observed-failure':{'incident','example'},'observed-outcome':{'example','measurement'}}[typ]
 c['evidence_refs']=[e for e in c['evidence_refs'] if e['kind'] not in required]
elif m=='missing-source': claims[0]['evidence_refs'][0]['locator']={'path':'does/not/exist'}
elif m=='mutable-version': claims[0]['evidence_refs'][1]['locator']={'version':'latest'}
elif m=='mutable-commit':
 c=next(x for x in claims if any('commit' in e.get('locator',{}) for e in x['evidence_refs'])); next(e for e in c['evidence_refs'] if 'commit' in e.get('locator',{}))['locator']['commit']='main'
elif m=='nonexistent-commit':
 c=next(x for x in claims if any('commit' in e.get('locator',{}) for e in x['evidence_refs'])); next(e for e in c['evidence_refs'] if 'commit' in e.get('locator',{}))['locator']['commit']='ffffffffffffffffffffffffffffffffffffffff'
elif m=='unnamed-observation':
 e=next(e for c in claims for e in c['evidence_refs'] if e.get('kind')=='incident'); e['observation_name']=''
elif m=='methodless-metric':
 e={'evidence_id':'EVIDENCE-METHODLESS','kind':'measurement','locator':{'path':'README.md'},'observation_name':'methodless metric','provenance':'repo','population':'one','period':'v0.2.0','relevance':'Purported metric'}; claims[-1]['evidence_refs'].append(e)
elif m=='irrelevant': claims[0]['evidence_refs'][0]['relevance']=''
elif m=='false-repair-claim': next(c for c in claims if c['claim_id']=='CLAIM-007')['statement']='The repaired enforcement now accepts literal placeholders.'
elif m=='tag-commit-mismatch': next(e for c in claims for e in c['evidence_refs'] if e['evidence_id']=='EVIDENCE-011')['locator']['version']='v0.2.0'
elif m=='future-before-after': next(e for c in claims for e in c['evidence_refs'] if e['evidence_id']=='EVIDENCE-012')['locator']['path']='issues/ISSUE-184-fixture-path-filter-diagnostics.issue.md'
elif m=='narrative-capture': next(e for c in claims for e in c['evidence_refs'] if e['evidence_id']=='EVIDENCE-014')['locator']['path']='issues/ISSUE-183-local-pack-relock-refreshes-stale-install.issue.md'
elif m=='missing-role': d['corpus_roles'].pop('architecture_view')
elif m=='invalid-role': d['corpus_roles']['architecture_view']='UNKNOWN-EVIDENCE'
elif m=='wrong-kind-role': d['corpus_roles']['architecture_view']='EVIDENCE-009'
yaml.safe_dump(d,open(p,'w'),sort_keys=False)
PY
	 reject "$tmp" "$2"; rm -f "$tmp"
}
verify_consequential_claim_contract(){ validate "$evidence"; validate_full; }; verify_consequential_claim_rejects_incomplete_record(){ reject "${root}/scripts/tests/public-product-model/fixtures/evidence-inventory-invalid.yml" claims; mutate_reject incomplete CLAIM-001; mutate_reject missing-claim-id 'missing claim_id'; mutate_reject unknown-claim-type 'CLAIM-001 claim_type'; }
verify_compatibility_claim_separates_operability_and_guarantee(){ validate "$evidence"; }; verify_compatibility_claim_rejects_guarantee_equivalence(){ mutate_reject equivalence CLAIM-006; }
verify_claim_type_evidence_matrix_positive(){ validate "$evidence"; }; verify_claim_type_evidence_matrix_negative(){
  mutate_reject remove-mechanism-mechanism CLAIM-002; mutate_reject remove-mechanism-runtime-behavior CLAIM-001; mutate_reject remove-mechanism-compatibility CLAIM-006; mutate_reject remove-mechanism-observed-failure CLAIM-007; mutate_reject remove-mechanism-observed-outcome CLAIM-008
  mutate_reject remove-runtime-behavior CLAIM-001; mutate_reject remove-compatibility CLAIM-006; mutate_reject remove-observed-failure CLAIM-007; mutate_reject remove-observed-outcome CLAIM-008
}
verify_evidence_sources_are_durable_and_resolvable(){ validate "$evidence"; mutate_reject missing-source CLAIM-001; mutate_reject mutable-version CLAIM-001; mutate_reject mutable-commit CLAIM-008; mutate_reject nonexistent-commit CLAIM-008; mutate_reject unnamed-observation CLAIM-007; mutate_reject methodless-metric EVIDENCE-METHODLESS; mutate_reject irrelevant CLAIM-001; mutate_reject false-repair-claim CLAIM-007; mutate_reject tag-commit-mismatch EVIDENCE-011; mutate_reject future-before-after EVIDENCE-012; mutate_reject narrative-capture EVIDENCE-014; }
verify_evidence_corpus_minimum_roles(){ validate "$evidence"; }
verify_evidence_corpus_rejects_missing_or_invalid_role(){ mutate_reject missing-role architecture_view; mutate_reject invalid-role architecture_view; mutate_reject wrong-kind-role architecture_view; }
verify_consequential_claim_contract; verify_consequential_claim_rejects_incomplete_record
verify_compatibility_claim_separates_operability_and_guarantee; verify_compatibility_claim_rejects_guarantee_equivalence
verify_claim_type_evidence_matrix_positive; verify_claim_type_evidence_matrix_negative
verify_evidence_sources_are_durable_and_resolvable; verify_evidence_corpus_minimum_roles; verify_evidence_corpus_rejects_missing_or_invalid_role
