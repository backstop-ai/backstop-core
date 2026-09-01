#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
topology="${root}/docs/_data/content-topology.yml"
validate() {
python3 - "$1" <<'PY'
import hashlib,sys,yaml
d=yaml.safe_load(open(sys.argv[1],encoding='utf-8'))
def exact_ids(records,key,expected,label):
 actual=[x.get(key) for x in records]
 if actual==expected: return
 missing=[x for x in expected if x not in actual]; extra=[x for x in actual if x not in expected]
 duplicate=[x for x in actual if actual.count(x)>1]
 if duplicate: raise AssertionError(f'{duplicate[0]}: duplicate {label} field {key}')
 if missing: raise AssertionError(f'{missing[0]}: missing {label} field {key}')
 if extra: raise AssertionError(f'{extra[0]}: unexpected {label} field {key}')
 raise AssertionError(f'{actual[0]}: {label} order field {key}')
paths=['/','/evaluate/','/model/','/adopt/','/use-cases/','/packs/','/extend/','/reference/','/status/','/contributing/']
sources=['docs/index.md','docs/evaluate.md','docs/model.md','docs/adopt.md','docs/use-cases.md','docs/packs.md','docs/extend.md','docs/reference.md','docs/status.md','docs/contributing.md']
heroes=['What failure does Backstop prevent?','Your agent already writes the code.','How it works','Start where it will pay you first.','Which problem-oriented adoption path applies?','Which maintained pack already owns this standard?','When should this concern become a pack?','What exact interface or behavior do I need?','What is supported, limited, planned, or intentionally outside Backstop?','How can I participate in Backstop and its ecosystem?']
pages=d.get('pages',[]); assert [(p.get('source'),p.get('canonical_path')) for p in pages]==list(zip(sources,paths)),'docs/index.md: page source/canonical_path inventory'
for p,hero in zip(pages,heroes): assert p.get('hero_question')==hero,f"{p.get('source')}: hero_question"
ns=d.get('neighborhoods',[]); exact_ids(ns,'neighborhood_id',[f'NBR-{i:03}' for i in range(1,13)],'neighborhood')
owners=['/','/evaluate/','/evaluate/','/status/','/model/','/model/','/use-cases/','/packs/','/extend/','/reference/','/status/','/contributing/']
for n,owner in zip(ns,owners): assert n.get('owner_route')==owner,f"{n.get('neighborhood_id')}: owner_route"
assigned={}
for page in pages:
 for nid in page.get('neighborhood_ids',[]):
  assert nid not in assigned,f'{nid}: multiply assigned'; assigned[nid]=page['canonical_path']
for n in ns:
 nid=n['neighborhood_id']; assert nid in assigned,f'{nid}: unassigned'; assert assigned[nid]==n['owner_route'],f'{nid}: page/record owner mismatch'
assert set(assigned)=={n['neighborhood_id'] for n in ns},'unknown page neighborhood ID'
assert d.get('navigation')=={'primary':['/evaluate/','/model/','/adopt/','/use-cases/','/packs/','/extend/','/reference/'],'utility':['/status/','/contributing/']},'navigation contract'
links=d.get('journey_links',[]); exact_ids(links,'link_id',[f'JLINK-{i:03}' for i in range(1,25)],'journey link')
edges=[('/','define-work','/evaluate/','failure-fit'),('/evaluate/','working-state','/model/','operating-model'),('/use-cases/','choose-use-case','/evaluate/','fit-decision'),('/evaluate/','fit-decision','/adopt/','install'),('/model/','product-category','/status/','adjacent-guidance'),('/model/','gates-and-policy','/status/','supported-and-limited'),('/status/','boundary-states','/model/','ownership-boundaries'),('/model/','harness-integration','/reference/','compatibility'),('/reference/','compatibility','/status/','adjacent-guidance'),('/model/','operating-model','/reference/','artifact-schema-catalog'),('/model/','ownership-boundaries','/status/','project-boundaries'),('/adopt/','install','/reference/','configuration'),('/adopt/','verify-enforcement','/model/','enforcement-loop'),('/model/','enforcement-loop','/reference/','gate'),('/use-cases/','choose-use-case','/adopt/','adoption-paths'),('/use-cases/','pack-backed-use-cases','/packs/','choose-a-pack'),('/packs/','installed-pack-catalog','/reference/','pack-commands'),('/packs/','choose-a-pack','/status/','pack-direction'),('/extend/','pack-or-not','/reference/','pack-artifact'),('/extend/','author-a-pack','/contributing/','contribution-paths'),('/model/','provenance-and-verification','/reference/','source-traceability'),('/packs/','installed-pack-catalog','/reference/','cli-command-catalog'),('/reference/','cli-command-catalog','/status/','release-history'),('/status/','adjacent-guidance','/contributing/','external-ownership')]
labels=['Evaluate the failure fit','See the operating model','Check the fit decision','Install Backstop','Find adjacent guidance','Review support and limits','See ownership boundaries','Check compatibility details','Follow compatibility guidance','Inspect artifact schemas','Review project boundaries','Configure Backstop','Understand the enforcement loop','Read the gate reference','Choose an adoption path','Choose a maintained pack','Use pack commands','Review pack direction','Inspect the pack artifact','Contribute the pack','Trace the sources','Browse the CLI catalog','Check release history','Continue outside Backstop']
for x,edge,label in zip(links,edges,labels):
 assert (x.get('source_route'),x.get('source_anchor'),x.get('destination_route'),x.get('destination_anchor'))==edge,x['link_id']+' edge fields'
 assert x.get('label')==label,x['link_id']+' label'
for x in links:
 assert x.get('source_route') in paths and x.get('destination_route') in paths
 assert x.get('source_anchor') and x.get('destination_anchor') and x.get('label')
ins=d.get('adoption_instructions',[]); exact_ids(ins,'instruction_id',['ADOPT-INSTALL','ADOPT-CONFIGURE','ADOPT-ENFORCE'],'adoption instruction')
commands=['GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0','backstop init','backstop gate']
argv=[['install','github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0'],['init'],['gate']]
outputs=[['executable-file:<disposable-root>/.backstop-bin/backstop'],['file:<disposable-root>/backstop.yml'],['verdict:exit-0']]
owners=[('/adopt/','install'),('/adopt/','configure'),('/adopt/','verify-enforcement')]
executables=['go','<disposable-root>/.backstop-bin/backstop','<disposable-root>/.backstop-bin/backstop']
environments=[{'GOBIN':'<disposable-root>/.backstop-bin'},{},{}]
provenance=[{'kind':'go-module-version','coordinate':'github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0'},*([{'kind':'instruction-output','instruction_id':'ADOPT-INSTALL','path':'<disposable-root>/.backstop-bin/backstop'}]*2)]
for x,cmd,av,out,owner,exe,env,prov in zip(ins,commands,argv,outputs,owners,executables,environments,provenance):
 assert x['command_text']==cmd and x['command_sha256']=='sha256:'+hashlib.sha256(cmd.encode()).hexdigest(),x['instruction_id']+' digest'
 assert x['working_directory']=='<disposable-root>',x['instruction_id']+' working_directory'
 assert x['expected_exit_code']==0,x['instruction_id']+' expected_exit_code'
 assert x['expected_outputs']==out,x['instruction_id']+' expected_outputs'
 assert x['executable']==exe,x['instruction_id']+' executable'
 assert x['argv']==av,x['instruction_id']+' argv'
 assert (x['owner_route'],x['owner_anchor'])==owner,x['instruction_id']+' owner fields'
 assert x['environment']==env,x['instruction_id']+' environment'
 assert x['provenance']==prov,x['instruction_id']+' provenance'
PY
}
reject_invalid() { local output; if output="$(validate "$1" 2>&1)"; then echo "invalid topology passed" >&2; return 1; fi; if [[ -n "${2:-}" ]] && ! grep -Fq "$2" <<<"$output"; then echo "invalid topology missing diagnostic $2: $output" >&2; return 1; fi; }
validate_full(){ BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 BACKSTOP_PUBLIC_MODEL_ROOT="$root" "$root/scripts/verify-public-product-model.sh"; }
mutate_reject(){
 local mutation="$1" tmp; tmp="$(mktemp)"; cp "$topology" "$tmp"
 python3 - "$tmp" "$mutation" <<'PY'
import sys,yaml
p,m=sys.argv[1:]; d=yaml.safe_load(open(p))
if m=='neighborhood-owner': d['neighborhoods'][0]['owner_route']='/evaluate/'
elif m=='neighborhood-unassigned': d['pages'][0]['neighborhood_ids']=[]
elif m=='neighborhood-multiple': d['pages'][1]['neighborhood_ids'].append('NBR-001')
elif m=='neighborhood-page-owner': d['pages'][0]['neighborhood_ids']=['NBR-002']; d['pages'][1]['neighborhood_ids']=['NBR-001','NBR-003']
elif m=='hero-missing': d['pages'][0].pop('hero_question')
elif m=='hero-duplicate': d['pages'][1]['hero_question']=d['pages'][0]['hero_question']
elif m=='hero-presentation-owned': d['pages'][0]['hero_question']='{{ page.hero_question }}'
elif m=='link-edge': d['journey_links'][0]['destination_anchor']='wrong'
elif m=='link-label': d['journey_links'][0]['label']='Changed'
elif m=='link-order': d['journey_links'][0],d['journey_links'][1]=d['journey_links'][1],d['journey_links'][0]
elif m=='link-missing': d['journey_links']=d['journey_links'][1:]
elif m=='link-additional': d['journey_links'].append(dict(d['journey_links'][0],link_id='JLINK-025'))
elif m=='link-duplicate': d['journey_links'][1]['link_id']='JLINK-001'
elif m=='link-nonroot': d['journey_links'][0]['destination_route']='evaluate/'
elif m=='link-source': d['journey_links'][0]['source_anchor']='choose-your-path'
elif m=='instruction-command': d['adoption_instructions'][1]['command_text']='backstop gate'
elif m=='instruction-digest': d['adoption_instructions'][1]['command_sha256']='sha256:0'
elif m=='instruction-argv': d['adoption_instructions'][1]['argv']=['gate']
elif m=='instruction-shell': d['adoption_instructions'][1]['executable']='sh'
elif m=='instruction-output': d['adoption_instructions'][1]['expected_outputs']=[]
elif m=='instruction-owner': d['adoption_instructions'][1]['owner_anchor']='install'
elif m=='instruction-environment': d['adoption_instructions'][1]['environment']={'PATH':'mutable'}
elif m=='instruction-directory': d['adoption_instructions'][1]['working_directory']='.'
elif m=='instruction-provenance': d['adoption_instructions'][0]['provenance']['coordinate']='github.com/backstop-ai/backstop-core/cmd/backstop@latest'
elif m=='instruction-missing': d['adoption_instructions']=d['adoption_instructions'][1:]
elif m=='instruction-additional': d['adoption_instructions'].append(dict(d['adoption_instructions'][0],instruction_id='ADOPT-EXTRA'))
elif m=='instruction-order': d['adoption_instructions'][0],d['adoption_instructions'][1]=d['adoption_instructions'][1],d['adoption_instructions'][0]
elif m=='instruction-duplicate-id': d['adoption_instructions'][1]['instruction_id']='ADOPT-INSTALL'
yaml.safe_dump(d,open(p,'w'),sort_keys=False)
PY
 reject_invalid "$tmp" "$2"; rm -f "$tmp"
}
full_link_reject(){
 local mutation="$1" expected="$2" tmp output; tmp="$(mktemp -d)"
 for d in docs artifacts pkg cmd bundles issues; do cp -R "$root/$d" "$tmp/$d"; done
 cp "$root/README.md" "$root/backstop.yml" "$root/backstop.lock" "$tmp/"
 python3 - "$tmp/docs/index.md" "$mutation" <<'PY'
import sys
p,m=sys.argv[1:]; s=open(p).read(); block='<!-- backstop-journey-link: JLINK-001 -->\n[Evaluate the failure fit](/evaluate/#failure-fit)'
if m=='unmarked': s=s.replace('<!-- backstop-journey-link: JLINK-001 -->\n','',1)
elif m=='multiply-marked': s=s.replace(block,block+'\n\n'+block,1)
elif m=='global-navigation': s=s.replace(block,'',1).replace('# Backstop\n','# Backstop\n\n'+block+'\n',1)
open(p,'w').write(s)
PY
 if output="$(BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 BACKSTOP_PUBLIC_MODEL_ROOT="$tmp" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$root" "$root/scripts/verify-public-product-model.sh" 2>&1)"; then rm -rf "$tmp"; echo "$mutation journey link passed" >&2; return 1; fi
 if ! grep -Fq "$expected" <<<"$output"; then rm -rf "$tmp"; echo "$mutation missing $expected: $output" >&2; return 1; fi
 rm -rf "$tmp"
}
verify_public_product_model_complete(){ validate_full; }
verify_public_product_model_rejects_narrow_completion(){ reject_invalid "${root}/scripts/tests/public-product-model/fixtures/content-topology-invalid.yml" docs/index.md; }
verify_content_topology_exact_inventory(){ validate "$topology"; }
verify_content_topology_rejects_invalid_neighborhood_ownership(){ mutate_reject neighborhood-owner NBR-001; mutate_reject neighborhood-unassigned NBR-001; mutate_reject neighborhood-multiple NBR-001; mutate_reject neighborhood-page-owner NBR-001; }
verify_content_topology_navigation_contract(){ validate "$topology"; }
verify_content_topology_hero_question_matrix(){ validate "$topology"; }
verify_content_topology_rejects_invalid_hero_question(){ mutate_reject hero-missing docs/index.md; mutate_reject hero-duplicate docs/evaluate.md; mutate_reject hero-presentation-owned docs/index.md; }
verify_journey_link_matrix(){ validate "$topology"; validate_full; }
verify_journey_link_matrix_rejects_invalid_edge(){ mutate_reject link-edge JLINK-001; mutate_reject link-label JLINK-001; mutate_reject link-order JLINK-002; mutate_reject link-missing JLINK-001; mutate_reject link-additional JLINK-025; mutate_reject link-duplicate JLINK-001; mutate_reject link-nonroot JLINK-001; mutate_reject link-source JLINK-001; full_link_reject unmarked JLINK-001; full_link_reject multiply-marked JLINK-001; full_link_reject global-navigation JLINK-001; }
verify_adoption_instruction_matrix(){ validate "$topology"; }
verify_adoption_instruction_matrix_rejects_invalid_record(){ mutate_reject instruction-missing ADOPT-INSTALL; mutate_reject instruction-additional ADOPT-EXTRA; mutate_reject instruction-order ADOPT-CONFIGURE; mutate_reject instruction-duplicate-id ADOPT-INSTALL; mutate_reject instruction-command ADOPT-CONFIGURE; mutate_reject instruction-digest ADOPT-CONFIGURE; mutate_reject instruction-argv ADOPT-CONFIGURE; mutate_reject instruction-shell ADOPT-CONFIGURE; mutate_reject instruction-output ADOPT-CONFIGURE; mutate_reject instruction-owner ADOPT-CONFIGURE; mutate_reject instruction-environment ADOPT-CONFIGURE; mutate_reject instruction-directory ADOPT-CONFIGURE; mutate_reject instruction-provenance ADOPT-INSTALL; }
verify_adoption_instruction_source_and_digest_binding(){ validate "$topology"; validate_full; }
verify_public_product_model_complete; verify_public_product_model_rejects_narrow_completion
verify_content_topology_exact_inventory; verify_content_topology_rejects_invalid_neighborhood_ownership
verify_content_topology_navigation_contract; verify_content_topology_hero_question_matrix
verify_content_topology_rejects_invalid_hero_question; verify_journey_link_matrix
verify_journey_link_matrix_rejects_invalid_edge; verify_adoption_instruction_matrix
verify_adoption_instruction_matrix_rejects_invalid_record; verify_adoption_instruction_source_and_digest_binding
