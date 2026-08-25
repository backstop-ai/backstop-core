#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
python3 - "$root" <<'PY'
import os,re,sys,yaml
r=sys.argv[1]; top=yaml.safe_load(open(os.path.join(r,'docs/_data/content-topology.yml'))); ev=yaml.safe_load(open(os.path.join(r,'docs/_data/evidence-inventory.yml')))
sources={'docs/index.md','docs/evaluate.md','docs/adopt.md','docs/status.md'}; pages=[p for p in top['pages'] if p['source'] in sources]
assert {p['source'] for p in pages}==sources
texts={p['canonical_path']:open(os.path.join(r,p['source']),encoding='utf-8').read() for p in pages}
for p in pages:
 t=texts[p['canonical_path']]; assert t.count(p['hero_question'])==2,p['source']+' hero'
 for anchor in p['required_blocks']: assert re.search(r'^#{1,6} .+ \{#'+re.escape(anchor)+r'\}$',t,re.M),p['source']+' '+anchor
for link in top['journey_links']:
 if link['source_route'] in texts: assert '<!-- backstop-journey-link: '+link['link_id']+' -->\n['+link['label']+']('+link['destination_route']+'#'+link['destination_anchor']+')' in texts[link['source_route']],link['link_id']
for ins in top['adoption_instructions']:
 assert ins['command_text'] in texts['/adopt/'],ins['instruction_id']
claims={c['claim_id']:c for c in ev['claims']}
for cid in ('CLAIM-001','CLAIM-002','CLAIM-003','CLAIM-004','CLAIM-006','CLAIM-007','CLAIM-008','CLAIM-011','CLAIM-012','CLAIM-017','CLAIM-018','CLAIM-020','CLAIM-021','CLAIM-024','CLAIM-033','CLAIM-034','CLAIM-035'):
 c=claims[cid]; assert '<!-- backstop-claim: '+cid+' -->\n'+c['statement_markdown'] in texts[c['owner']['route']],cid
status=texts['/status/']; exact='<!-- backstop-claim: CLAIM-005 -->\n'+claims['CLAIM-005']['statement_markdown'].replace('\n\n[Continue','\n\n<!-- backstop-journey-link: JLINK-024 -->\n[Continue')+'\n<!-- /backstop-claim -->'
assert status.count(exact)==1,'CLAIM-005/JLINK-024 exact layout'
responsibilities=[('/adopt/','GOBIN=./.backstop-bin go install'),('/adopt/','backstop init'),('/adopt/','backstop gate'),('/status/','Slash-bearing engine path patterns can fail open')]
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
production_delete_reject docs/adopt.md 'GOBIN=./.backstop-bin go install' ADOPT-INSTALL
production_delete_reject docs/adopt.md 'backstop init' ADOPT-CONFIGURE
production_delete_reject docs/adopt.md 'backstop gate' ADOPT-ENFORCE
production_delete_reject docs/status.md 'Slash-bearing engine path patterns can fail open' PACK-004
tmp="$(mktemp -d)"; for d in docs artifacts pkg cmd bundles issues; do cp -R "$root/$d" "$tmp/$d"; done; cp "$root/README.md" "$root/backstop.yml" "$root/backstop.lock" "$tmp/"
python3 - "$tmp/docs/evaluate.md" <<'PY'
import sys
p=sys.argv[1]; s=open(p).read(); open(p,'w').write(s+'\nInstalled packs execute their declared engines and the lock guarantees those exact bytes.\n')
PY
if BACKSTOP_PUBLIC_MODEL_ROOT="$tmp" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$root" BACKSTOP_PUBLIC_MODEL_SKIP_DISPATCH=1 "$root/scripts/verify-public-product-model.sh" >/dev/null 2>&1; then rm -rf "$tmp"; echo unmarked-consequential-addition-passed >&2; exit 1; fi
rm -rf "$tmp"
