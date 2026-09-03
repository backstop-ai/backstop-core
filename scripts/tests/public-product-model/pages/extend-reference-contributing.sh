#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
corpus_copy() {
  local tmp; tmp="$(mktemp -d)"
  for d in docs artifacts pkg cmd bundles issues; do cp -R "$root/$d" "$tmp/$d"; done
  cp "$root/README.md" "$root/backstop.yml" "$root/backstop.lock" "$tmp/"
  printf '%s' "$tmp"
}
run_verifier_expect_fail() {
  local tmp="$1" expected="$2" output
  if output="$(BACKSTOP_PUBLIC_MODEL_ROOT="$tmp" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$root" BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 "$root/scripts/verify-public-product-model.sh" 2>&1)"; then
    rm -rf "$tmp"; echo "expected verifier failure but got success" >&2; exit 1
  fi
  grep -Fq "$expected" <<<"$output" || { rm -rf "$tmp"; echo "missing diagnostic '$expected': $output" >&2; exit 1; }
  rm -rf "$tmp"
}
verify_reference_artifact_lifecycle_machine() {
python3 - "$root" <<'PY'
import os,re,sys,yaml
r=sys.argv[1]
ref=open(os.path.join(r,'docs/reference.md'),encoding='utf-8',newline='').read().replace('\r\n','\n')
ev=yaml.safe_load(open(os.path.join(r,'docs/_data/evidence-inventory.yml')))
heading='## Artifact lifecycle and closure {#artifact-lifecycle-and-closure}'
assert ref.count(heading)==1,'lifecycle heading cardinality'
start=re.search(r'^#{1,6} .+ \{#artifact-lifecycle-and-closure\}\s*$',ref,re.M)
assert start,'missing lifecycle heading'
nxt=re.search(r'^#{1,6} .+ \{#[a-z0-9-]+\}\s*$',ref[start.end():],re.M)
section=ref[start.end():start.end()+(nxt.start() if nxt else len(ref))]
assert re.search(r'^## Source traceability \{#source-traceability\}\s*$',ref[start.end()+nxt.start():],re.M),'next anchored heading must be source-traceability'
assert not re.search(r'^#{1,6} .+ \{#[a-z0-9-]+\}\s*$',section,re.M),'no explicit anchor inside lifecycle machine'
for noun in ('Issue','Spec','Bundle','Plan'):
 assert section.count('### '+noun)==1,'missing ### '+noun
assert section.count('<div class="tactics-matrix">')==4,'tactics-matrix cardinality'
for panel in re.findall(r'<div class="tactics-matrix">(.*?)</div>',section,re.S):
 assert '<th>State</th>' in panel and '<th>Before you can enter it</th>' in panel
 assert '<th>Validator / gate</th>' in panel and '<th>Enables</th>' in panel
assert section.count('<div class="state-index" aria-label="Live states">')==1,'state-index cardinality'
assert section.count('state-coupling')==1,'state-coupling cardinality'
assert '<!-- backstop-claim: CLAIM-030 -->' in section and '<!-- backstop-claim: CLAIM-031 -->' in section
art003='Bundles progress through `idea`, `exploring`, `defined`, and `ready`'
art004='`delivered_by` names a completed plan'
assert ref.count(art003)==1,'ART-003 needle cardinality'
assert ref.count(art004)==1,'ART-004 needle cardinality'
claim_start=re.compile(r'^<!-- backstop-claim: ([A-Z0-9-]+) -->\n',re.M)
claim_end=re.compile(r'^<!-- /backstop-claim -->$',re.M)
def claim_body(text,cid):
 for s in claim_start.finditer(text):
  if s.group(1)!=cid: continue
  end=claim_end.search(text[s.end():])
  body=text[s.end():s.end()+end.start()]
  if body.endswith('\n'): body=body[:-1]
  return body
body=claim_body(section,'CLAIM-030')
claim=next(c for c in ev['claims'] if c['claim_id']=='CLAIM-030')
assert body==claim['statement_markdown'],'CLAIM-030 bytes mismatch evidence-inventory'
raw=open(os.path.join(r,'docs/_data/evidence-inventory.yml'),encoding='utf-8').read()
assert 'statement_markdown: |-' in raw.split('claim_id: CLAIM-030',1)[1].split('claim_id:',1)[0],'CLAIM-030 must use |- block scalar'
paths={'artifacts/bundle/v2/schema.json','artifacts/issue/v1/schema.json','artifacts/spec/v1/schema.json','artifacts/plan/v1/schema.json'}
seen={ref['locator']['path'] for ref in claim['evidence_refs'] if ref.get('kind')=='schema'}
assert paths<=seen,'CLAIM-030 schema refs'
PY
}
verify_reference_artifact_lifecycle_state_vocabulary_is_live() {
python3 - "$root" <<'PY'
import json,os,re,sys
r=sys.argv[1]
ref=open(os.path.join(r,'docs/reference.md'),encoding='utf-8',newline='').read().replace('\r\n','\n')
start=re.search(r'^#{1,6} .+ \{#artifact-lifecycle-and-closure\}\s*$',ref,re.M)
nxt=re.search(r'^#{1,6} .+ \{#[a-z0-9-]+\}\s*$',ref[start.end():],re.M)
section=ref[start.end():start.end()+(nxt.start() if nxt else len(ref))]
plan=json.load(open(os.path.join(r,'artifacts/plan/v1/schema.json')))
issue=json.load(open(os.path.join(r,'artifacts/issue/v1/schema.json')))
spec=json.load(open(os.path.join(r,'artifacts/spec/v1/schema.json')))
bundle=json.load(open(os.path.join(r,'artifacts/bundle/v2/schema.json')))
enums={'plan':set(plan['metadata']['properties']['status']['enum']),'issue':set(issue['nested_blocks']['issue']['properties']['status']['enum']),'spec':set(spec['metadata']['properties']['status']['enum']),'bundle':set(bundle['nested_blocks']['status']['properties']['maturity']['enum'])}
assert 'approved' not in section,'approved must be absent from lifecycle section'
plan=section.split('### Plan',1)[1]
assert all(x in plan for x in ('draft','ready','implementing','completed'))
assert 'in-progress' in section.split('### Issue',1)[1].split('### Spec',1)[0]
for name in ('Current Thinking','Draft Requirements','Draft Design Decisions','Spec Seeds','Version History'):
 assert name in section,'missing matureSections value '+name
def validate(section_text):
 if 'approved' in section_text: raise AssertionError('plan state approved is not live')
validate(section)
PY
  local tmp; tmp="$(corpus_copy)"
  python3 - "$tmp/docs/reference.md" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
open(p,'w').write(s.replace('Live: `draft` → `ready` → `implementing` → `completed`.','Live: `draft` → `approved` → `implementing` → `completed`.'))
PY
  if python3 - "$tmp/docs/reference.md" <<'PY' >/dev/null 2>&1
import sys,re
ref=open(sys.argv[1]).read()
start=re.search(r'^#{1,6} .+ \{#artifact-lifecycle-and-closure\}\s*$',ref,re.M)
nxt=re.search(r'^#{1,6} .+ \{#[a-z0-9-]+\}\s*$',ref[start.end():],re.M)
section=ref[start.end():start.end()+(nxt.start() if nxt else len(ref))]
if 'approved' not in section: raise SystemExit(2)
if 'approved' in section: raise SystemExit(1)
raise SystemExit(0)
PY
  then rm -rf "$tmp"; echo 'approved-plan-state mutation was accepted' >&2; exit 1; fi
  rm -rf "$tmp"
}
verify_reference_artifact_lifecycle_allowances_are_exact_bytes() {
  local tmp
  tmp="$(corpus_copy)"
  python3 - "$tmp/docs/reference.md" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read(); j=s.index('## Source traceability')
open(p,'w').write(s[:j]+'\nUnmarked lifecycle prose that should fail closed-world classification.\n\n'+s[j:])
PY
  run_verifier_expect_fail "$tmp" 'unclassified final-copy block'
  tmp="$(corpus_copy)"
  python3 - "$tmp/docs/reference.md" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
old='Feature work uses bundle → spec → plan. Bounded work uses issue → plan. Both tracks meet at a plan. Each spec has exactly one plan. An issue does not get a spec. Product code is not written until a plan is ready.'
new=old.replace('→','->',1)
open(p,'w').write(s.replace(old,new,1))
PY
  run_verifier_expect_fail "$tmp" 'unclassified final-copy block'
  tmp="$(corpus_copy)"
  python3 - "$tmp/docs/reference.md" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
old='<dd>bundle <code>ready</code> → spec <code>ready-for-implementation</code> → plan <code>ready</code> → plan <code>completed</code> → spec <code>implemented</code><br>bundle <code>delivered</code> is declared separately</dd>'
new=old.replace('bundle <code>ready</code>','bundle <code>ready</code> ')
open(p,'w').write(s.replace(old,new,1))
PY
  run_verifier_expect_fail "$tmp" 'unclassified final-copy block'
}
verify_reference_no_missing_route_links() {
python3 - "$root" <<'PY'
import os,re,sys
r=sys.argv[1]
ref=open(os.path.join(r,'docs/reference.md'),encoding='utf-8',newline='').read().replace('\r\n','\n')
site=open(os.path.join(r,'scripts/sitecheck/site.go'),encoding='utf-8').read()
routes=set(x.strip().strip('"') for x in re.search(r'func canonicalRoutes\(\) \[\]string \{\n\treturn \[\]string\{(.*?)\}',site,re.S).group(1).split(',') if x.strip().strip('"'))
start=re.search(r'^#{1,6} .+ \{#artifact-lifecycle-and-closure\}\s*$',ref,re.M)
nxt=re.search(r'^#{1,6} .+ \{#[a-z0-9-]+\}\s*$',ref[start.end():],re.M)
section=ref[start.end():start.end()+(nxt.start() if nxt else len(ref))]
for route in re.findall(r'\]\((/[^)#]+)',section):
 assert route in routes,f'missing route link {route}'
for route in re.findall(r'href="(/[^"#]+)"',section):
 assert route in routes,f'missing route href {route}'
for forbidden in ('/directive/','/adr/','/capability/'):
 assert forbidden not in section,f'forbidden route link {forbidden}'
def validate(section):
 for route in re.findall(r'\]\((/[^)#]+)',section):
  if route not in routes: raise AssertionError('missing route link '+route)
PY
  local tmp; tmp="$(corpus_copy)"
  python3 - "$tmp/docs/reference.md" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
needle='Directive, ADR, and Capability each carry their own status vocabularies in their artifact schemas. They are not paths into implementation.'
open(p,'w').write(s.replace(needle,needle+' See [Directive](/directive/).'))
PY
  if python3 - "$root" "$tmp/docs/reference.md" <<'PY' >/dev/null 2>&1
import os,re,sys
r,refp=sys.argv[1:3]
ref=open(refp).read(); site=open(os.path.join(r,'scripts/sitecheck/site.go')).read()
routes=set(x.strip().strip('"') for x in re.search(r'func canonicalRoutes\(\) \[\]string \{\n\treturn \[\]string\{(.*?)\}',site,re.S).group(1).split(',') if x.strip().strip('"'))
start=re.search(r'^#{1,6} .+ \{#artifact-lifecycle-and-closure\}\s*$',ref,re.M)
nxt=re.search(r'^#{1,6} .+ \{#[a-z0-9-]+\}\s*$',ref[start.end():],re.M)
section=ref[start.end():start.end()+(nxt.start() if nxt else len(ref))]
if '/directive/' not in section: raise SystemExit(2)
for route in re.findall(r'\]\((/[^)#]+)',section):
 if route not in routes: raise SystemExit(1)
raise SystemExit(0)
PY
  then rm -rf "$tmp"; echo 'missing-route-link mutation was accepted' >&2; exit 1; fi
  rm -rf "$tmp"
}
verify_reference_navigation_roster_is_consistent() {
python3 - "$root" <<'PY'
import os,re,sys,yaml
r=sys.argv[1]
top=yaml.safe_load(open(os.path.join(r,'docs/_data/content-topology.yml')))
site=open(os.path.join(r,'scripts/sitecheck/site.go'),encoding='utf-8').read()
header=open(os.path.join(r,'docs/_includes/site-header.html'),encoding='utf-8').read()
ref=open(os.path.join(r,'docs/reference.md'),encoding='utf-8').read()
assert os.path.isfile(os.path.join(r,'docs/reference.md'))
assert 'published: false' not in ref.split('---',2)[1]
assert 'redirect:' not in ref.split('---',2)[1]
primary=top['navigation']['primary']; utility=top['navigation']['utility']
def parse_nav(fn):
 m=re.search(r'func '+fn+r'\(\) \[\]string \{\n\treturn \[\]string\{(.*?)\}',site,re.S)
 return [x.strip().strip('"') for x in m.group(1).split(',')]
assert parse_nav('primaryNavigation')==primary
assert parse_nav('utilityNavigation')==utility
routes=set(x.strip().strip('"') for x in re.search(r'func canonicalRoutes\(\) \[\]string \{\n\treturn \[\]string\{(.*?)\}',site,re.S).group(1).split(',') if x.strip().strip('"'))
roster=set(primary+utility)
nav_blocks=[]
for label in ('Primary','Utility'):
 start=header.index('<nav aria-label="'+label+'">')
 end=header.index('</nav>',start)
 nav_blocks.append(header[start:end])
nav_html='\n'.join(nav_blocks)
for route in roster:
 assert nav_html.count('href="'+route+'"')==1,f'rostered nav anchor cardinality {route}'
for route in routes-roster:
 assert 'href="'+route+'"' not in nav_html,f'off-roster nav anchor {route}'
PY
  local tmp; tmp="$(corpus_copy)"
  python3 - "$tmp/docs/_includes/site-header.html" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
open(p,'w').write(s.replace('</nav>\n      <nav aria-label="Utility">','<a href="/reference/">Reference</a>\n      </nav>\n      <nav aria-label="Utility">',1))
PY
  if python3 - "$root" "$tmp/docs/_includes/site-header.html" <<'PY' >/dev/null 2>&1
import sys,yaml,os
r,hp=sys.argv[1:3]; top=yaml.safe_load(open(os.path.join(r,'docs/_data/content-topology.yml')))
header=open(hp).read(); roster=set(top['navigation']['primary']+top['navigation']['utility'])
if '/reference/' in roster: raise SystemExit(2)
if header.count('href="/reference/"')!=1: raise SystemExit(2)
raise SystemExit(1)
PY
  then rm -rf "$tmp"; echo 'off-roster nav mutation was accepted' >&2; exit 1; fi
  rm -rf "$tmp"
  tmp="$(corpus_copy)"
  python3 - "$tmp/docs/_includes/site-header.html" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
open(p,'w').write(s.replace('<a href="/evaluate/"','<!-- removed evaluate -->',1))
PY
  if python3 - "$root" "$tmp/docs/_includes/site-header.html" <<'PY' >/dev/null 2>&1
import sys,yaml,os
r,hp=sys.argv[1:3]; top=yaml.safe_load(open(os.path.join(r,'docs/_data/content-topology.yml')))
header=open(hp).read(); route=top['navigation']['primary'][0]
if 'href="'+route+'"' in header: raise SystemExit(2)
raise SystemExit(1)
PY
  then rm -rf "$tmp"; echo 'deleted-roster-nav mutation was accepted' >&2; exit 1; fi
  rm -rf "$tmp"
  tmp="$(corpus_copy)"
  python3 - "$tmp/docs/reference.md" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
open(p,'w').write(s.replace('---\n','---\npublished: false\n',1))
PY
  if python3 - "$tmp/docs/reference.md" <<'PY' >/dev/null 2>&1
import sys
ref=open(sys.argv[1]).read()
if 'published: false' not in ref.split('---',2)[1]: raise SystemExit(2)
raise SystemExit(1)
PY
  then rm -rf "$tmp"; echo 'published-false mutation was accepted' >&2; exit 1; fi
  rm -rf "$tmp"
}
validate_paper_ink() {
python3 - "$1" "$2" <<'PY'
import re,sys
layout=open(sys.argv[1]).read(); css=open(sys.argv[2]).read()
assert layout.index('{% assign presentation') < layout.index('<!doctype html>')
m=re.search(r'paper_kinds = "([^"]+)"',layout)
assert m and 'reference' in set(m.group(1).split(',')),'paper_kinds must contain reference'
assert '{% if paper_kinds contains page_kind %}' in layout
assert layout.index('/assets/css/site.css') < layout.index('/assets/css/backstop-tokens.css')
for sel in ('[data-page-kind="reference"] .state-index {','[data-page-kind="reference"] .state-coupling','[data-page-kind="reference"] .tactics-matrix'):
 assert sel in css,sel
assert ':focus-visible' in css and 'prefers-reduced-motion' in css
if re.search(r'#[0-9a-fA-F]{3,8}\b',css): raise SystemExit('raw hex literal')
for pat in ('rgb(','rgba(','hsl(','hsla('):
 if pat in css: raise SystemExit('raw color literal '+pat)
PY
}
verify_reference_paper_ink_chrome() {
  local layout="$root/docs/_layouts/default.html" css="$root/docs/assets/css/site.css" tmp
  validate_paper_ink "$layout" "$css"
  tmp="$(mktemp -d)"; cp "$layout" "$tmp/layout.html"; cp "$css" "$tmp/site.css"
  python3 - "$tmp/layout.html" <<'PY'
import re,sys
p=sys.argv[1]; s=open(p).read()
open(p,'w').write(re.sub(r'paper_kinds = "[^"]+"','paper_kinds = "evaluation,model,adoption,entity,extension,ecosystem"',s))
PY
  if validate_paper_ink "$tmp/layout.html" "$tmp/site.css" 2>/dev/null; then rm -rf "$tmp"; echo 'drop-reference mutation passed' >&2; exit 1; fi
  rm -rf "$tmp"
  tmp="$(mktemp -d)"; cp "$layout" "$tmp/layout.html"; cp "$css" "$tmp/site.css"
  sed -i 's/{% if paper_kinds contains page_kind %}//g; s/{% endif %}//g' "$tmp/layout.html"
  if validate_paper_ink "$tmp/layout.html" "$tmp/site.css" 2>/dev/null; then rm -rf "$tmp"; echo 'unguarded-tokens mutation passed' >&2; exit 1; fi
  rm -rf "$tmp"
  tmp="$(mktemp -d)"; cp "$layout" "$tmp/layout.html"; cp "$css" "$tmp/site.css"
  python3 - "$tmp/site.css" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
needle='[data-page-kind="reference"] .state-index {'
assert needle in s
start=s.index(needle)
end=s.index('[data-page-kind="reference"] .state-coupling', start)
open(p,'w').write(s[:start]+s[end:])
PY
  if validate_paper_ink "$tmp/layout.html" "$tmp/site.css" 2>/dev/null; then rm -rf "$tmp"; echo 'delete-state-index mutation passed' >&2; exit 1; fi
  rm -rf "$tmp"
  tmp="$(mktemp -d)"; cp "$layout" "$tmp/layout.html"; cp "$css" "$tmp/site.css"
  echo 'html { color: #ff0000; }' >> "$tmp/site.css"
  if validate_paper_ink "$tmp/layout.html" "$tmp/site.css" 2>/dev/null; then rm -rf "$tmp"; echo 'hex-literal mutation passed' >&2; exit 1; fi
  rm -rf "$tmp"
}
verify_reference_paper_ink_preserves_home_navigation() {
python3 - "$root/docs/assets/css/site.css" <<'PY'
import sys
css=open(sys.argv[1]).read()
assert '[data-page-kind="home"] .nav' in css
assert '[data-page-kind="home"] .nav-links' in css
PY
  local tmp; tmp="$(mktemp -d)"; cp "$root/docs/assets/css/site.css" "$tmp/site.css"
  python3 - "$tmp/site.css" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read()
for rule in ('[data-page-kind="home"] .nav','[data-page-kind="home"] .nav-links'):
 s=s.replace(rule,'')
open(p,'w').write(s)
PY
  if python3 - "$tmp/site.css" <<'PY' >/dev/null 2>&1
import sys
css=open(sys.argv[1]).read()
if '[data-page-kind="home"] .nav' not in css and '[data-page-kind="home"] .nav-links' not in css:
 raise SystemExit(1)
raise SystemExit(0)
PY
  then rm -rf "$tmp"; echo 'home-nav deletion mutation was accepted' >&2; exit 1; fi
  rm -rf "$tmp"
}
python3 - "$root" <<'PY'
import os,re,subprocess,sys,yaml
r=sys.argv[1]; top=yaml.safe_load(open(os.path.join(r,'docs/_data/content-topology.yml'))); ev=yaml.safe_load(open(os.path.join(r,'docs/_data/evidence-inventory.yml'))); presentation=yaml.safe_load(open(os.path.join(r,'docs/_data/site-presentation.yml')))
sources={'docs/pack/guide.md','docs/reference.md','docs/contributing.md'}; pages=[p for p in top['pages'] if p['source'] in sources]; assert {p['source'] for p in pages}==sources
texts={p['canonical_path']:open(os.path.join(r,p['source']),encoding='utf-8').read() for p in pages}
presented={p['route']:p['hero_question'] for p in presentation['pages']}
for p in pages:
 t=texts[p['canonical_path']]; assert t.count(p['hero_question'])==1,p['source']+' owning hero'; assert presented[p['canonical_path']]==p['hero_question'],p['source']+' presented hero'
 for anchor in p['required_blocks']: assert re.search(r'^#{1,6} .+ \{#'+re.escape(anchor)+r'\}$',t,re.M),p['source']+' '+anchor
