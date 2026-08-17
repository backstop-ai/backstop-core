---
name: reviving-dead-dispatch-adds-tool-dep
description: A plan that makes a dead dispatch guard live silently adds an external-tool dependency (ast-grep/semgrep/jq) to PRE-EXISTING unguarded tests that ran zero subprocesses before
metadata:
  type: project
---

When a plan widens a dispatch-eligibility guard so a previously-dead code path executes for
real, count the subprocesses the affected suites ran BEFORE and AFTER. Every pre-existing
test that transitively drives that path gains a hard external-tool requirement it never had,
and those tests carry no skip guard.

**Why:** PLAN-ISSUE-142 (2026-08-17) makes `pattern-arg` rules dispatch in `pkg/packval`
phase 3. Measured before: `./bin/backstop pack test packs/contracts` reports pass having run
ZERO subprocesses (all 7 rules are pattern-arg, all dead; the pack has no tool_config, no
scaffolds, no validators). After: 14 runs requiring `ast-grep` 0.43.0, `grep`, AND `jq`
(both of the pack's convert scripts pipe through jq). The plan's own sharp edges mandated
loud `t.Skipf` guards for ITS OWN tests, but `pkg/pack/distribution/contracts_local_install_test.go`,
`contracts_provisioning_test.go` and `cmd/backstop/gate_contract_e2e_test.go` reach the same
path through `distribution.Add` (validation is unconditional via `RunValidationOnScratchCopy`,
command.go:257) and hard-fail without those tools. CI happened to be covered
(`.github/workflows/ci.yml:72-76` installs semgrep 1.96.0 + ast-grep 0.43.0; jq ships on the
ubuntu runner) — but the plan neither stated the new requirement nor recorded checking.

**How to apply:**
1. Run the pre-fix command and count real subprocesses (`pack test <scratch copy>` — never
   the live tree, phase 3 renders `sample_config` into the pack dir).
2. Grep for every test reaching that path, including via `distribution.Add`/`Update`/`Upgrade`,
   which all validate unconditionally. `pack install` does NOT validate
   (`NewInstallCommand` takes no validator) — that distinction decides whether CI's fleet
   install step is exposed.
3. Read the convert scripts for their own tool deps — `jq` is easy to miss because it never
   appears in `pack.yml` or the allowlist.
4. Confirm CI installs each tool at the pinned version; if it does, the plan must still SAY
   the local-dev requirement changed.

Also worth pinning: `engine.CheckToolAllowed(allowlist, Provision.Tool, Provision.Version)`
compares the DECLARED version to the allowlist pin and NEVER reads the installed binary. So
"declaring the real installed version would be refused" is a truthfulness argument, not a
mechanical one — a testdata pack declaring `provision: {semgrep, 1.96.0}` passes on a box
running semgrep 1.156.0. `pkg/packval/testdata/rulepath-pack` is often miscited as "omits
provision": it declares no `engines:` block at all, so it inherits the BASE binding, which
DOES carry a provision.

Related: [[packval-real-execution-premises]], [[base-registry-binding-overrides-fixture]],
[[astgrep-pack-convert-script-scope]], [[selfpack-absence-claim-tests]].
