#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
python3 - "$root" <<'PY'
import os,re,sys,yaml
r=sys.argv[1]; top=yaml.safe_load(open(os.path.join(r,'docs/_data/content-topology.yml'))); model=yaml.safe_load(open(os.path.join(r,'docs/_data/product-model.yml'))); ev=yaml.safe_load(open(os.path.join(r,'docs/_data/evidence-inventory.yml'))); presentation=yaml.safe_load(open(os.path.join(r,'docs/_data/site-presentation.yml')))
sources={'docs/model.md','docs/use-cases.md','docs/pack/examples.md'}; pages=[p for p in top['pages'] if p['source'] in sources]; assert {p['source'] for p in pages}==sources
texts={p['canonical_path']:open(os.path.join(r,p['source']),encoding='utf-8').read() for p in pages}
presented={p['route']:p['hero_question'] for p in presentation['pages']}
for p in pages:
 t=texts[p['canonical_path']]; assert t.count(p['hero_question'])==1,p['source']+' owning hero'; assert presented[p['canonical_path']]==p['hero_question'],p['source']+' presented hero'
 for anchor in p['required_blocks']: assert re.search(r'^#{1,6} .+ \{#'+re.escape(anchor)+r'\}$',t,re.M),p['source']+' '+anchor
for concept in model['concepts']: assert concept['owner']['route']=='/model/' and '{#'+concept['owner']['anchor']+'}' in texts['/model/'],concept['concept_id']
for view in model['architecture_views']: assert view['diagram_source'] in texts['/model/'] and os.path.getsize(os.path.join(r,view['diagram_source']))>0,view['architecture_id']
for link in top['journey_links']:
 if link['source_route'] in texts: assert '<!-- backstop-journey-link: '+link['link_id']+' -->\n['+link['label']+']('+link['destination_route']+'#'+link['destination_anchor']+')' in texts[link['source_route']],link['link_id']
claims={c['claim_id']:c for c in ev['claims']}
for cid in ('CLAIM-011','CLAIM-019','CLAIM-020','CLAIM-022','CLAIM-023','CLAIM-025'):
 claim=claims[cid]; assert '<!-- backstop-claim: '+cid+' -->\n'+claim['statement_markdown'] in texts[claim['owner']['route']],cid
responsibilities=[('/model/','Terminal state records whether work was delivered'),('/model/','`delivered_by` or a direct typed artifact'),('/pack/examples/','pins the version in `backstop.yml` and writes `backstop.lock`')]
def validate_responsibilities(corpus):
 for route,needle in responsibilities: assert needle in corpus[route],needle
validate_responsibilities(texts)
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
production_delete_reject docs/model.md 'Terminal state records whether work was delivered' ART-003
production_delete_reject docs/model.md '`delivered_by` or a direct typed artifact' ART-004
production_delete_reject docs/pack/examples.md 'pins the version in `backstop.yml` and writes `backstop.lock`' CLAIM-025
tmp="$(mktemp -d)"; for d in docs artifacts pkg cmd bundles issues; do cp -R "$root/$d" "$tmp/$d"; done; cp "$root/README.md" "$root/backstop.yml" "$root/backstop.lock" "$tmp/"
python3 - "$tmp/docs/pack/examples.md" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read(); open(p,'w').write(s+'\nInstalled packs execute their declared engines and the lock guarantees those exact bytes.\n')
PY
if BACKSTOP_PUBLIC_MODEL_ROOT="$tmp" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$root" BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 "$root/scripts/verify-public-product-model.sh" >/dev/null 2>&1; then rm -rf "$tmp"; echo unmarked-consequential-addition-passed >&2; exit 1; fi
rm -rf "$tmp"