for link in top['journey_links']:
 if link['source_route'] in texts: assert '<!-- backstop-journey-link: '+link['link_id']+' -->\n['+link['label']+']('+link['destination_route']+'#'+link['destination_anchor']+')' in texts[link['source_route']],link['link_id']
claims={c['claim_id']:c for c in ev['claims']}
visible=lambda text:text.replace('{% raw %}','').replace('{% endraw %}','')
for cid in ('CLAIM-006','CLAIM-007','CLAIM-008','CLAIM-009','CLAIM-012','CLAIM-013','CLAIM-014','CLAIM-015','CLAIM-016','CLAIM-021','CLAIM-026','CLAIM-027','CLAIM-028','CLAIM-029','CLAIM-030','CLAIM-031','CLAIM-032','CLAIM-036'):
 c=claims[cid]; assert '<!-- backstop-claim: '+cid+' -->\n'+c['statement_markdown'] in visible(texts[c['owner']['route']]),cid
assert '[adjacent guidance](/status/#adjacent-guidance)' in texts['/contributing/'],'external owner seam link'
responsibilities=[('/reference/','`backstop doctor` diagnoses configuration discovery'),('/reference/','Bundles progress through `idea`, `exploring`, `defined`, and `ready`'),('/reference/','`delivered_by` names a completed plan'),('/reference/','slash-bearing include or exclude pattern'),('/reference/','return `0` for success, `1` for blocking violations or broken promises, and `2` for configuration failure'),('/reference/','`backstop pack install`'),('/reference/','Artifact schemas live under'),('/reference/','initialization, diagnosis, gates, packs, artifacts, recipes, baselines, waivers, version reporting, and command discovery')]
def validate_responsibilities(corpus):
 for route,needle in responsibilities: assert needle in corpus[route],needle
