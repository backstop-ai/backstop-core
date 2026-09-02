#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
python3 - "$root" <<'PY'
import os,re,sys,yaml
r=sys.argv[1]; top=yaml.safe_load(open(os.path.join(r,'docs/_data/content-topology.yml'))); ev=yaml.safe_load(open(os.path.join(r,'docs/_data/evidence-inventory.yml'))); presentation=yaml.safe_load(open(os.path.join(r,'docs/_data/site-presentation.yml')))
sources={'docs/extend.md','docs/reference.md','docs/contributing.md'}; pages=[p for p in top['pages'] if p['source'] in sources]; assert {p['source'] for p in pages}==sources
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
responsibilities=[('/reference/','`backstop doctor` diagnoses configuration discovery'),('/reference/','blocked waits on named work'),('/reference/','`delivered_by` names a completed plan'),('/extend/','slash-bearing include or exclude pattern'),('/reference/','return `0` for success, `1` for blocking violations or broken promises, and `2` for configuration failure'),('/reference/','`backstop pack install`'),('/reference/','Artifact schemas live under'),('/reference/','initialization, diagnosis, gates, packs, artifacts, recipes, baselines, waivers, version reporting, and command discovery')]
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
production_delete_reject docs/reference.md '`backstop doctor` diagnoses configuration discovery' GET-004
production_delete_reject docs/reference.md 'blocked waits on named work' ART-003
production_delete_reject docs/reference.md '`delivered_by` names a completed plan' ART-004
production_delete_reject docs/extend.md 'slash-bearing include or exclude pattern' PACK-004
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
