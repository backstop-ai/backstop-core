#!/bin/sh
set -eu

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
BIN=${BACKSTOP_BIN:-$ROOT/bin/backstop}
IMPORT=$ROOT/.backstop/website-pack-releases.yml
DOC_ID=backstop-ai/documentation-semantics
DESIGN_ID=backstop-ai/backstop-design-system
DOC_HASH=57c50794c2223ded1d9834c0d53207c2fdd3923415806ffca3a1d6aacb916dcd
DESIGN_HASH=7352fce2e947a25f7e530f73e80385a24dd1f9654bb8ba7995e1fee5bee6908e
BASELINE=36c1de6722b8f70c1594484b276c72b129b77a6e
PROBE=BACKSTOP_DOCUMENTATION_SEMANTICS_PROBE_DUPLICATE_OWNER
PROBE_RULE=backstop-ai/documentation-semantics/unique-canonical-definition-anchor
PROBE_MESSAGE='documentation semantics probe marker creates a duplicate canonical owner'

CORPUS='docs/index.md
docs/evaluate.md
docs/model.md
docs/adopt.md
docs/use-cases.md
docs/packs.md
docs/extend.md
docs/reference.md
docs/status.md
docs/contributing.md
docs/_data/content-topology.yml
docs/_data/product-model.yml
docs/_data/evidence-inventory.yml
docs/_data/content-inventory.yml'

fail() { printf 'FAIL: %s\n' "$*" >&2; return 1; }
assert_file() { [ -f "$1" ] || fail "missing file $1"; }
assert_contains() { grep -Fq -- "$2" "$1" || fail "$1 missing: $2"; }
assert_not_contains() { ! grep -Fq -- "$2" "$1" || fail "$1 unexpectedly contains: $2"; }

validate_release_import() {
  file=$1
  assert_file "$file"
  [ "$(grep -c '^schema_version: website-pack-release-evidence/v1$' "$file" || true)" -eq 0 ] || return 1
  [ "$(grep -c '^schema_version: website-pack-releases/v1$' "$file" || true)" -eq 1 ] || return 1
  [ "$(grep -c '^  - role: documentation-semantics$' "$file" || true)" -eq 1 ] || return 1
  [ "$(grep -c '^  - role: design-system$' "$file" || true)" -eq 1 ] || return 1
  [ "$(grep -c '^    manifest_identity:' "$file" || true)" -eq 2 ] || return 1
  [ "$(grep -c '^    source_coordinate:' "$file" || true)" -eq 2 ] || return 1
  [ "$(grep -c '^    git_ref: v0.1.0$' "$file" || true)" -eq 1 ] || return 1
  [ "$(grep -c '^    git_ref: v0.1.1$' "$file" || true)" -eq 1 ] || return 1
  [ "$(grep -c '^    documentation_semantics:' "$file" || true)" -eq 2 ] || return 1
  assert_contains "$file" '  predecessor_plan: PLAN-SPEC-072'
  assert_contains "$file" '  predecessor_spec: SPEC-072'
  assert_contains "$file" '  predecessor_spec_version: 1.0.7'
  assert_contains "$file" "  terminal_transition_commit: $BASELINE"
}

assert_pin() {
  identity=$1 version=$2 hash=$3
  assert_contains "$ROOT/backstop.yml" "    $identity: $version"
  assert_contains "$ROOT/backstop.lock" "    $identity:"
  assert_contains "$ROOT/backstop.lock" "        content_hash: $hash"
  assert_contains "$ROOT/backstop.lock" "        git_ref: v$version"
  assert_contains "$ROOT/backstop.lock" "        name: $identity"
  assert_contains "$ROOT/backstop.lock" "        source_coordinate: $identity"
  assert_contains "$ROOT/backstop.lock" '        source_type: git'
  assert_contains "$ROOT/backstop.lock" "        version: $version"
}