validate_responsibilities(texts)
guide=texts['/pack/guide/']
assert guide.count('<div class="pack-model">')==1,'pack-model cardinality'
dl=re.search(r'<div class="pack-model">\s*<dl>(.*?)</dl>',guide,re.S)
assert dl,'pack-model dl'
terms=['Claim','Rule','Engine','Fixtures']
for term in terms:
 assert re.search(r'<dt>\s*'+re.escape(term)+r'\s*</dt>\s*<dd>[^<]+</dd>',dl.group(1)),term
pack_help=subprocess.check_output([os.path.join(r,'bin/backstop'),'pack','--help'],text=True)
pack_verbs=set(re.findall(r'^\s{2,}(\w+)\s',pack_help,re.M))
assert 'publish' not in pack_verbs,'pack publish must not exist'
rendered=set(re.findall(r'backstop pack (\w+)',guide))
assert rendered,'pack command probe must be non-empty on guide'
for verb in rendered:
 assert verb in pack_verbs,f'unknown pack verb on guide: {verb}'
PY
production_delete_reject(){
 local file="$1" needle="$2" expected="$3" tmp output; tmp="$(mktemp -d)"
 for d in docs artifacts pkg cmd bundles issues; do cp -R "$root/$d" "$tmp/$d"; done; cp "$root/README.md" "$root/backstop.yml" "$root/backstop.lock" "$tmp/"
 python3 - "$tmp/$file" "$needle" <<'PY'
import sys
p,n=sys.argv[1:]; s=open(p).read(); assert n in s; open(p,'w').write(s.replace(n,'',1))
PY
 if output="$(BACKSTOP_PUBLIC_MODEL_ROOT="$tmp" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$root" BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 "$root/scripts/verify-public-product-model.sh" 2>&1)"; then rm -rf "$tmp"; echo "$expected deletion passed production verifier" >&2; exit 1; fi
 grep -Fq "$expected" <<<"$output" || { rm -rf "$tmp"; echo "$expected deletion missing diagnostic: $output" >&2; exit 1; }
 rm -rf "$tmp"
}
production_delete_reject docs/reference.md '`backstop doctor` diagnoses configuration discovery' GET-004
production_delete_reject docs/reference.md 'Bundles progress through `idea`, `exploring`, `defined`, and `ready`' ART-003
production_delete_reject docs/reference.md '`delivered_by` names a completed plan' ART-004
production_delete_reject docs/reference.md 'slash-bearing include or exclude pattern' PACK-004
production_delete_reject docs/reference.md 'return `0` for success, `1` for blocking violations or broken promises, and `2` for configuration failure' CLI-001
production_delete_reject docs/reference.md '`backstop pack install`' CLI-003
production_delete_reject docs/reference.md 'Artifact schemas live under' CLI-004
production_delete_reject docs/reference.md 'initialization, diagnosis, gates, packs, artifacts, recipes, baselines, waivers, version reporting, and command discovery' CLI-005
tmp="$(mktemp -d)"; for d in docs artifacts pkg cmd bundles issues; do cp -R "$root/$d" "$tmp/$d"; done; cp "$root/README.md" "$root/backstop.yml" "$root/backstop.lock" "$tmp/"
python3 - "$tmp/docs/reference.md" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read(); open(p,'w').write(s+'\nInstalled packs execute their declared engines and the lock guarantees those exact bytes.\n')
PY
if BACKSTOP_PUBLIC_MODEL_ROOT="$tmp" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$root" BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 "$root/scripts/verify-public-product-model.sh" >/dev/null 2>&1; then rm -rf "$tmp"; echo unmarked-consequential-addition-passed >&2; exit 1; fi
rm -rf "$tmp"
verify_reference_artifact_lifecycle_machine
verify_reference_artifact_lifecycle_state_vocabulary_is_live
verify_reference_artifact_lifecycle_allowances_are_exact_bytes
verify_reference_no_missing_route_links
verify_reference_navigation_roster_is_consistent
verify_reference_paper_ink_chrome
verify_reference_paper_ink_preserves_home_navigation
