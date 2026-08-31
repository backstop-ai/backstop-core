#!/usr/bin/env bash
set -euo pipefail

repo_root="${BACKSTOP_PUBLIC_MODEL_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

verify_public_product_model() {
local mode="${1:-full}"
local git_root="${BACKSTOP_PUBLIC_MODEL_GIT_ROOT:-$repo_root}"

python3 - "$repo_root" "$mode" "$git_root" <<'PY'
import hashlib,json,os,re,subprocess,sys,yaml
root,mode,git_root=sys.argv[1:]
def fail(msg): raise SystemExit('public-product-model: '+msg)
def load(rel):
 p=os.path.join(root,rel)
 if not os.path.isfile(p): fail('missing '+rel)
 try: return yaml.safe_load(open(p,encoding='utf-8'))
 except Exception as e: fail(f'{rel}: invalid YAML: {e}')
def exact_ids(records,key,expected,label):
 actual=[x.get(key) for x in records]
 if actual==expected: return
 missing=[x for x in expected if x not in actual]
 extra=[x for x in actual if x not in expected]
 duplicate=sorted({x for x in actual if actual.count(x)>1},key=str)
 if duplicate: fail(str(duplicate[0])+': duplicate '+label+' record field '+key)
 if missing: fail(str(missing[0])+': missing '+label+' record field '+key)
 if extra: fail(str(extra[0])+': unexpected '+label+' record field '+key)
 for index,(got,want) in enumerate(zip(actual,expected)):
  if got!=want: fail(str(got)+': '+label+' order field '+key+' expected '+str(want)+' at index '+str(index))
 fail(label+': '+key+' cardinality')

top=load('docs/_data/content-topology.yml'); model=load('docs/_data/product-model.yml'); evidence=load('docs/_data/evidence-inventory.yml'); inventory=load('docs/_data/content-inventory.yml'); presentation=load('docs/_data/site-presentation.yml')
parallel=[]
for base,_,files in os.walk(os.path.join(root,'docs')):
 for name in files:
  rel=os.path.relpath(os.path.join(base,name),root)
  low=rel.lower()
  if ('product-model' in low and rel!='docs/_data/product-model.yml') or 'agent-ia' in low or '/mcp' in low: parallel.append(rel)
if parallel: fail('parallel product truth or machine-only publication: '+','.join(parallel))
paths=['/','/evaluate/','/model/','/adopt/','/use-cases/','/packs/','/extend/','/reference/','/status/','/contributing/']
sources=['docs/index.md','docs/evaluate.md','docs/model.md','docs/adopt.md','docs/use-cases.md','docs/packs.md','docs/extend.md','docs/reference.md','docs/status.md','docs/contributing.md']
pages=top.get('pages',[])
if [(x.get('source'),x.get('canonical_path')) for x in pages]!=list(zip(sources,paths)): fail('page inventory must contain exact ten source/path pairs')
heroes=['What failure does Backstop prevent?','Your agent already writes the code.','How does it work?','What does a first working adoption require?','Which problem-oriented adoption path applies?','Which maintained pack already owns this standard?','When should this concern become a pack?','What exact interface or behavior do I need?','What is supported, limited, planned, or intentionally outside Backstop?','How can I participate in Backstop and its ecosystem?']
for page,hero in zip(pages,heroes):
 if page.get('hero_question')!=hero: fail(page.get('source','page')+': hero_question')
presentation_pages=presentation.get('pages',[])
if [(x.get('route'),x.get('hero_question')) for x in presentation_pages]!=list(zip(paths,heroes)): fail('site presentation must consume the exact Seed 1 hero matrix')
ns=top.get('neighborhoods',[])
exact_ids(ns,'neighborhood_id',[f'NBR-{i:03}' for i in range(1,13)],'neighborhood')
neighborhood_owners=['/','/evaluate/','/evaluate/','/status/','/model/','/model/','/use-cases/','/packs/','/extend/','/reference/','/status/','/contributing/']
for n,expected_owner in zip(ns,neighborhood_owners):
 nid=n.get('neighborhood_id','NBR-unknown')
 if n.get('owner_route')!=expected_owner: fail(nid+': neighborhood owner must be '+expected_owner)
page_by_route={p.get('canonical_path'):p for p in pages}
assigned={}
for page in pages:
 route=page.get('canonical_path','<missing-route>')
 ids=page.get('neighborhood_ids')
 if not isinstance(ids,list): fail(route+': neighborhood_ids must be a list')
 for nid in ids:
  if nid in assigned: fail(str(nid)+': multiply assigned to '+assigned[nid]+' and '+route)
  assigned[nid]=route
for n in ns:
 nid=n['neighborhood_id']; owner=n['owner_route']
 if nid not in assigned: fail(nid+': unassigned from page.neighborhood_ids')
 if assigned[nid]!=owner: fail(nid+': page.neighborhood_ids owner '+assigned[nid]+' does not match neighborhood owner '+owner)
if set(assigned)!={n['neighborhood_id'] for n in ns}: fail('unknown page neighborhood ID: '+','.join(sorted(set(assigned)-{n['neighborhood_id'] for n in ns})))
if top.get('navigation')!={'primary':['/evaluate/','/model/','/adopt/','/use-cases/','/packs/','/extend/','/reference/'],'utility':['/status/','/contributing/']}: fail('navigation membership/order')
links=top.get('journey_links',[])
exact_ids(links,'link_id',[f'JLINK-{i:03}' for i in range(1,25)],'journey link')
edge_specs=[
('/','define-work','/evaluate/','failure-fit'),('/evaluate/','working-state','/model/','operating-model'),('/use-cases/','choose-use-case','/evaluate/','fit-decision'),('/evaluate/','fit-decision','/adopt/','install'),('/model/','product-category','/status/','adjacent-guidance'),('/model/','gates-and-policy','/status/','supported-and-limited'),('/status/','boundary-states','/model/','ownership-boundaries'),('/model/','harness-integration','/reference/','compatibility'),('/reference/','compatibility','/status/','adjacent-guidance'),('/model/','operating-model','/reference/','artifact-schema-catalog'),('/model/','ownership-boundaries','/status/','project-boundaries'),('/adopt/','install','/reference/','configuration'),('/adopt/','verify-enforcement','/model/','enforcement-loop'),('/model/','enforcement-loop','/reference/','gate'),('/use-cases/','choose-use-case','/adopt/','adoption-paths'),('/use-cases/','pack-backed-use-cases','/packs/','choose-a-pack'),('/packs/','installed-pack-catalog','/reference/','pack-commands'),('/packs/','choose-a-pack','/status/','pack-direction'),('/extend/','pack-or-not','/reference/','pack-artifact'),('/extend/','author-a-pack','/contributing/','contribution-paths'),('/model/','provenance-and-verification','/reference/','source-traceability'),('/packs/','installed-pack-catalog','/reference/','cli-command-catalog'),('/reference/','cli-command-catalog','/status/','release-history'),('/status/','adjacent-guidance','/contributing/','external-ownership')]
actual_edges=[(x.get('source_route'),x.get('source_anchor'),x.get('destination_route'),x.get('destination_anchor')) for x in links]
if actual_edges!=edge_specs or any(not x.get('label') for x in links): fail('exact JLINK edge matrix')
link_labels=['Evaluate the failure fit','See the operating model','Check the fit decision','Install Backstop','Find adjacent guidance','Review support and limits','See ownership boundaries','Check compatibility details','Follow compatibility guidance','Inspect artifact schemas','Review project boundaries','Configure Backstop','Understand the enforcement loop','Read the gate reference','Choose an adoption path','Choose a maintained pack','Use pack commands','Review pack direction','Inspect the pack artifact','Contribute the pack','Trace the sources','Browse the CLI catalog','Check release history','Continue outside Backstop']
for link,label in zip(links,link_labels):
 if link.get('label')!=label: fail(link.get('link_id','JLINK')+': exact label')
instructions=top.get('adoption_instructions',[])
exact_ids(instructions,'instruction_id',['ADOPT-INSTALL','ADOPT-CONFIGURE','ADOPT-ENFORCE'],'adoption instruction')
commands=['GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0','backstop init','backstop gate']
owners=[('/adopt/','install'),('/adopt/','configure'),('/adopt/','verify-enforcement')]
for x in instructions:
 digest='sha256:'+hashlib.sha256(x.get('command_text','').encode()).hexdigest()
 if x.get('command_sha256')!=digest: fail(x.get('instruction_id','instruction')+': command_sha256')
 if x.get('executable') in ('sh','bash') or not x.get('expected_outputs'): fail(x.get('instruction_id','instruction')+': structured execution')
for idx,(x,cmd,owner) in enumerate(zip(instructions,commands,owners)):
 if x.get('command_text')!=cmd or (x.get('owner_route'),x.get('owner_anchor'))!=owner or x.get('working_directory')!='<disposable-root>' or x.get('expected_exit_code')!=0: fail(x['instruction_id']+': exact execution record')
 if idx==0:
  if x.get('executable')!='go' or x.get('argv')!=['install','github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0'] or x.get('environment')!={'GOBIN':'<disposable-root>/.backstop-bin'} or x.get('provenance')!={'kind':'go-module-version','coordinate':'github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0'}: fail(x['instruction_id']+': immutable install provenance')
 else:
  if x.get('executable')!='<disposable-root>/.backstop-bin/backstop' or x.get('environment')!={} or x.get('provenance')!={'kind':'instruction-output','instruction_id':'ADOPT-INSTALL','path':'<disposable-root>/.backstop-bin/backstop'}: fail(x['instruction_id']+': installed-output provenance')
expected_argv=[['install','github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0'],['init'],['gate']]
expected_outputs=[['executable-file:<disposable-root>/.backstop-bin/backstop'],['file:<disposable-root>/backstop.yml'],['verdict:exit-0']]
for x,argv,outputs in zip(instructions,expected_argv,expected_outputs):
 if x.get('argv')!=argv: fail(x['instruction_id']+': argv')
 if x.get('expected_outputs')!=outputs: fail(x['instruction_id']+': expected_outputs')
concepts=model.get('concepts',[])
if len(concepts)<13 or len({x.get('concept_id') for x in concepts})!=len(concepts): fail('canonical concept inventory')
territories=['product-category','intent-artifacts','work-tracks','plans','standards-packs','recipes','gates','baselines-ratchets','waivers','capabilities-journeys','provenance-verification','harness-integration','ownership-trust-boundaries']
if [x.get('territory') for x in concepts]!=territories: fail('exact canonical concept territories')
concept_anchors=['product-category','intent-artifacts','work-tracks','bounded-execution','standards-packs','recipes','gates-and-policy','baselines-and-ratchets','waivers','capabilities-and-journeys','provenance-and-verification','harness-integration','ownership-boundaries']
if [x.get('owner') for x in concepts]!=[{'route':'/model/','anchor':a} for a in concept_anchors]: fail('exact canonical concept owners')
def durable_path(ref,owner):
 if not isinstance(ref,str) or not ref or not os.path.exists(os.path.join(root,ref)): fail(owner+': missing durable source_ref '+str(ref))
for concept in concepts:
 cid=concept.get('concept_id','concept')
 if not concept.get('name') or not concept.get('definition') or not concept.get('source_refs') or concept.get('related_concept_ids') is None: fail(cid+': incomplete concept')
 if not isinstance(concept.get('owner'),dict) or set(concept['owner'])!={'route','anchor'}: fail(cid+': one exact owner')
 for ref in concept['source_refs']: durable_path(ref,cid)
views=model.get('architecture_views',[])
expected_view_ids=['ARCH-001','ARCH-002','ARCH-003']
exact_ids(views,'architecture_id',expected_view_ids,'architecture view')
view_paths=['docs/_diagrams/ARCH-001-delivery-lifecycle.mmd','docs/_diagrams/ARCH-002-enforcement-loop.mmd','docs/_diagrams/ARCH-003-ownership-boundaries.mmd']
view_anchors=['delivery-lifecycle','enforcement-loop','ownership-boundaries']
for view,path,anchor in zip(views,view_paths,view_anchors):
 aid=view.get('architecture_id','ARCH-unknown')
 if view.get('diagram_source')!=path: fail(aid+': diagram_source must be '+path)
 if view.get('owner')!={'route':'/model/','anchor':anchor}: fail(aid+': owner must be /model/#'+anchor)
for view in views:
 p=view.get('diagram_source','')
 if not p.endswith('.mmd') or not os.path.isfile(os.path.join(root,p)): fail(view.get('architecture_id','ARCH')+': diagram source')
 aid=view.get('architecture_id','ARCH')
 for ref in view.get('source_refs',[]): durable_path(ref,aid)
 content=open(os.path.join(root,p),encoding='utf-8').read()
 required={'ARCH-001':['Issue','Bundle','Spec','Plan','Validation'],'ARCH-002':['Intent','Agent','Engines','Verdict','Evidence','Provenance'],'ARCH-003':['Core','Packs','Runtime','Tools','Verdict']}[aid]
 if not content.strip() or not all(token in content for token in required): fail(aid+': empty or incomplete Mermaid contract')
boundaries=model.get('boundaries',[]); claims=evidence.get('claims',[]); cmap={x.get('claim_id'):x for x in claims}
exact_ids(boundaries,'boundary_id',[f'BOUNDARY-{i:03}' for i in range(1,6)],'boundary')
for b,state in zip(boundaries,['supported','limitation','planned','non-goal','adjacent-guidance']):
 if b.get('state')!=state: fail(b.get('boundary_id','boundary')+': state must be '+state)
if len({x.get('claim_id') for x in boundaries})!=len(boundaries): fail('boundary claim bijection has duplicate claim_id')
linkmap={x['link_id']:x for x in links}
for b in boundaries:
 bid=b.get('boundary_id','boundary'); c=cmap.get(b.get('claim_id'))
 if set(b)!={'boundary_id','statement','state','owner','source_refs','visitor_implication','claim_id','explanation_markdown','continuation','guarantee_denial_markdown'}: fail(bid+': exact boundary fields')
 if not c or c.get('boundary_id')!=bid: fail(bid+': claim linkage')
 if c.get('owner')!=b.get('owner'): fail(bid+': claim owner linkage')
 allowed={'supported':{'mechanism','runtime-behavior'},'limitation':{'mechanism','observed-failure'},'planned':{'mechanism'},'non-goal':{'mechanism'},'adjacent-guidance':{'mechanism'}}
 if c.get('claim_type') not in allowed[b['state']]: fail(bid+'/'+str(c.get('claim_id'))+': invalid claim type '+str(c.get('claim_type')))
 if not b.get('source_refs') or not b.get('visitor_implication'): fail(bid+': source_refs/visitor_implication')
 for ref in b['source_refs']:
  if not isinstance(ref,str) or not os.path.exists(os.path.join(root,ref)): fail(bid+': missing durable source_ref '+str(ref))
 exp=b.get('explanation_markdown')
 if not exp: fail(bid+': explanation_markdown')
 if b['state']=='adjacent-guidance':
  cont=b.get('continuation'); denial=b.get('guarantee_denial_markdown')
  if not cont or not denial: fail(bid+': continuation/guarantee_denial_markdown')
  link=linkmap.get(cont.get('journey_link_id'))
  if not link or (link['source_route'],link['source_anchor'])!=(b['owner']['route'],b['owner']['anchor']) or (link['destination_route'],link['destination_anchor'],link['label'])!=(cont['route'],cont['anchor'],cont['label']): fail(bid+': continuation JLINK mismatch')
  expected=exp+'\n\n['+cont['label']+']('+cont['route']+'#'+cont['anchor']+')\n\n'+denial
 else:
  if b.get('continuation') is not None or b.get('guarantee_denial_markdown') is not None: fail(bid+': non-null continuation or denial')
  expected=exp
 if c.get('statement_markdown')!=expected: fail(bid+': claim composition')
boundary_claims={b.get('claim_id'):b.get('boundary_id') for b in boundaries}
for c in claims:
 cid=c.get('claim_id','CLAIM-unknown'); bid=c.get('boundary_id')
 if bid is not None and boundary_claims.get(cid)!=bid: fail(cid+': boundary linkage '+str(bid))
legacy_sources=['docs/index.html','docs/getting-started.md','docs/concepts.md','docs/artifact-workflow.md','docs/pack-authoring.md','docs/cli-reference.md']
source_records=inventory.get('sources',[])
actual_legacy=[x.get('source') for x in source_records]
if actual_legacy!=legacy_sources:
 missing=sorted(set(legacy_sources)-set(actual_legacy)); extra=sorted(set(actual_legacy)-set(legacy_sources),key=str)
 fail('legacy source inventory missing='+','.join(missing)+' extra='+','.join(str(x) for x in extra))
units=[]
for src in source_records: units.extend(src.get('useful_units',[]))
expected_ids=[*[f'HOME-{i:03}' for i in range(1,5)],*[f'GET-{i:03}' for i in range(1,6)],*[f'CON-{i:03}' for i in range(1,7)],*[f'ART-{i:03}' for i in range(1,6)],*[f'PACK-{i:03}' for i in range(1,7)],*[f'CLI-{i:03}' for i in range(1,6)]]
actual_ids=[x.get('unit_id') for x in units]
if actual_ids!=expected_ids:
 missing=sorted(set(expected_ids)-set(actual_ids)); extra=sorted(set(actual_ids)-set(expected_ids),key=str)
 duplicates=sorted({x for x in actual_ids if actual_ids.count(x)>1},key=str)
 fail('useful-unit inventory missing='+','.join(missing)+' extra='+','.join(str(x) for x in extra)+' duplicate='+','.join(str(x) for x in duplicates))
expected_unit_contract={
'HOME-001':('Landing failure/category framing','rewrite',['/']),'HOME-002':('Define / enforce / drift model','decompose',['/','/model/']),'HOME-003':('Composable adoption modes','decompose',['/evaluate/','/use-cases/']),'HOME-004':('Adoption call to action','rewrite',['/adopt/']),
'GET-001':('Before you start','merge',['/adopt/']),'GET-002':('Project, install, configure, first run','merge',['/adopt/']),'GET-003':('Failure-to-green walkthrough','decompose',['/adopt/','/use-cases/']),'GET-004':('Troubleshooting','merge',['/reference/']),'GET-005':('What Backstop did not do','decompose',['/evaluate/','/status/']),
'CON-001':('Premise and trust thesis','decompose',['/evaluate/','/model/']),'CON-002':('Packs and zero bundled checks','decompose',['/model/','/packs/']),'CON-003':('Thin-executor distinction','merge',['/model/']),'CON-004':('Gate, dimensions, severity, policy','decompose',['/model/','/reference/']),'CON-005':('Baselines and waivers','merge',['/model/']),'CON-006':('Artifacts and system composition','merge',['/model/']),
'ART-001':('Two work tracks','merge',['/model/']),'ART-002':('CLI creation and ID reservation','decompose',['/adopt/','/reference/']),'ART-003':('Lifecycle states','decompose',['/model/','/reference/']),'ART-004':('Closure and traceability','decompose',['/model/','/reference/']),'ART-005':('Artifact validation and gate integration','merge',['/model/']),
'PACK-001':('Pack definition and selection boundary','decompose',['/packs/','/extend/']),'PACK-002':('Scaffold, manifest, and engine authoring','decompose',['/extend/','/reference/']),'PACK-003':('Rules, claims, fixtures, and tools','decompose',['/extend/','/reference/']),'PACK-004':('Path-filter sharp edge','decompose',['/extend/','/status/']),'PACK-005':('Check, test, findings, iteration','merge',['/extend/']),'PACK-006':('Publishing and ecosystem continuation','decompose',['/extend/','/contributing/']),
'CLI-001':('Conventions, exit codes, JSON, discovery','merge',['/reference/']),'CLI-002':('Init, doctor, and gate','decompose',['/adopt/','/reference/']),'CLI-003':('Pack commands','merge',['/reference/']),'CLI-004':('Artifact commands','merge',['/reference/']),'CLI-005':('Recipe, baseline, waiver, version, commands','merge',['/reference/'])}
unit_hashes=['7f4934c9ccdb591bce40dbdb3c3cc145bca8532e249c19310dd48d5e70b2ca11','b68f0717ce7e9cecf0e25da1c451806766ddb8acee531a8a52ec2075fe291536','f7c9aade4df4f01bbcf8ff2c69a001d503b11b0a04db9b1457c5ab360c45536f','4e388af02a780d264334600c96fa92d2fc7599a1903664b66a1f5b03d08d38dc','32f0ed1d5632e754556e638d2b9cf17bee6f3013684c64cf67420613b34eae10','3fd94e34e1b5b2ec9a172c36a7a26cbac57961bf5fe189d6a2fd5044cb22c796','d598e6ca2c7e1421ba63b1e6f295335c38842249a6798c1ed4fdeae07dc425d6','519e4ac3d9b0e7fe15e31aec9a70423b960d648b3852a529dc1094fda22873ff','13b26f5cd479c70df80eb1ca253d1703e466a371691db9ead612bdd061cd319b','a4590561810bc47d07ceafcab35d1aabd42dcf863fe6599b79f253961517135b','ce5e0fa9dd49c4912244dfc3c0b485545c773d90370a6239ee606aff1eada722','78055712c8b8252d85fdf68b4c6f8e81dbe4b01a03ad8434bcb31d34003cf4fb','448c43ffe5005327e7d62ed2e17b6758564653d1a791ba3ec31c4a0326b951ea','3d2ec74c4a7df0a98add08ddc566d39b7c17c47ad19637de959bf065d938d10e','bf86f3247892040a1c4d90f2a58c54ad1bf46c5127291663b885a0eb245c99e2','5f1fd3183b63f35d4933f0b0701ad5773460d058aca53c29a6e4e0cdd29c1088','403a2f202142262caa1442183884f0119aed54d9c0f743fee1465243f112db08','1bcc61d361539432517863989591a6b5ddbb35ce704a731780f15fa00467006c','297bfb466b1f16964354b6032686927bd63088f2e108de57ecff05e1ddc014f5','ad385ee7eef88081ba690043b6e08421025bc0c65da872eb0d04c7f1e8b5495b','83ca7461040436bc18760c6f8697abfcbdc5aacdd07a86a1e7fef578d1cca63c','2290835113f1e6614b70742d812201c9db543354bfc6f27e78994d197149374c','ffed0b8265ac633fe0489f5cc194a73c697ee32d03c6b4b290a4b8338713f67e','23865acfca0fde9cf430d0ff962d68c5f65bbf21a2fabee173f760ec639de4e4','718d32e3701e7ac66a983278695dd004e688841af3c19e7b799ac22321d9ed7e','f927f2f6a87da6a0025cf603de0758281579be415c26e7eb4c3846e32e5b02c0','ad437a650356f813475b7e9cde77f9b24e14bb0a3e9ee63c1fe31ab4048137d7','20caf1a9aa46b786d6e424a57f277feb40fa2a60b57925f32753fe725b1c0a8a','19ac26e25551a4a0eadfcd33fdba2633a572594ee4cb3f839001a6e7abf6ae4d','094b751ea325b80921f74ccaf62a414b4ff498d4fad7384d78ca325233756f45','6787c57bcd9ee1d71abff57f5d1ca6e15edea05fc8e832818c1571272ec6e49e']
for unit,expected_hash in zip(units,unit_hashes):
 uid=unit.get('unit_id','unit')
 for field in ('source_locator','topic','summary','disposition','rationale','target_routes'):
  if field not in unit or unit[field] in ('',None): fail(uid+': '+field)
 disp=unit['disposition']; targets=unit['target_routes']
 if disp not in ('rewrite','merge','decompose','retain','retire'): fail(uid+': disposition')
 if disp in ('rewrite','merge','retain') and len(targets)!=1: fail(uid+': target_routes')
 if disp=='decompose' and len(set(targets))<2: fail(uid+': target_routes')
 if disp=='retire' and targets: fail(uid+': target_routes')
 if (unit.get('topic'),disp,targets)!=expected_unit_contract[uid]: fail(uid+': exact topic/disposition/targets')
 payload={k:unit[k] for k in ('unit_id','source_locator','topic','summary','disposition','target_routes','rationale')}
 actual_hash=hashlib.sha256(json.dumps(payload,sort_keys=True,separators=(',',':')).encode()).hexdigest()
 if actual_hash!=expected_hash: fail(uid+': exact locator/topic/summary/disposition/targets/rationale')
claim_types={'mechanism','runtime-behavior','compatibility','observed-failure','observed-outcome'}
evidence_kinds={'source','schema','test','implementation-commit','captured-execution','reproducible-execution','incident','example','measurement'}
mechanism_kinds={'source','schema','test','implementation-commit'}
evidence_ids={}
for claim in claims:
 cid=claim.get('claim_id','claim')
 for field in ('statement','statement_markdown','claim_type','owner','mechanism_summary','guarantee_boundary','known_limitations','adoption_implications','direction','evidence_refs'):
  if claim.get(field) in (None,'',[]): fail(cid+': '+field)
 if claim['claim_type'] not in claim_types: fail(cid+': claim_type')
 kinds={x.get('kind') for x in claim['evidence_refs']}
 for x in claim['evidence_refs']:
  if x.get('kind') not in evidence_kinds: fail(cid+': '+str(x.get('evidence_id'))+': unknown evidence kind '+str(x.get('kind')))
 if not kinds & mechanism_kinds: fail(cid+': mechanism evidence')
 if claim['claim_type'] in ('runtime-behavior','compatibility') and not kinds & {'captured-execution','reproducible-execution'}: fail(cid+': execution evidence')
 if claim['claim_type']=='observed-failure' and not kinds & {'incident','example'}: fail(cid+': incident/example evidence')
 if claim['claim_type']=='observed-outcome' and not kinds & {'example','measurement'}: fail(cid+': example/measurement evidence')
 if claim['claim_type']=='compatibility':
  if not all(claim.get(x) for x in ('operability','preserved_guarantees','unpreserved_guarantees')): fail(cid+': compatibility operability/guarantees')
  if claim['operability']==claim['preserved_guarantees'] or not claim['unpreserved_guarantees']: fail(cid+': compatibility guarantee equivalence')
 for ref in claim['evidence_refs']:
  eid=ref.get('evidence_id'); locator=ref.get('locator')
  if not eid or eid in evidence_ids or not isinstance(locator,dict): fail(cid+': evidence reference')
  kind=ref.get('kind')
  if kind not in evidence_kinds: fail(cid+': '+str(eid)+': unknown evidence kind '+str(kind))
  if not ref.get('relevance'): fail(cid+': evidence relevance')
  evidence_ids[eid]=ref
  if 'path' in locator and not os.path.exists(os.path.join(root,locator['path'])): fail(cid+': missing evidence path '+locator['path'])
  if 'version' in locator and not re.fullmatch(r'v\d+\.\d+\.\d+',str(locator['version'])): fail(cid+': mutable version locator')
  if 'commit' in locator:
   commit=str(locator['commit'])
   if not re.fullmatch(r'[0-9a-f]{40}',commit): fail(cid+': '+str(eid)+': mutable commit locator '+commit)
   resolved=subprocess.run(['git','-C',git_root,'cat-file','-e',commit+'^{commit}'],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
   if resolved.returncode!=0: fail(cid+': '+str(eid)+': nonexistent commit '+commit)
   if 'version' in locator:
    tagged=subprocess.run(['git','-C',git_root,'rev-list','-n','1',str(locator['version'])],text=True,stdout=subprocess.PIPE,stderr=subprocess.DEVNULL)
    if tagged.returncode!=0 or tagged.stdout.strip()!=commit: fail(cid+': '+str(eid)+': version/commit provenance mismatch')
  if not any(k in locator for k in ('path','version','commit')): fail(cid+': durable locator')
  if ref.get('kind')=='reproducible-execution' and (not ref.get('command') or not ref.get('prerequisites')): fail(cid+': reproducible execution fields')
  if kind in ('captured-execution','incident','example','measurement') and not ref.get('observation_name'): fail(cid+': '+str(eid)+': unnamed observation')
  if kind in ('captured-execution','example'):
   if not ref.get('artifact') or not os.path.exists(os.path.join(root,ref['artifact'])): fail(cid+': provenance-bearing artifact')
  if kind=='example' and not ref.get('provenance'): fail(cid+': '+str(eid)+': example provenance')
  if kind=='incident':
   p=locator.get('path','')
   if not p or not re.search(r'(?i)(issue|incident|report)',os.path.basename(p)): fail(cid+': '+str(eid)+': incident must identify durable issue or report')
  if kind=='measurement' and not all(ref.get(x) for x in ('provenance','population','period','method')): fail(cid+': measurement provenance')
# These seed records make deliberately narrow public assertions. Keep their truth
# contract explicit so an open issue or narrative cannot be relabeled as proof.
truth_contract={
 'CLAIM-007':('currently interpreted','missing literal-escape capability'),
 'CLAIM-008':('Linux CI exposed','f8b3846fe5d4c2bc6465efc6eb5e4594e1b591da')}
for cid,needles in truth_contract.items():
 statement=next(x for x in claims if x.get('claim_id')==cid).get('statement','')
 if not all(x in statement for x in needles): fail(cid+': evidence truth contract')
exact_evidence_paths={
 'EVIDENCE-009':'pkg/recipe/substitute.go',
 'EVIDENCE-010':'issues/ISSUE-182-recipe-literal-placeholder-escaping.issue.md',
 'EVIDENCE-012':'issues/ISSUE-166-contracts-pack-phase3-fixtures-fail-on-linux-ci.issue.md',
 'EVIDENCE-014':'issues/ISSUE-097-unbound-selfpack-waivers-fail-open.issue.md'}
for eid,path in exact_evidence_paths.items():
 if evidence_ids[eid].get('locator',{}).get('path')!=path: fail(eid+': evidence truth/provenance path')
if evidence_ids['EVIDENCE-011'].get('locator',{}).get('commit')!='f8b3846fe5d4c2bc6465efc6eb5e4594e1b591da': fail('EVIDENCE-011: source trace commit')
for eid in ('EVIDENCE-012','EVIDENCE-014'):
 artifact=evidence_ids[eid]['artifact']; body=open(os.path.join(root,artifact),encoding='utf-8').read()
 if not re.search(r'(?m)^\s*status:\s*closed\s*$',body): fail(eid+': qualifying artifact must be closed')
roles=evidence.get('corpus_roles',{})
if set(roles)!={'failure_incident','failure_to_enforcement_example','captured_gate_result','source_or_commit_trace','architecture_view'} or len(set(roles.values()))!=5: fail('five distinct corpus roles')
role_kinds={'failure_incident':{'incident'},'failure_to_enforcement_example':{'example'},'captured_gate_result':{'captured-execution'},'source_or_commit_trace':{'source','implementation-commit'},'architecture_view':{'source'}}
for role,eid in roles.items():
 ref=evidence_ids.get(eid)
 if not ref: fail(role+': unknown evidence '+str(eid))
 if ref.get('kind') not in role_kinds[role]: fail(role+': '+str(eid)+': nonqualifying kind '+str(ref.get('kind')))
 if not ref.get('relevance'): fail(role+': '+str(eid)+': missing relevance')
 if role=='failure_to_enforcement_example' and not all(ref.get(x) for x in ('artifact','provenance','before','after')): fail(role+': '+str(eid)+': requires provenance-bearing before/after example')
 if role=='failure_to_enforcement_example' and eid!='EVIDENCE-012': fail(role+': '+str(eid)+': semantically unqualified failure-to-enforcement example')
 if role=='captured_gate_result' and eid!='EVIDENCE-014': fail(role+': '+str(eid)+': semantically unqualified captured gate result')
 if role=='source_or_commit_trace' and eid!='EVIDENCE-011': fail(role+': '+str(eid)+': semantically unqualified source trace')
 if role=='architecture_view' and not str(ref.get('locator',{}).get('path','')).endswith('.mmd'): fail(role+': '+str(eid)+': requires Mermaid architecture source')
if mode=='--registries-only': print('public-product-model: registries PASS'); raise SystemExit(0)

claim_start=re.compile(r'^<!-- backstop-claim: ([A-Z0-9-]+) -->\n',re.M)
claim_end=re.compile(r'^<!-- /backstop-claim -->$',re.M)
jlink024='<!-- backstop-journey-link: JLINK-024 -->\n'
jlink024_link='[Continue outside Backstop](/contributing/#external-ownership)'
def claim_regions(page,text):
 starts=list(claim_start.finditer(text)); ends=list(claim_end.finditer(text))
 if len(starts)!=len(ends): fail(page+': unpaired claim region')
 result=[]
 for s in starts:
  end=claim_end.search(text[s.end():])
  if not end: fail(page+': unclosed claim '+s.group(1))
  body=text[s.end():s.end()+end.start()]
  if body.endswith('\n'): body=body[:-1]
  if '<!-- backstop-claim:' in body: fail(page+': nested claim '+s.group(1))
  result.append((s,s.group(1),body))
 return result
def canonical_payload(page,cid,body):
 body=body.replace('{% raw %}','').replace('{% endraw %}','')
 if cid!='CLAIM-005': return body
 if body.count(jlink024.rstrip('\n'))!=1: fail(page+': CLAIM-005 requires exactly one JLINK-024 marker')
 if jlink024+jlink024_link not in body: fail(page+': CLAIM-005 JLINK-024 marker must immediately precede continuation link')
 canonical=body.replace(jlink024,'',1)
 if '<!--' in canonical: pass
 return canonical
if mode=='--claim-005-layout-only':
 p='docs/status.md'; text=open(os.path.join(root,p),encoding='utf-8',newline='').read().replace('\r\n','\n')
 regions=claim_regions(p,text)
 matches=[x for x in regions if x[1]=='CLAIM-005']
 if len(matches)!=1: fail(p+': CLAIM-005 cardinality')
 _,cid,body=matches[0]; canonical=canonical_payload(p,cid,body)
 if canonical!=cmap[cid]['statement_markdown']: fail(p+': CLAIM-005 canonical visible bytes')
 marker_pos=text.find(jlink024.rstrip('\n')); start=matches[0][0].start(); end=text.find('<!-- /backstop-claim -->',start)
 if not (start < marker_pos < end): fail(p+': JLINK-024 must be inside CLAIM-005')
 print('public-product-model: CLAIM-005 layout PASS'); raise SystemExit(0)

texts={}
for p,route in zip(sources,paths):
 full=os.path.join(root,p)
 if not os.path.isfile(full): fail('missing '+p)
 text=open(full,encoding='utf-8',newline='').read().replace('\r\n','\n')
 if re.search(r'(?i)\b(TODO|TBD|lorem ipsum|draft placeholder)\b',text): fail(p+': draft placeholder')
 if re.search(r'(?i)Cayman theme|theme: jekyll-theme-cayman',text): fail(p+': stale Cayman positioning')
 anchors=re.findall(r'^#{1,6} .+ \{#([a-z0-9-]+)\}\s*$',text,re.M)
 if len(anchors)!=len(set(anchors)): fail(p+': duplicate explicit anchor')
 required=next(x for x in pages if x['source']==p).get('required_blocks',[])
 for a in required:
  if a not in anchors: fail(p+': missing anchor '+a)
 hero=next(x for x in pages if x['source']==p)['hero_question']
 if text.count(hero)!=1: fail(p+': hero question must occur once in owning frontmatter; presentation consumes it separately')
 texts[route]=(p,text,anchors)

legacy_responsibilities={
'GET-004':[('/reference/','troubleshooting','`backstop doctor` diagnoses configuration discovery')],
'ART-003':[('/model/','operating-model','Terminal state records whether work was delivered'),('/reference/','artifact-lifecycle-and-closure','Bundles progress through `idea`, `exploring`, `defined`, and `ready`')],
'ART-004':[('/model/','operating-model','`delivered_by` or a direct typed artifact'),('/reference/','artifact-lifecycle-and-closure','`delivered_by` names a completed plan')],
'PACK-004':[('/extend/','path-filter-diagnostics','slash-bearing include or exclude pattern'),('/status/','path-filter-limitation','Slash-bearing engine path patterns can fail open')],
'CLI-001':[('/reference/','cli-conventions','return `0` for success, `1` for blocking violations or broken promises, and `2` for configuration failure')],
'CLI-002':[('/adopt/','install',commands[0]),('/adopt/','configure',commands[1]),('/adopt/','verify-enforcement',commands[2]),('/reference/','troubleshooting','`backstop doctor` diagnoses'),('/reference/','gate','`backstop gate` evaluates changed files by default')],
'CLI-003':[('/reference/','pack-commands','`backstop pack install`')],
'CLI-004':[('/reference/','artifact-schema-catalog','Artifact schemas live under')],
'CLI-005':[('/reference/','cli-command-catalog','initialization, diagnosis, gates, packs, artifacts, recipes, baselines, waivers, version reporting, and command discovery')]}
def anchored_section(route,anchor):
 p,text,_=texts[route]
 heading=re.search(r'^#{1,6} .+ \{#'+re.escape(anchor)+r'\}\s*$',text,re.M)
 if not heading: fail(p+': missing responsibility anchor '+anchor)
 nxt=re.search(r'^#{1,6} .+ \{#[a-z0-9-]+\}\s*$',text[heading.end():],re.M)
 return text[heading.end():heading.end()+(nxt.start() if nxt else len(text))]
for uid,requirements in legacy_responsibilities.items():
 for route,anchor,needle in requirements:
  if needle not in anchored_section(route,anchor):
   instruction_id=next((x['instruction_id'] for x in instructions if x.get('command_text')==needle),None)
   fail(uid+('/'+instruction_id if instruction_id else '')+': missing final-copy responsibility at '+route+'#'+anchor+' field content')

# Every substantive Markdown block is closed-world accepted: it is a claim region,
# source-owned journey link, adoption code block, exact hero, heading, or one of these
# narrowly justified nonconsequential blocks. This prevents new prose from bypassing
# claim linkage without attempting the reusable semantic judgments owned by SPEC-073.
allowances={
'docs/index.md':{
'Evaluate the control surface, learn the operating model, or move directly into a small working adoption. The site follows those decisions instead of mirroring the repository tree.':'navigation connective',
 '<p class="home-lede">The problem is not generating more code. The problem is knowing whether the code that arrived is the code you actually asked for, whether it still follows the standards you intended to preserve, and whether a green result means the same thing tomorrow that it means today.</p>':'HOME-001 discovery framing',
 '<div class="home-capabilities" data-home-capabilities>\n  <article>\n    <span>01</span>\n    <h3>Define the work</h3>\n    <p>Use explicit artifacts to make intent inspectable before implementation begins. Larger work moves from bundle to spec to plan; smaller reactive work can move from issue to plan. Both paths create a concrete acceptance boundary instead of asking an agent to infer one.</p>\n  </article>\n  <article>\n    <span>02</span>\n    <h3>Enforce your standards</h3>\n    <p>Put mechanical engineering decisions in versioned packs and run them while the agent works and again at integration. Architecture boundaries, test expectations, dependency policy, and implementation patterns stop being prompt suggestions and become blocking rules.</p>\n  </article>\n  <article>\n    <span>03</span>\n    <h3>Detect drift</h3>\n    <p>Compare standards against an explicit baseline and requirements against implementation evidence. Existing debt can remain visible without becoming permanent permission for new debt, while completion claims stay tied to the tests and evidence that make them true.</p>\n  </article>\n</div>':'HOME-002 define/enforce/drift summary',
 '<div class="home-gate-proof" data-home-gate-proof aria-label="Example Backstop gate result">\n  <div class="home-gate-command"><span aria-hidden="true">$</span> backstop gate</div>\n  <div class="home-gate-results">\n    <span>✓ pack integrity</span>\n    <span>✓ artifacts</span>\n    <span>✓ engineering standards</span>\n    <span>✓ tests</span>\n    <span>✓ requirements</span>\n    <strong>PASS · exit 0</strong>\n  </div>\n</div>':'HOME-002 illustrative gate presentation',
 'Backstop does not make probabilistic systems deterministic. It makes the important boundaries around them deterministic: what work was promised, which standards apply, what evidence must exist, and what conditions are allowed to produce a passing verdict.':'HOME-002 discovery connective',
 'You do not have to adopt the entire framework at once. Start with the failure you need to control, then add the pieces that make that boundary enforceable.':'visitor decision connective',
 '<div class="home-paths" data-home-paths>\n  <article>\n    <span>Evaluate</span>\n    <h3>Decide whether Backstop fits</h3>\n    <p>Start with the failure class, guarantees, limits, compatibility boundary, and evidence model before changing a repository.</p>\n    <a href="/evaluate/">Evaluate the control surface →</a>\n  </article>\n  <article>\n    <span>Understand</span>\n    <h3>See how the pieces reinforce one another</h3>\n    <p>Follow the artifact lifecycle, enforcement loop, ownership boundaries, and the point where deterministic checks replace agent judgment.</p>\n    <a href="/model/">Read the operating model →</a>\n  </article>\n  <article>\n    <span>Adopt</span>\n    <h3>Put one real standard behind the gate</h3>\n    <p>Install Backstop, initialize the repository, choose a maintained pack, and prove that a violation actually blocks before expanding the surface.</p>\n    <a href="/adopt/">Start a working adoption →</a>\n  </article>\n</div>':'visitor decision paths',
 'The framework is composable: artifacts can make delivery intent traceable without policy packs; packs and the gate can enforce standards without the full artifact chain; recipes can provide deterministic scaffolding without owning enforcement. The pieces are stronger together, but each should earn its place by controlling a real failure.':'HOME-003 composability summary',
 "> If it has to be right, it must be deterministic. If it's green, it ships.":'product principle presentation'},
'docs/evaluate.md':{
 '<div class="tactics-intro">\n<p class="tactics-kicker">Bigger models write great code. None of them write code like you.</p>\n<p class="tactics-bridge">Backstop enforces your standards so the agent\'s code looks like your code.</p>\n</div>':'visitor category sentence',
 '<div class="tactics-matrix">\n<table>\n<thead>\n<tr><th>What you already use</th><th>What that gets you</th><th>What it cannot guarantee</th><th>What Backstop adds</th><th>Result</th></tr>\n</thead>\n<tbody>\n<tr><td data-label="What you already use">Markdown specs</td><td data-label="What that gets you">Named intent</td><td data-label="What it cannot guarantee">The agent can skip them</td><td data-label="What Backstop adds">Plan-before-code, verifiable claims, mandated tests</td><td data-label="Result">“Done” can be contradicted</td></tr>\n<tr><td data-label="What you already use">Skills / AGENTS.md</td><td data-label="What that gets you">Better default behavior</td><td data-label="What it cannot guarantee">A prompt, not a verdict</td><td data-label="What Backstop adds">Packs and fixtures for the non-negotiable subset</td><td data-label="Result">Green means the standard held</td></tr>\n<tr><td data-label="What you already use">MCP</td><td data-label="What that gets you">Tool access</td><td data-label="What it cannot guarantee">A protocol, not a policy</td><td data-label="What Backstop adds">A required gate whose exit status controls the workflow</td><td data-label="Result">The exit code controls what happens next</td></tr>\n<tr><td data-label="What you already use">LLM review</td><td data-label="What that gets you">Another opinion on the diff</td><td data-label="What it cannot guarantee">Still a guess</td><td data-label="What Backstop adds">Deterministic engines for what must not be a guess</td><td data-label="Result">Same input, same verdict</td></tr>\n<tr><td data-label="What you already use">Standards as docs</td><td data-label="What that gets you">Shared language</td><td data-label="What it cannot guarantee">Unenforced</td><td data-label="What Backstop adds">Encode only what you would fail a merge over</td><td data-label="Result">The rest stays a doc on purpose</td></tr>\n</tbody>\n</table>\n</div>':'visitor tactics matrix',
 'SDD gives the agent better instructions. The artifact chain gives it less to guess.':'visitor decision guidance',
 'A spec tells the agent what you want. The artifact chain tells it what to do next, what it may assume, what it must prove, and when it is done.':'visitor decision guidance',
 'Most spec-driven workflows hand the agent one or more Markdown documents and leave it to infer the rest. Backstop turns intent into bounded, linked work: requirements become a plan, the plan produces implementation claims, and those claims require evidence.':'visitor decision guidance',
 'At every stage, the agent knows what is authoritative, what remains unresolved, and what must be true before the work can proceed.':'visitor decision guidance',
 '<div class="tactics-matrix sdd-matrix">\n<table>\n<thead>\n<tr><th>Common SDD</th><th>What the agent still has to infer</th><th>What the artifact chain makes explicit</th><th>Result</th></tr>\n</thead>\n<tbody>\n<tr><td data-label="Common SDD">A single spec.md</td><td data-label="What the agent still has to infer">Decomposition, dependencies, and boundaries</td><td data-label="What the artifact chain makes explicit">Atomic requirements and explicit relationships</td><td data-label="Result">A smaller problem to solve</td></tr>\n<tr><td data-label="Common SDD">Generated plan and tasks</td><td data-label="What the agent still has to infer">Whether every requirement survived planning</td><td data-label="What the artifact chain makes explicit">Validated coverage between artifacts</td><td data-label="Result">Nothing quietly falls out</td></tr>\n<tr><td data-label="Common SDD">Checked task boxes</td><td data-label="What the agent still has to infer">What was actually implemented and proven</td><td data-label="What the artifact chain makes explicit">Claims tied to evidence and terminal states</td><td data-label="Result">“Done” can be contradicted</td></tr>\n<tr><td data-label="Common SDD">A revised specification</td><td data-label="What the agent still has to infer">Which existing work is now stale</td><td data-label="What the artifact chain makes explicit">Required downstream reconciliation</td><td data-label="Result">Change does not become drift</td></tr>\n<tr><td data-label="Common SDD">A new agent session</td><td data-label="What the agent still has to infer">Decisions, progress, and unresolved work</td><td data-label="What the artifact chain makes explicit">Durable artifact and workflow state</td><td data-label="Result">Resume without reconstruction</td></tr>\n</tbody>\n</table>\n</div>':'visitor sdd matrix',
 'The same structure that makes the agent easier to trust also makes the agent better at the job.':'visitor decision guidance',
 '<div class="failed-verdict" aria-label="Example failing Backstop gate">\n<div class="failed-verdict-bar"><span>backstop gate</span><span>exit 1</span></div>\n<div class="failed-verdict-row"><span>Tests</span><span>promised test is absent</span><strong>fail</strong></div>\n<div class="failed-verdict-row"><span>Requirements</span><span>completion claim without coverage</span><strong>fail</strong></div>\n<div class="failed-verdict-foot"><strong>FAIL</strong><span>The work is not allowed to ship.</span></div>\n</div>':'example failed verdict',
 'Fix problems while your agent is still writing the code.':'visitor decision guidance',
 'Backstop puts deterministic gates inside the agent\'s working loop. The agent gets the failure, fixes the issue, and reruns the gate before the work reaches review. CI runs the same checks again—but confirms the verdict instead of delivering the surprise.':'visitor decision guidance',
 '<div class="ci-workflows">\n<article>\n<p>Typical workflow</p>\n<ol>\n<li>Agent writes</li>\n<li>PR opens</li>\n<li>CI fails</li>\n<li>Human or agent reconstructs context</li>\n<li>Retry</li>\n</ol>\n</article>\n<article>\n<p>Backstop workflow</p>\n<ol class="has-loop">\n<li>Agent writes</li>\n<li>Gate fails</li>\n<li>Agent fixes</li>\n<li>PR opens</li>\n<li>CI confirms</li>\n</ol>\n</article>\n</div>':'workflow comparison',
 'CI should confirm, not discover.':'visitor decision guidance'},
'docs/model.md':{
 'One bundle. Many specs. Each spec one plan.':'visitor operating guide',
 '<div class="work-topology">\n<div class="topo-flow" data-overflow-region>\n<div class="topo-bundle">Bundle</div>\n<div class="topo-arrow" aria-hidden="true">→</div>\n<div class="topo-stack">\n<div class="topo-track"><span class="topo-node spec">Spec A</span><span class="topo-arrow" aria-hidden="true">→</span><span class="topo-node plan">Plan A</span></div>\n<div class="topo-track"><span class="topo-node spec">Spec B</span><span class="topo-arrow" aria-hidden="true">→</span><span class="topo-node plan">Plan B</span></div>\n</div>\n<div class="topo-arrow" aria-hidden="true">→</div>\n<div class="topo-track"><span class="topo-node spec">Spec C</span><span class="topo-arrow" aria-hidden="true">→</span><span class="topo-node plan">Plan C</span></div>\n</div>\n<p class="topo-order">A and B can run together. C waits.</p>\n</div>':'visitor work topology',
 '<div class="tactics-matrix legend-matrix">\n<table>\n<tbody>\n<tr><td data-label="Piece">Bundle</td><td data-label="What it is">The body of work</td></tr>\n<tr><td data-label="Piece">Spec</td><td data-label="What it is">One bounded implementation contract</td></tr>\n<tr><td data-label="Piece">Plan</td><td data-label="What it is">The ordered steps that realize that spec</td></tr>\n<tr><td data-label="Piece">Dependencies</td><td data-label="What it is">The known-safe execution order</td></tr>\n</tbody>\n</table>\n</div>':'visitor operating table',
 'Independent branches can execute in parallel. Dependencies establish the order when they cannot.':'visitor operating guide',
 'A bounded fix skips the bundle and spec. The issue becomes a plan.':'visitor operating guide',
 '<div class="canonical-note">':'canonical source note',
 '<!-- backstop-journey-link: JLINK-010 -->\n[Inspect artifact schemas](/reference/#artifact-schema-catalog)\n</div>':'visitor operating guide',
 '<div class="tactics-matrix">\n<table>\n<tbody>\n<tr><td data-label="Piece">Packs</td><td data-label="What it is">Versioned standards with engines and fixtures. Core only runs what a pack declares.</td></tr>\n<tr><td data-label="Piece">Gate</td><td data-label="What it is">Pass or fail on the installed packs. Same input, same verdict.</td></tr>\n<tr><td data-label="Piece">Recipes</td><td data-label="What it is">Pinned scaffolding — a project, a pipeline, a config. They do not own the standard.</td></tr>\n</tbody>\n</table>\n</div>':'visitor operating table',
 '<div class="tactics-matrix">\n<table>\n<tbody>\n<tr><td data-label="Piece">Baseline</td><td data-label="What it is">Existing debt stays visible. Touched code and new violations fail.</td></tr>\n<tr><td data-label="Piece">Waiver</td><td data-label="What it is">A named, time-bounded exception. The rule stays.</td></tr>\n<tr><td data-label="Piece">Provenance</td><td data-label="What it is">Green has to point at source, command, and tests.</td></tr>\n</tbody>\n</table>\n</div>':'visitor operating table',
 '<div class="tactics-matrix">\n<table>\n<tbody>\n<tr><td data-label="Piece">Agent loop</td><td data-label="What it is">The gate fails to the agent. The agent fixes it before review.</td></tr>\n<tr><td data-label="Piece">CI</td><td data-label="What it is">Confirms the same verdict. Does not discover it.</td></tr>\n</tbody>\n</table>\n</div>':'visitor operating table',
 '<div class="canonical-note">\nThe authoritative enforcement loop is `docs/_diagrams/ARCH-002-enforcement-loop.mmd`: intent bounds execution, pack engines return a verdict, and evidence feeds provenance back into intent.':'canonical source note',
 '<!-- backstop-journey-link: JLINK-014 -->\n[Read the gate reference](/reference/#gate)\n</div>':'visitor operating guide',
 '<div class="tactics-matrix">\n<table>\n<tbody>\n<tr><td data-label="Piece">Core</td><td data-label="What it is">Runs the process.</td></tr>\n<tr><td data-label="Piece">Packs</td><td data-label="What it is">Own the standards.</td></tr>\n<tr><td data-label="Piece">Harness</td><td data-label="What it is">Must honor the exit.</td></tr>\n<tr><td data-label="Piece">Agent</td><td data-label="What it is">Backstop sits around it. It is not a coding agent.</td></tr>\n</tbody>\n</table>\n</div>':'visitor operating table',
 '<div class="canonical-note">':'canonical source note',
 'Core owns execution and lifecycle primitives. Packs own standards and engines. Harnesses own orchestration. External toolchains own their behavior. The authoritative boundary view is `docs/_diagrams/ARCH-003-ownership-boundaries.mmd`.':'architecture-source identification',
 '<!-- backstop-journey-link: JLINK-011 -->\n[Review project boundaries](/status/#project-boundaries)\n</div>':'visitor operating guide',
 '<div class="canonical-anchors">':'canonical claim anchors',
 '</div>':'canonical claim anchors'
},
'docs/adopt.md':{
 'Start in a disposable repository, prove one gate, and only then widen the policy surface. A first adoption needs Go, Git, an explicit project root, and a standard worth making non-negotiable.':'adoption instruction',
 'Install the exact released binary into the disposable repository rather than relying on a machine-global copy.':'adoption instruction',
 'Initialize the repository-owned declaration.':'adoption instruction',
 'Inspect the created `backstop.yml`, select a maintained pack, and keep the pinned declaration in version control.':'adoption instruction',
 'Run the gate from the repository root.':'adoption instruction'},
'docs/use-cases.md':{
 'Start from the failure you need to prevent: policy drift, unreviewable agent output, inconsistent artifact execution, or a standard that exists only in prose. Then choose the smallest enforceable seam.':'visitor decision guidance',
 'Use a maintained pack when the concern is shared, repeatable, and already has an owner. Compose packs for architecture, language contracts, CI, security, or repository conventions instead of copying their rules into prompts.':'visitor decision guidance'},
'docs/packs.md':{
 "Choose the narrowest maintained pack that owns the standard and supports the repository's tools. Check its release, engine requirements, fixture coverage, and maintenance state before composing it with other packs.":'selection guidance'},
'docs/extend.md':{
 'Create a pack when a concern is deterministic, reusable across repositories, independently versionable, and owned by a maintainable standard. Keep repository-specific wiring local when reuse would manufacture an abstraction without a real consumer.':'author decision guidance',
 'Scaffold the declaration, define claims, bind each claim to an engine and fixtures, and make negative cases prove the finding. Run the pack against representative repositories, preserve path-filter diagnostics, and publish an immutable release only after its own gate passes.':'author instruction',
 'Rules explain the violation, fixtures prove detection, and tool pins make execution reproducible. Iteration belongs inside the pack rather than in every consuming repository.':'author instruction',
 'When an engine receives explicit changed-file arguments, do not assume a slash-bearing include or exclude pattern behaves as it does during directory traversal. Use `pack check` and `pack test` to surface path-scope advisories, preserve production-relative fixture paths, and prefer a slash-free single-segment pattern only when it retains the intended scope.':'path-filter repair instruction'},
'docs/reference.md':{
 'Durable references use repository paths, immutable commits, or published versions. Execution evidence includes the exact command and prerequisites; observation evidence identifies its checked-in artifact and provenance.':'evidence-record definition'},
'docs/status.md':{
 'The first statement names shipped mechanism. The second names the process boundary without turning it into a future commitment.':'claim relationship explanation',
 'Public status uses five explicit states: supported, limitation, planned, non-goal, and adjacent guidance. Each state has durable sources and a visitor implication.':'registry vocabulary explanation'},
'docs/contributing.md':{
 'Contribute to Core when the change belongs to lifecycle, execution, artifact infrastructure, or a shared interface. Contribute to a pack when the change owns a standard, engine, fixture, or rule. Bring a reproducible failure, a bounded artifact chain, tests first, and the gate receipt.':'contribution instruction',
 'Agent scheduling, organization-wide policy adoption, external toolchain correctness, and operational response retain their own owners. For the product boundary and continuation path, follow the evidence-backed [adjacent guidance](/status/#adjacent-guidance).':'owner-routing connective'}}
def check_closed_blocks(page,text,hero):
 masked=text
 front=re.match(r'^---\n.*?\n---\n',masked,re.S)
 if front: masked=re.sub(r'[^\n]',' ',masked[:front.end()])+masked[front.end():]
 for s,_,_ in reversed(claim_regions(page,text)):
  end=claim_end.search(text,s.end())
  stop=end.end() if end else s.end()
  masked=masked[:s.start()]+re.sub(r'[^\n]',' ',text[s.start():stop])+masked[stop:]
 masked=re.sub(r'^```[^\n]*\n.*?^```\s*$',lambda m:re.sub(r'[^\n]',' ',m.group(0)),masked,flags=re.M|re.S)
 for instruction in instructions:
  rendered='<pre><code>'+instruction['command_text']+'</code></pre>'
  if rendered in masked: masked=masked.replace(rendered,re.sub(r'[^\n]',' ',rendered),1)
 masked=re.sub(r'(?:<section data-generated-region data-product-truth-job="[a-z0-9-]+">\n)?<!-- PRODUCT-TRUTH-INCLUDE:BEGIN job=[a-z0-9-]+ -->\n\{% include generated/[a-z0-9-]+\.md %\}\n<!-- PRODUCT-TRUTH-INCLUDE:END job=[a-z0-9-]+ -->(?:\n</section>)?',lambda m:re.sub(r'[^\n]',' ',m.group(0)),masked)
 for block in (x.strip() for x in re.split(r'\n[ \t]*\n',masked)):
  if not block or block==hero or block.startswith('#'): continue
  if re.fullmatch(r'<!-- backstop-journey-link: JLINK-\d{3} -->\n\[[^\n]+\]\(/[^\n]+\)',block): continue
  if block not in allowances.get(page,{}): fail(page+': unclassified final-copy block requires claim linkage or explicit nonconsequential allowance: '+block[:100])
# Diagnose declared-link loss/cardinality before the closed-block policy so page
# mutations retain the responsible stable journey ID.
for link in links:
 if link['link_id']=='JLINK-024': continue
 p,text,_=texts[link['source_route']]; marker='<!-- backstop-journey-link: '+link['link_id']+' -->'
 expected=marker+'\n['+link['label']+']('+link['destination_route']+'#'+link['destination_anchor']+')'
 if text.count(marker)!=1 or text.count(expected)!=1: fail(p+': '+link['link_id']+' marker/link cardinality')
for route,(p,text,_) in texts.items():
 check_closed_blocks(p,text,next(x for x in pages if x['source']==p)['hero_question'])

for concept in concepts:
 owner=concept.get('owner',{}); route=owner.get('route'); anchor=owner.get('anchor')
 if route not in texts or anchor not in texts[route][2]: fail(concept.get('concept_id','concept')+': owner anchor')
for view in views:
 owner=view.get('owner',{}); route=owner.get('route'); anchor=owner.get('anchor')
 if route not in texts or anchor not in texts[route][2]: fail(view.get('architecture_id','ARCH')+': owner anchor')

seen_claims={}
for route,(p,text,anchors) in texts.items():
 for s,cid,body in claim_regions(p,text):
  if cid in seen_claims: fail(p+': duplicate claim '+cid)
  prior=text[:s.start()]; hs=list(re.finditer(r'^#{1,6} .+ \{#([a-z0-9-]+)\}\s*$',prior,re.M)); anchor=hs[-1].group(1) if hs else None
  c=cmap.get(cid)
  canonical=canonical_payload(p,cid,body)
  if not c or c['owner']!={'route':route,'anchor':anchor} or c.get('statement_markdown')!=canonical: fail(p+': claim linkage/bytes '+cid)
  seen_claims[cid]=p
if set(seen_claims)!=set(cmap): fail('claim-region bijection mismatch: '+','.join(sorted(set(cmap)^set(seen_claims))))
for link in links:
 p,text,_=texts[link['source_route']]; marker='<!-- backstop-journey-link: '+link['link_id']+' -->'
 expected=marker+'\n['+link['label']+']('+link['destination_route']+'#'+link['destination_anchor']+')'
 if text.count(marker)!=1 or expected not in text: fail(p+': '+link['link_id']+' marker/link')
 pos=text.index(marker); hs=list(re.finditer(r'^#{1,6} .+ \{#([a-z0-9-]+)\}\s*$',text[:pos],re.M))
 if not hs or hs[-1].group(1)!=link['source_anchor']: fail(p+': '+link['link_id']+' source anchor')
 dest_text=texts[link['destination_route']][1]
 if len(re.findall(r'^#{1,6} .+ \{#'+re.escape(link['destination_anchor'])+r'\}\s*$',dest_text,re.M))!=1: fail(p+': '+link['link_id']+' destination anchor')
for ins in instructions:
 p,text,_=texts[ins['owner_route']]
 if text.count(ins['command_text'])!=1: fail(p+': '+ins['instruction_id']+' displayed command cardinality')
 pos=text.index(ins['command_text']); hs=list(re.finditer(r'^#{1,6} .+ \{#([a-z0-9-]+)\}\s*$',text[:pos],re.M))
 if not hs or hs[-1].group(1)!=ins['owner_anchor']: fail(p+': '+ins['instruction_id']+' owner anchor')
print('public-product-model: PASS')
PY

if [[ "$mode" == "full" && "${BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH:-0}" != "1" ]]; then
  export BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1
  "$repo_root/scripts/tests/public-product-model/verify-content-inventory.sh"
  "$repo_root/scripts/tests/public-product-model/verify-content-topology.sh"
  "$repo_root/scripts/tests/public-product-model/verify-product-model.sh"
  "$repo_root/scripts/tests/public-product-model/verify-evidence-inventory.sh"
  "$repo_root/scripts/tests/public-product-model/verify-structural-verifier.sh"
  "$repo_root/scripts/tests/public-product-model/pages/discovery-evaluation-adoption-status.sh"
  "$repo_root/scripts/tests/public-product-model/pages/model-use-cases-packs.sh"
  "$repo_root/scripts/tests/public-product-model/pages/extend-reference-contributing.sh"
fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  verify_public_product_model "$@"
fi