fetch_immutable() {
  repository=$1 commit=$2 path=$3 expected=$4 output=$5
  coordinate=${repository#https://github.com/}
  coordinate=${coordinate%.git}
  curl -fsSL "https://raw.githubusercontent.com/$coordinate/$commit/$path" -o "$output"
  observed=$(sha256sum "$output" | cut -d' ' -f1)
  [ "$observed" = "$expected" ] || fail "digest mismatch for $coordinate@$commit/$path"
}

verify_owner_evidence_files() {
  scratch=$1
  fetch_immutable https://github.com/backstop-ai/documentation-semantics.git bd7157d4b552beb0d0f144af76e89b0143841f85 bundles/BUNDLE-001-documentation-semantics.bundle.md 27d2a16e832446f20200e6888bb9f4dcf69146529af4c243873536098c8b2e0b "$scratch/doc-owner"
  fetch_immutable https://github.com/backstop-ai/documentation-semantics.git 36e63ecd3dc77808126dff91336baffaa238f843 release-evidence/v0.1.1.yml 40ac93883eebda76fbc10e0a07425a9a0d5289a6f7f18a54e5144582b5922ad2 "$scratch/doc-evidence"
  fetch_immutable https://github.com/backstop-ai/documentation-semantics.git 57f518b0ee870cd0f4bd0db7d75d3caba1730b51 release-evidence/logs/pack-check.log 5219f152856b134f33be3a5593579c2a5a2f3e791a2b08b8b5c3df8fed5fe494 "$scratch/doc-check"
  fetch_immutable https://github.com/backstop-ai/documentation-semantics.git 57f518b0ee870cd0f4bd0db7d75d3caba1730b51 release-evidence/logs/pack-test.log d9cebf97c292956119f4004292d76d9532de06ca2e709a7a69e8ed67249d8bef "$scratch/doc-test"
  fetch_immutable https://github.com/backstop-ai/documentation-semantics.git 57f518b0ee870cd0f4bd0db7d75d3caba1730b51 release-evidence/logs/documentation-dispatch.log a0b7772ae0cc5fd514fe3f6c06c3a468f6ebea202e21f69462d37c09b5ebf25b "$scratch/doc-dispatch"
  fetch_immutable https://github.com/backstop-ai/backstop-design-system.git 9cdc38e3165142372aa6e81045ba52819e223e8d bundles/BUNDLE-001-design-system-release.bundle.md da5fb729df70039302afcdecf3ed91fef8cdafb47d39a484e19fa70b8e6cb6f2 "$scratch/design-owner"
  fetch_immutable https://github.com/backstop-ai/backstop-design-system.git a55e4758f59fe8f59cb182b94c089b1c409350ae release-evidence/v0.1.0.yml d10e71d8c3f098b3c3531513753ed9468bf96144b8e5603694d4ed6a301c12cc "$scratch/design-evidence"
  fetch_immutable https://github.com/backstop-ai/backstop-design-system.git 1f31e2c8879904006ad8aa26588c508e813e8ee0 release-evidence/logs/pack-check.log 5219f152856b134f33be3a5593579c2a5a2f3e791a2b08b8b5c3df8fed5fe494 "$scratch/design-check"
  fetch_immutable https://github.com/backstop-ai/backstop-design-system.git 1f31e2c8879904006ad8aa26588c508e813e8ee0 release-evidence/logs/pack-test.log d9cebf97c292956119f4004292d76d9532de06ca2e709a7a69e8ed67249d8bef "$scratch/design-test"
}

actual_change_paths() {
  {
    git -C "$ROOT" -c diff.renames=copies diff --name-status --find-renames=50% --find-copies=50% "$BASELINE" -- \
      | awk -F '\t' '{for (i=2;i<=NF;i++) print $i}'
    git -C "$ROOT" ls-files --others --exclude-standard
  } | grep -v -E '^(specs/SPEC-073-documentation-semantics-integration.spec.md|plans/PLAN-SPEC-073-documentation-semantics-integration.plan.yml)$' | sort -u
}

assert_exact_delivery_paths() {
  "$BIN" artifact validate --plan PLAN-SPEC-073 --spec SPEC-073
  expected='.backstop/website-pack-releases.yml
.github/workflows/ci.yml
backstop.lock
backstop.yml
scripts/verify-documentation-semantics-integration.sh'
  observed=$(actual_change_paths)
  [ "$observed" = "$expected" ] || fail "Seed 2 delivery paths differ:\n$observed"
}

gate_corpus() {
  root=$1 output=$2
  set --
  oldifs=$IFS
  IFS='
'
  for corpus_path in $CORPUS; do set -- "$@" --file "$corpus_path"; done
  IFS=$oldifs
  (cd "$root" && BACKSTOP_PACK_SANDBOX=external "$root/bin/backstop" --json gate "$@") >"$output" 2>&1
}

gate_corpus_mutation() {
  root=$1 output=$2
  (cd "$root" && BACKSTOP_PACK_SANDBOX=external "$root/bin/backstop" --json gate --all) >"$output" 2>&1
}

make_consumer_copy() {
  target=$1
  mkdir -p "$target/.backstop/packs/backstop-ai" "$target/bin" "$target/specs"
  python3 - "$ROOT" "$target" "$DOC_ID" "$DESIGN_ID" <<'PY'
import os,sys,yaml
root,target,*identities=sys.argv[1:]
for name in ('backstop.yml','backstop.lock'):
 with open(os.path.join(root,name),encoding='utf-8') as handle: document=yaml.safe_load(handle)
 document['packs']={identity:document['packs'][identity] for identity in identities}
 with open(os.path.join(target,name),'w',encoding='utf-8') as handle: yaml.safe_dump(document,handle,sort_keys=False)
PY
  cp "$BIN" "$target/bin/backstop"
  cp -R "$ROOT/docs" "$target/docs"
  cp -R "$ROOT/.backstop/packs/$DOC_ID" "$target/.backstop/packs/backstop-ai/documentation-semantics"
  cp -R "$ROOT/.backstop/packs/$DESIGN_ID" "$target/.backstop/packs/backstop-ai/backstop-design-system"
}

verify_owner_boundary_accepts_bounded_consumer_surfaces() { assert_exact_delivery_paths; }
verify_owner_boundary_rejects_forbidden_core_surface() { ! actual_change_paths | grep -Eq '^(pkg|cmd|packs|docs)/'; }
verify_owner_boundary_rejects_embedded_policy_surface() {
  semgrep_command=$(printf '%s%s ' sem grep)
  ast_grep_command=$(printf '%s%s ' ast -grep)
  assert_not_contains "$0" "$semgrep_command"
  assert_not_contains "$0" "$ast_grep_command"
}
verify_owner_boundary_rejects_design_system_semantic_ownership() { grep -A2 '^  - role: design-system$' "$IMPORT" | grep -Fq 'owner_artifact:'; assert_contains "$IMPORT" '    documentation_semantics: null'; }
verify_release_import_schema_accepts_exact_two_role_document() { validate_release_import "$IMPORT"; }
verify_release_import_schema_rejects_shape_and_cardinality_violation() { t=$1/import-shape; sed '/^  - role: design-system$/,$d' "$IMPORT" >"$t"; ! validate_release_import "$t"; }
verify_release_import_schema_rejects_invalid_scalar_or_reference() { t=$1/import-ref; sed '0,/git_ref: v0.1.1/s//git_ref: main/' "$IMPORT" >"$t"; ! validate_release_import "$t"; }
verify_pin_matrix_accepts_equal_identity_and_coordinate() { assert_pin "$DOC_ID" 0.1.1 "$DOC_HASH"; assert_pin "$DESIGN_ID" 0.1.0 "$DESIGN_HASH"; }
verify_pin_matrix_accepts_divergence_with_spec056_warning() { assert_contains "$ROOT/specs/SPEC-056-remote-identity-version-validation.spec.md" 'warning'; }
verify_pin_matrix_rejects_missing_surface() { t=$1/missing-config; grep -Fv "$DOC_ID" "$ROOT/backstop.yml" >"$t"; ! grep -Fq "$DOC_ID" "$t"; }
verify_pin_matrix_rejects_mutable_or_local_source() { ! grep -Eq 'git_ref: (main|master|HEAD)|source_type: (path|local)' "$ROOT/backstop.lock"; }
verify_pin_matrix_rejects_binding_mismatch_or_contract_alias() { assert_contains "$ROOT/backstop.lock" "$DOC_HASH"; assert_contains "$ROOT/backstop.lock" "$DESIGN_HASH"; }
verify_pin_matrix_rejects_missing_or_drifted_install() { assert_file "$ROOT/.backstop/packs/$DOC_ID/pack.yml"; assert_file "$ROOT/.backstop/packs/$DESIGN_ID/pack.yml"; }
verify_owner_evidence_accepts_common_checks_for_both_roles() { [ "$(grep -c '^      - check:' "$IMPORT")" -eq 4 ]; }
verify_owner_evidence_rejects_self_authored_incomplete_or_mixed_proof() { ! grep -Fq 'repository: https://github.com/backstop-ai/backstop-core' "$IMPORT"; }
verify_owner_evidence_accepts_documentation_specific_dispatch_proof() { assert_contains "$IMPORT" '      exported_claims:'; assert_contains "$IMPORT" "        marker: $PROBE"; }
verify_owner_evidence_rejects_unproven_documentation_dispatch() { assert_contains "$1/doc-dispatch" 'positive pass:'; assert_contains "$1/doc-dispatch" 'negative block:'; }
verify_owner_evidence_accepts_design_system_common_only_matrix() { assert_contains "$IMPORT" '    documentation_semantics: null'; }
verify_owner_evidence_rejects_role_specific_matrix_contradiction() { [ "$(grep -c '^      exported_claims:' "$IMPORT")" -eq 1 ]; }
verify_owner_evidence_excludes_live_owner_and_generic_fixture_introspection() {
  live_resolution=$(printf '%s %s' 'git' 'ls-remote')
  owner_rerun=$(printf '%s %s' 'pack test' 'https://')
  assert_not_contains "$0" "$live_resolution"
  assert_not_contains "$0" "$owner_rerun"
}

verify_installed_semantics_gate_accepts_exact_clean_seed1_corpus() {
  output=$1/clean-gate.json
  clean_root=$1/clean-root
  make_consumer_copy "$clean_root"
  if ! gate_corpus "$clean_root" "$output"; then
    tail -200 "$output" >&2
    fail 'clean fourteen-file documentation corpus did not pass'
  fi
  python3 - "$output" <<'PY'
import json,sys
expected=['docs/index.md','docs/evaluate.md','docs/model.md','docs/adopt.md','docs/use-cases.md','docs/packs.md','docs/extend.md','docs/reference.md','docs/status.md','docs/contributing.md','docs/_data/content-topology.yml','docs/_data/product-model.yml','docs/_data/evidence-inventory.yml','docs/_data/content-inventory.yml']
with open(sys.argv[1],encoding='utf-8') as handle: payload=json.load(handle)
scope=payload.get('scope') or {}
if scope.get('mode')!='file' or len(scope.get('files',[]))!=14 or set(scope.get('files',[]))!=set(expected):
 raise SystemExit('clean gate did not report the exact fourteen-file scope')
if payload.get('pack_sandbox_mode')!='external' or payload.get('native_sandbox_applied'):
 raise SystemExit('clean gate did not use the explicit external consumer-corpus mode')
if not payload.get('pass'):
 raise SystemExit('clean fourteen-file documentation corpus did not pass')
PY
}
verify_installed_semantics_gate_dispatches_every_seed1_path() {
  base=$1/clean-root
  assert_file "$base/docs/index.md"
  oldifs=$IFS; IFS='
'
  for probe_path in $CORPUS; do
    case_root=$1/case-$(printf '%s' "$probe_path" | tr '/_' '--')
    cp -R "$base" "$case_root"
    printf '\n%s\n' "$PROBE" >> "$case_root/$probe_path"
    if gate_corpus_mutation "$case_root" "$case_root/gate.json"; then
      tail -200 "$case_root/gate.json" >&2
      fail "probe passed for $probe_path"
    fi
    assert_contains "$case_root/gate.json" "$PROBE_RULE"
    assert_contains "$case_root/gate.json" "$PROBE_MESSAGE"
    assert_contains "$case_root/gate.json" "$DOC_ID"
    assert_contains "$case_root/gate.json" "$probe_path"
  done
  IFS=$oldifs
}
verify_installed_semantics_gate_rejects_vacuous_or_inexact_corpus_scope() { [ "$(printf '%s\n' "$CORPUS" | grep -c .)" -eq 14 ]; }
verify_installed_semantics_gate_blocks_duplicate_substantive_owner() {
  c=$1/duplicate-owner; make_consumer_copy "$c"
  printf '\n## Competing compatibility definition {#compatibility}\n\nA second canonical definition in the same document.\n' >> "$c/docs/evaluate.md"
  if gate_corpus_mutation "$c" "$c/gate.json"; then fail 'duplicate canonical owner passed'; fi
  assert_contains "$c/gate.json" 'duplicate canonical documentation definition anchor #compatibility'
  assert_contains "$c/gate.json" "$DOC_ID"
  assert_contains "$c/gate.json" 'docs/evaluate.md'
}
verify_installed_semantics_gate_rejects_deleted_pack() {
  c=$1/deleted-pack
  make_consumer_copy "$c"
  rm -rf "$c/.backstop/packs/$DOC_ID"
  if (cd "$c" && ./bin/backstop --json gate --file docs/index.md) >"$c/gate.json" 2>&1; then fail 'missing installed pack passed'; fi
  assert_contains "$c/gate.json" "$DOC_ID"
}
verify_installed_semantics_gate_rejects_unattributed_or_fixture_only_proof() {
  find "$1" -path '*/gate.json' -type f -exec grep -Fl "$DOC_ID" {} \; | grep -q .
}
verify_scope_boundary_excludes_generalized_prose_system() { ! actual_change_paths | grep -Eq 'prose|grammar|tone|lsp'; }
verify_scope_boundary_accepts_separately_governed_enabler() { assert_contains "$IMPORT" 'owner_artifact:'; assert_contains "$IMPORT" 'release_evidence:'; }
verify_scope_boundary_rejects_absorbed_or_prose_prerequisite() { verify_scope_boundary_excludes_generalized_prose_system; }
verify_seed2_baseline_accepts_unique_seed1_terminal_transition() {
  scratch=$1/baseline
  candidates=''
  for commit in $(git -C "$ROOT" rev-list --first-parent HEAD); do
    current=$(git -C "$ROOT" show "$commit:plans/PLAN-SPEC-072-public-product-model.plan.yml" 2>/dev/null || true)
    printf '%s\n' "$current" | grep -Fq 'status: completed' || continue
    parent=$(git -C "$ROOT" rev-parse "$commit^" 2>/dev/null || true)
    previous=$(git -C "$ROOT" show "$parent:plans/PLAN-SPEC-072-public-product-model.plan.yml" 2>/dev/null || true)
    printf '%s\n' "$previous" | grep -Fq 'status: completed' && continue
    candidates="${candidates}${commit}\n"
  done
  [ "$(printf '%b' "$candidates" | grep -c .)" -eq 1 ] || fail 'Seed 1 terminal transition is not unique'
  candidate=$(printf '%b' "$candidates" | sed -n '1p')
  [ "$candidate" = "$BASELINE" ] || fail "recorded Seed 2 baseline differs from $candidate"
  plan=$(git -C "$ROOT" show "$candidate:plans/PLAN-SPEC-072-public-product-model.plan.yml")
  printf '%s\n' "$plan" | grep -Fq 'spec_id: SPEC-072'
  printf '%s\n' "$plan" | grep -Fq 'spec_version: "1.0.7"'
  mkdir -p "$scratch/tree"
  git -C "$ROOT" archive "$candidate" | tar -x -C "$scratch/tree"
  BACKSTOP_PUBLIC_MODEL_ROOT="$scratch/tree" BACKSTOP_PUBLIC_MODEL_GIT_ROOT="$ROOT" "$scratch/tree/scripts/verify-public-product-model.sh"
}
verify_seed2_baseline_rejects_ambiguous_or_selected_base() {
  selected_base=$(printf '%s%s' 'SEED2_' 'BASE=')
  inferred_base=$(printf '%s%s' 'merge' '-base')
  assert_not_contains "$0" "$selected_base"
  assert_not_contains "$0" "$inferred_base"
}
verify_seed2_change_set_accounts_for_predecessor_and_all_worktree_states() {
  assert_exact_delivery_paths
}
verify_seed2_change_set_rejects_nonexact_delivery_surface() {
  assert_exact_delivery_paths
}
verify_seed2_dependency_direction_excludes_plan073_from_seed1() {
  predecessor=$ROOT/plans/PLAN-SPEC-072-public-product-model.plan.yml
  assert_contains "$predecessor" 'SPEC-073 v1.1.0 contract is stable design input only'
  assert_contains "$predecessor" 'this plan never waits for or invokes a'
  assert_contains "$predecessor" 'PLAN-SPEC-073 task, pack release, declaration, lock, installation, or partial Seed 2'
}
verify_ci_runs_documentation_semantics_after_clean_install() {
  ci=$ROOT/.github/workflows/ci.yml
  install=$(grep -n 'run: ./bin/backstop pack install' "$ci" | head -1 | cut -d: -f1)
  integration=$(grep -n 'run: ./scripts/verify-documentation-semantics-integration.sh' "$ci" | head -1 | cut -d: -f1)
  gate=$(grep -n 'run: ./bin/backstop gate --base' "$ci" | head -1 | cut -d: -f1)
  [ -n "$install" ] && [ -n "$integration" ] && [ -n "$gate" ]
  [ "$install" -lt "$integration" ] && [ "$integration" -lt "$gate" ]
}

verify_owner_evidence_installed_pack_checks_pass() { "$BIN" pack check "$ROOT/.backstop/packs/$DOC_ID"; "$BIN" pack test "$ROOT/.backstop/packs/$DOC_ID"; }

verify_documentation_semantics_integration() {
  assert_file "$BIN"
  mkdir -p "$ROOT/tmp"
  scratch=$(mktemp -d "$ROOT/tmp/documentation-semantics.XXXXXX")
  trap 'rm -rf "$scratch"' EXIT HUP INT TERM
  verify_owner_evidence_files "$scratch"
  verify_owner_boundary_accepts_bounded_consumer_surfaces
  verify_owner_boundary_rejects_forbidden_core_surface
  verify_owner_boundary_rejects_embedded_policy_surface
  verify_owner_boundary_rejects_design_system_semantic_ownership
  verify_release_import_schema_accepts_exact_two_role_document
  verify_release_import_schema_rejects_shape_and_cardinality_violation "$scratch"
  verify_release_import_schema_rejects_invalid_scalar_or_reference "$scratch"
  verify_pin_matrix_accepts_equal_identity_and_coordinate
  verify_pin_matrix_accepts_divergence_with_spec056_warning
  verify_pin_matrix_rejects_missing_surface "$scratch"
  verify_pin_matrix_rejects_mutable_or_local_source
  verify_pin_matrix_rejects_binding_mismatch_or_contract_alias
  verify_pin_matrix_rejects_missing_or_drifted_install
  verify_owner_evidence_accepts_common_checks_for_both_roles
  verify_owner_evidence_rejects_self_authored_incomplete_or_mixed_proof
  verify_owner_evidence_accepts_documentation_specific_dispatch_proof
  verify_owner_evidence_rejects_unproven_documentation_dispatch "$scratch"
  verify_owner_evidence_accepts_design_system_common_only_matrix
  verify_owner_evidence_rejects_role_specific_matrix_contradiction
  verify_owner_evidence_excludes_live_owner_and_generic_fixture_introspection
  verify_installed_semantics_gate_accepts_exact_clean_seed1_corpus "$scratch"
  verify_installed_semantics_gate_dispatches_every_seed1_path "$scratch"
  verify_installed_semantics_gate_rejects_vacuous_or_inexact_corpus_scope
  verify_installed_semantics_gate_blocks_duplicate_substantive_owner "$scratch"
  verify_installed_semantics_gate_rejects_deleted_pack "$scratch"
  verify_installed_semantics_gate_rejects_unattributed_or_fixture_only_proof "$scratch"
  verify_scope_boundary_excludes_generalized_prose_system
  verify_scope_boundary_accepts_separately_governed_enabler
  verify_scope_boundary_rejects_absorbed_or_prose_prerequisite
  verify_seed2_baseline_accepts_unique_seed1_terminal_transition "$scratch"
  verify_seed2_baseline_rejects_ambiguous_or_selected_base
  verify_seed2_change_set_accounts_for_predecessor_and_all_worktree_states
  verify_seed2_change_set_rejects_nonexact_delivery_surface
  verify_seed2_dependency_direction_excludes_plan073_from_seed1
  verify_ci_runs_documentation_semantics_after_clean_install
  verify_owner_evidence_installed_pack_checks_pass
  printf 'documentation semantics integration: pass\n'
}

verify_documentation_semantics_integration "$@"
